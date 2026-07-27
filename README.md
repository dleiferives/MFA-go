# MFA-go

MFA-go is a small Go wrapper around the [Montreal Forced Aligner (MFA)](https://montreal-forced-aligner.readthedocs.io/). It runs MFA as a subprocess and returns word- and phone-level timing information as Go values.

## Requirements

- Go 1.25 or newer
- `ffmpeg`
- Linux on `x86_64` for the bundled micromamba setup target

MFA itself and its models are installed into the local `mfa/` directory by the Makefile.

## Setup

Install `ffmpeg`, then run:

```sh
make setup
```

This downloads micromamba, creates an MFA environment, and downloads the English and Greek acoustic and dictionary models used by the setup targets. The first run can take a while.

To remove the downloaded environment and binary:

```sh
make clean
```

## Usage

`Provider.Align` expects mono, 44.1 kHz, signed 16-bit little-endian PCM audio. The audio is converted to 16 kHz mono WAV before MFA runs.

```go
package main

import (
	"context"
	"fmt"
	"os"

	align "github.com/dleiferives/MFA-go"
)

func main() {
	audio, err := os.ReadFile("utterance.pcm")
	if err != nil {
		panic(err)
	}

	provider := align.New(align.Config{
		MFAEnv:  "mfa/env",
		WorkDir: "mfa/work",
	})

	result, err := provider.Align(
		context.Background(),
		audio,
		"hello world",
		"english_mfa",     // acoustic model
		"english_us_arpa", // dictionary model
		"",                // optional G2P model path
	)
	if err != nil {
		panic(err)
	}

	for _, word := range result.Words {
		fmt.Printf("%s: %.3f-%.3f seconds\n", word.Text, word.Start, word.End)
	}
}
```

The package also exposes phone-level alignments through `Result.Phones`.

## Development

Run the tests and build with:

```sh
make test
make build
```

The conversion test is skipped when `ffmpeg` is not installed. The full alignment flow requires a configured MFA environment and downloaded models.

## License

No license has been selected for this project yet.
