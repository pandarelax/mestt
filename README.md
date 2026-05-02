# mestt

`mestt` is a local-first speech-to-text CLI/TUI for terminal users.

It lets you record audio from the terminal, transcribe existing audio files, route transcripts to stdout, clipboard, or a file, and keep a local SQLite history.

The project is inspired by tools like `ostt`, but is built idiomatically in Go with a small service-oriented architecture and a Bubble Tea recording UI.

## Status

`mestt` is early, usable MVP software.

Current features:

- record microphone audio from a terminal UI
- transcribe existing audio files
- OpenAI transcription backend
- local `faster-whisper` transcription backend
- output to stdout, clipboard, or file
- local SQLite transcription history
- Linux and macOS-oriented command surface

Current limitations:

- local GPU support depends on your `faster-whisper` / `ctranslate2` runtime build
- pause/resume recording is not implemented yet
- device listing and recording behavior are best-effort across platforms
- history is plain text output today

## Features

- `mestt` defaults to recording mode
- `mestt record` launches a Bubble Tea recording UI
- `mestt transcribe <file>` reuses the same transcription pipeline for existing audio files
- `mestt auth` configures either OpenAI or local transcription
- `mestt history` shows previous transcripts from SQLite
- `mestt list-devices` lists available input devices when supported by `ffmpeg`

## Install

### Option 1: Build with Go

Requirements:

- Go 1.26+
- `ffmpeg`

Clone and build:

```sh
git clone https://github.com/<your-org-or-user>/mestt.git
cd mestt
go build -o bin/mestt ./cmd/mestt
```

Run it:

```sh
./bin/mestt version
```

### Option 2: Use the Nix Dev Shell

This repository includes a `flake.nix` and `.envrc` for `nix-direnv`.

```sh
direnv allow
nix develop
go build -o bin/mestt ./cmd/mestt
./bin/mestt version
```

## Requirements

### Recording

`mestt` records audio through `ffmpeg`, so `ffmpeg` must be installed and available in `PATH`.

### Clipboard

Clipboard output requires:

- macOS: `pbcopy`
- Linux Wayland: `wl-copy`
- Linux X11: `xclip`

### OpenAI Backend

If you use OpenAI transcription, you need:

- an OpenAI Platform API key
- API billing/quota enabled on your OpenAI account

### Local Whisper Backend

If you use local transcription, you need a working Python runtime with `faster-whisper` available.

The repository currently supports local transcription through a small Python helper invoked by the Go application.

## Quick Start

### Local Transcription

Build the binary:

```sh
go build -o bin/mestt ./cmd/mestt
```

Configure the backend:

```sh
./bin/mestt auth
```

When prompted, choose one of the `[local]` models.

Recommended multilingual local model:

- `Local Whisper Large V3 Turbo`

Then transcribe a file:

```sh
./bin/mestt transcribe sample.wav
```

Or record directly:

```sh
./bin/mestt record
```

### OpenAI Transcription

Configure the backend:

```sh
./bin/mestt auth
```

When prompted, choose one of the OpenAI models and enter your API key.

Then use the same commands:

```sh
./bin/mestt transcribe sample.wav
./bin/mestt record
```

## Usage

### Commands

```sh
mestt
mestt record
mestt record -c
mestt record -o transcript.txt
mestt transcribe audio.wav
mestt transcribe audio.wav -c
mestt transcribe audio.wav -o transcript.txt
mestt auth
mestt history
mestt list-devices
mestt config
mestt version
```

### Recording Controls

In the recording UI:

- `Enter`: stop and transcribe
- `Esc`, `q`, `Ctrl+C`: cancel

### Output Modes

Default output is stdout.

Copy to clipboard:

```sh
mestt record -c
mestt transcribe sample.wav -c
```

Write to file:

```sh
mestt record -o transcript.txt
mestt transcribe sample.wav -o transcript.txt
```

### History

Show recent transcripts:

```sh
mestt history
mestt history --limit 50
```

### Device Listing

List audio devices:

```sh
mestt list-devices
```

## Configuration

Print the config path:

```sh
mestt config
```

Default config location:

- `~/.config/mestt/config.toml`

Example config:

```toml
[audio]
device = "default"
driver = ""
sample_rate = 16000
format = "wav"

[transcription]
provider = "local"
model = "large-v3-turbo"
timeout_seconds = 120
base_url = "https://api.openai.com/v1"

[local]
python_command = "python3"
device = "cpu"
compute_type = "int8"

[output]
default_target = "stdout"
```

Notes:

- `transcription.provider` must match the selected model type
- for local CPU mode, use:
  - `device = "cpu"`
  - `compute_type = "int8"`
- for local GPU mode, you may try:
  - `device = "cuda"`
  - `compute_type = "float16"`
  - but this requires a CUDA-enabled `ctranslate2` runtime

## Local Whisper Notes

`mestt` currently uses a local Python helper with `faster-whisper` for on-device transcription.

Important behavior:

- the model is typically downloaded on first use
- first local transcription is usually slower than later runs
- CPU mode is the safest default
- GPU mode depends on how `faster-whisper` and `ctranslate2` were installed

If local GPU mode fails with an error like:

```text
This CTranslate2 package was not compiled with CUDA support
```

switch to CPU mode in config:

```toml
[local]
device = "cpu"
compute_type = "int8"
```

## File Locations

- config: `~/.config/mestt/config.toml`
- data: `~/.local/share/mestt/`
- history database: `~/.local/share/mestt/history.sqlite3`
- secrets: `~/.local/share/mestt/credentials.json`
- logs: `~/.local/state/mestt/mestt.log`

## Architecture

High-level structure:

```text
cmd/mestt/
internal/cli/
internal/app/
internal/audio/
internal/transcribe/
internal/history/
internal/config/
internal/secret/
internal/output/
internal/tui/
```

The project is structured around:

- CLI command wiring in `internal/cli`
- application services in `internal/app`
- adapters for audio, transcription, history, output, and secrets
- Bubble Tea UI for recording

Additional design docs:

- `docs/architecture.md`
- `docs/mvp-plan.md`

## Development

### Run Tests

```sh
go test ./...
```

### Build

```sh
go build ./...
```

### Run Without Installing

```sh
go run ./cmd/mestt version
go run ./cmd/mestt auth
go run ./cmd/mestt record
```

## Troubleshooting

### `ffmpeg not found in PATH`

Install `ffmpeg` and ensure it is available in your shell.

### Local transcription fails on GPU

Try CPU mode:

```toml
[local]
device = "cpu"
compute_type = "int8"
```

### OpenAI transcription fails

Check:

- your API key is valid
- API billing/quota is enabled
- you selected an OpenAI model with `mestt auth`

### Recording does not work

Try:

```sh
mestt list-devices
```

Then update the configured audio device/driver if needed.

## Roadmap

Planned or likely future work:

- better local GPU setup story
- richer history UI
- pause/resume recording
- retry/replay commands
- more transcription providers
- better retained recording management

## Contributing

Issues and pull requests are welcome.

If you plan to contribute a larger change, open an issue first so the design and scope can be discussed before implementation.

## License

MIT
