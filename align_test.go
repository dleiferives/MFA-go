package align

import (
	"os"
	"os/exec"
	"testing"
)

func TestConvertTo16kHz(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	tmp, _ := os.CreateTemp("", "test-*.wav")
	defer os.Remove(tmp.Name())
	tmp.Close()

	if err := convertTo16kHz([]byte("not audio"), tmp.Name()); err != nil {
		t.Logf("ffmpeg conversion failed (expected for non-audio data): %v", err)
	}
}

func TestParseMFAJSON(t *testing.T) {
	data := []byte(`{
		"start": 0,
		"end": 2.5,
		"tiers": {
			"words": {
				"type": "interval",
				"entries": [
					[0.1, 0.5, "hello"],
					[0.6, 1.0, "world"]
				]
			},
			"phones": {
				"type": "interval",
				"entries": [
					[0.1, 0.2, "h"],
					[0.2, 0.3, "ə"],
					[0.3, 0.5, "l"]
				]
			}
		}
	}`)

	r, err := parseMFAJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(r.Words))
	}
	if r.Words[0].Text != "hello" || r.Words[0].Start != 0.1 || r.Words[0].End != 0.5 {
		t.Fatalf("unexpected word alignment: %+v", r.Words[0])
	}
	if len(r.Phones) != 3 {
		t.Fatalf("expected 3 phones, got %d", len(r.Phones))
	}
	if r.Start != 0 || r.End != 2.5 {
		t.Fatalf("unexpected start/end: %f/%f", r.Start, r.End)
	}
}
