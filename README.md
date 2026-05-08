# mestt

`mestt` is a local-first speech-to-text CLI/TUI for terminal users.

It lets you record audio from the terminal, transcribe existing audio files, route transcripts to stdout, clipboard, or a file, and keep a local SQLite history.

An optional Fyne popup GUI is also available for a small record-to-clipboard window.

The project is inspired by tools like `ostt`, but is built idiomatically in Go with a small service-oriented architecture and a Bubble Tea recording UI.

## Status

`mestt` is early, usable MVP software.

Current features:

- record microphone audio from a terminal UI
- transcribe existing audio files
- OpenAI transcription backend
- local `whisper.cpp` transcription backend
- output to stdout, clipboard, or file
- local SQLite transcription history
- Linux and macOS-oriented command surface

Current limitations:

- local GPU support depends on how your `whisper.cpp` package was built
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

### Option 1: Install User-Local Binaries

Requirements:

- Go 1.26+
- `ffmpeg`

Clone and install:

```sh
git clone https://github.com/<your-org-or-user>/mestt.git
cd mestt
./scripts/install.sh
```

This installs:

- `mestt`
- `mestt-gui`
- `mesttd`

Default install location:

- `~/.local/bin`

Override the install directory if needed:

```sh
MESTT_INSTALL_DIR=/some/bin/dir ./scripts/install.sh
```

Verify:

```sh
mestt version
mesttd doctor
```

If `mestt` is not found afterward, add `~/.local/bin` to your `PATH`.

### Option 2: Use Nix

Build packages directly:

```sh
nix build .#mestt
nix build .#mestt-gui
```

Run apps directly:

```sh
nix run .#mestt -- version
nix run .#mestt-gui
```

Or use the dev shell and install user-local binaries:

```sh
nix develop
./scripts/install.sh
```

### Optional: Build the Fyne GUI

The GUI is behind the `fyne` build tag so normal CLI builds and tests do not require the Linux GUI stack.

```sh
nix develop
go build -tags fyne -o bin/mestt-gui ./cmd/mestt-gui
./bin/mestt-gui
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

If you use local transcription, you need `whisper-cli` available in `PATH`.

`mestt` will try to download the selected standard Whisper model into its data directory on first use.

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

### GUI Popup

Run the Fyne popup GUI:

```sh
go run -tags fyne ./cmd/mestt-gui
```

GUI controls:

- `Enter` from idle: prepare and start recording
- `Enter` while recording: stop, transcribe, and copy to clipboard
- `Esc`: cancel active work or dismiss copied/error state

On tiling window managers such as Niri, treat `mestt-record` as a small floating popup window.

### Trigger Helper

Run the popup trigger helper:

```sh
mesttd trigger
```

Inspect current trigger configuration:

```sh
mesttd doctor
```

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
model = "large-v3-turbo-q5_0"
timeout_seconds = 120
base_url = "https://api.openai.com/v1"

[local]
command = "whisper-cli"
download_command = "whisper-cpp-download-ggml-model"
model_path = ""
use_gpu = true

[output]
default_target = "stdout"

[daemon]
trigger_command = "mestt-gui"
trigger_args = []
```

Notes:

- `transcription.provider` must match the selected model type
- if `model_path` is empty, `mestt` uses `~/.local/share/mestt/models/ggml-<model>.bin`
- `download_command` is used to fetch the standard model if the file is missing
- set `use_gpu = false` to force CPU execution
- `daemon.trigger_command` controls what `mesttd trigger` launches

## Keybinding Integration

For MVP, global hotkey capture is intentionally left to your desktop or compositor. Bind your launcher key to:

```sh
mesttd trigger
```

See `docs/daemon.md` for an example Niri configuration.

## Local Whisper Notes

`mestt` currently uses `whisper.cpp` for on-device transcription.

Important behavior:

- the model is typically downloaded on first use
- first local transcription is usually slower than later runs
- `large-v3-turbo-q5_0` is the default built-in local model
- known built-in models are verified before use so partial downloads fail clearly
- GPU mode depends on how your `whisper.cpp` package was built

If local GPU mode is unstable or unavailable, switch to CPU mode in config:

```toml
[local]
use_gpu = false
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
use_gpu = false
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
