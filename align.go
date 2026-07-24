// Package align wraps Montreal Forced Aligner via subprocess.
//
// MFA is called via exec.Command using a local micromamba environment.
// The caller is responsible for setting up the environment (mfa/bootstrap.sh)
// before using this package.
package align

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Provider struct {
	MFAEnv  string
	WorkDir string
	BinDir  string
}

type Config struct {
	MFAEnv  string // path to micromamba environment (e.g., mfa/env)
	WorkDir string // MFA working directory (e.g., mfa/work)
	BinDir  string // path containing mfa/micromamba binaries (e.g., mfa/bin)
}

func New(cfg Config) *Provider {
	return &Provider{
		MFAEnv:  cfg.MFAEnv,
		WorkDir: cfg.WorkDir,
		BinDir:  cfg.BinDir,
	}
}

type WordAlignment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type PhoneAlignment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Phone string  `json:"phone"`
}

type Result struct {
	Words  []WordAlignment  `json:"words"`
	Phones []PhoneAlignment `json:"phones"`
	Start  float64          `json:"start"`
	End    float64          `json:"end"`
}

// Align runs forced alignment on the given audio and transcript using the
// specified MFA model names (acoustic, dictionary, optional G2P).
func (p *Provider) Align(ctx context.Context, audio []byte, transcript, acousticModel, dictionaryModel, g2pModel string) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "mfa-align-")
	if err != nil {
		return nil, fmt.Errorf("align: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	corpusDir := filepath.Join(tmpDir, "corpus")
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(corpusDir, 0755)
	os.MkdirAll(outputDir, 0755)

	// Write audio as 16kHz mono WAV
	wavPath := filepath.Join(corpusDir, "utterance.wav")
	if err := convertTo16kHz(audio, wavPath); err != nil {
		return nil, fmt.Errorf("align: %w", err)
	}

	// Write transcript
	labPath := wavPath[:len(wavPath)-4] + ".lab"
	if err := os.WriteFile(labPath, []byte(transcript), 0644); err != nil {
		return nil, fmt.Errorf("align: %w", err)
	}

	// Build MFA command
	mfaBin := filepath.Join(p.MFAEnv, "bin", "mfa")
	args := []string{
		"align", corpusDir,
		acousticModel, dictionaryModel, outputDir,
		"--output_format", "json",
		"--single_speaker",
		"--no_use_mp",
		"--clean",
		fmt.Sprintf("--temporary_directory=%s", filepath.Join(tmpDir, "mfa-tmp")),
	}
	if g2pModel != "" {
		args = append(args, "--g2p_model_path", g2pModel)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, mfaBin, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MFA_ROOT_DIR=%s", p.WorkDir),
		fmt.Sprintf("PATH=%s/bin:%s", p.MFAEnv, os.Getenv("PATH")),
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("align: mfa failed: %w (took %v)", err, time.Since(start))
	}

	// Find and parse output JSON
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("align: %w", err)
	}

	var r *Result
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			data, err := os.ReadFile(filepath.Join(outputDir, e.Name()))
			if err != nil {
				return nil, fmt.Errorf("align: %w", err)
			}
			r, err = parseMFAJSON(data)
			if err != nil {
				return nil, fmt.Errorf("align: %w", err)
			}
			break
		}
	}
	if r == nil {
		return nil, fmt.Errorf("align: no alignment JSON found in output")
	}
	return r, nil
}

func parseMFAJSON(data []byte) (*Result, error) {
	var raw struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Tiers map[string]struct {
			Type    string     `json:"type"`
			Entries [][3]any   `json:"entries"`
		} `json:"tiers"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	r := &Result{Start: raw.Start, End: raw.End}

	if words, ok := raw.Tiers["words"]; ok {
		for _, entry := range words.Entries {
			start, _ := entry[0].(float64)
			end, _ := entry[1].(float64)
			text, _ := entry[2].(string)
			r.Words = append(r.Words, WordAlignment{Start: start, End: end, Text: text})
		}
	}

	if phones, ok := raw.Tiers["phones"]; ok {
		for _, entry := range phones.Entries {
			start, _ := entry[0].(float64)
			end, _ := entry[1].(float64)
			phone, _ := entry[2].(string)
			r.Phones = append(r.Phones, PhoneAlignment{Start: start, End: end, Phone: phone})
		}
	}

	return r, nil
}

func convertTo16kHz(audio []byte, outPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1",
		"-i", "pipe:0",
		"-ar", "16000", "-ac", "1",
		outPath,
	)
	cmd.Stdin = strings.NewReader(string(audio))
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
