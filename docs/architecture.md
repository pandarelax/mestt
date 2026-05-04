# mestt Architecture

## Purpose

`mestt` is a local-first speech-to-text CLI/TUI written in Go. The current MVP records audio from the terminal, transcribes it with OpenAI or a local whisper.cpp backend, and sends the result to stdout, clipboard, file output, and local history.

This document defines the intended architecture before implementation.

## Product Constraints

- Binary name: `mestt`
- Platforms: Linux and macOS
- Initial providers: OpenAI and local whisper.cpp
- History storage: SQLite
- Recording backend: `ffmpeg`
- UI: Bubble Tea
- Scope: single local CLI/TUI application only

## Design Principles

- Keep the first version small and end-to-end complete.
- Prefer plain Go composition over frameworks.
- Put business behavior in testable services, not in Cobra commands or Bubble Tea models.
- Define interfaces at the consumer boundary and keep them small.
- Use `context.Context` for cancellation and timeouts.
- Treat external systems such as `ffmpeg`, OpenAI, SQLite, clipboard, and filesystem as adapters.
- Avoid speculative abstractions for future daemon or streaming modes, but leave clean seams for them.

## Architectural Shape

The recommended shape is ports and adapters with a thin application layer.

Layers:

1. `cmd/mestt`
2. `internal/cli`
3. `internal/app`
4. `internal/{audio,transcribe,history,config,secret,output,tui}`
5. external systems: `ffmpeg`, OpenAI HTTP API, SQLite, OS clipboard, filesystem

Dependency direction:

- `cmd/mestt` depends on `internal/cli`
- `internal/cli` depends on `internal/app` and concrete adapters
- `internal/app` depends on interfaces only
- adapters implement interfaces required by `internal/app`
- Bubble Tea models call application services but do not own provider or persistence logic

## Recommended Package Layout

```text
cmd/mestt/main.go

internal/cli/
  root.go
  record.go
  transcribe.go
  auth.go
  history.go
  config.go
  devices.go

internal/app/
  record.go
  transcribe.go
  auth.go
  history.go

internal/audio/
  recorder.go
  ffmpeg.go
  devices.go
  level.go
  wav.go

internal/tui/
  record/
    model.go
    view.go
    messages.go
  history/
    model.go
  auth/
    model.go

internal/transcribe/
  provider.go
  model.go
  registry.go
  openai.go

internal/history/
  store.go
  sqlite.go
  migrations.go

internal/config/
  config.go
  store.go
  defaults.go

internal/secret/
  store.go
  file.go

internal/output/
  output.go
  clipboard.go

internal/paths/
  paths.go

internal/logging/
  logging.go
```

## Command Surface

Initial commands:

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

Behavior:

- `mestt` defaults to `record`
- `record` launches the recording TUI
- `transcribe` reuses the same transcription pipeline for an existing audio file
- `auth` configures model selection and API key storage
- `history` shows saved transcriptions
- `list-devices` verifies available input devices

Deferred commands:

- `retry`
- `replay`
- `keywords`
- `logs`

## Application Services

Application services should own use-case behavior.

Recommended services:

- `RecordService`
- `TranscribeService`
- `AuthService`
- `HistoryService`

Example interface boundaries:

```go
type Recorder interface {
	Start(ctx context.Context, opts RecordOptions) (RecordingSession, error)
	ListDevices(ctx context.Context) ([]Device, error)
}

type RecordingSession interface {
	Events() <-chan AudioEvent
	Pause() error
	Resume() error
	Stop(ctx context.Context) (Recording, error)
	Cancel() error
}

type Transcriber interface {
	Transcribe(ctx context.Context, req TranscribeRequest) (TranscribeResult, error)
}

type HistoryStore interface {
	Save(ctx context.Context, entry Entry) error
	List(ctx context.Context, limit int) ([]Entry, error)
}

type SecretStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

type OutputWriter interface {
	Write(ctx context.Context, text string, target Target) error
}
```

These should live near the consuming service, not necessarily in a shared interfaces package.

## Audio Architecture

Recording should use `ffmpeg` as the backend process rather than a native audio library in the first version.

Recommended split:

- `internal/audio/ffmpeg.go` manages process invocation and platform-specific device syntax
- `internal/audio/recorder.go` exposes the Go-facing recording API
- `internal/audio/level.go` computes volume/peak values from streamed PCM sample windows

Important notes:

- Linux and macOS require different `ffmpeg` input device arguments.
- Device enumeration may need OS-specific adapter code or shelling out to system tools where `ffmpeg` alone is insufficient.
- MVP should prefer uncompressed intermediate recording such as WAV for simplicity, then submit that file to OpenAI.
- TUI metering can be driven by periodic sample windows parsed from the recording stream or from a tee'd PCM stream.

Recommended MVP simplification:

- Record to temporary WAV.
- Parse enough audio data during recording to show duration and level meter.
- Defer advanced waveform/spectrum rendering unless it remains straightforward.

## Transcription Architecture

The first version supports OpenAI and a local whisper.cpp adapter, while keeping the package shape open for more providers later.

Recommended model:

- `ProviderID` and `ModelID` as string-backed named types
- a small registry for supported models and defaults
- one OpenAI client implementing `Transcriber`

OpenAI scope:

- support `gpt-4o-transcribe`, `gpt-4o-mini-transcribe`, and optionally `whisper-1`
- multipart upload over `net/http`
- request timeout configured in app config
- clean error wrapping with provider context

Local whisper.cpp scope:

- invoke a small Python helper from Go rather than embedding model inference directly in-process
- support CPU-safe defaults first, with the runtime environment free to enable GPU later
- keep the helper interface narrow: audio path in, transcript JSON out

Do not build a provider plugin system. A simple registry map and adapters are enough.

## Configuration And Secrets

Use XDG-style paths.

Recommended locations:

- config: `~/.config/mestt/config.toml`
- data: `~/.local/share/mestt/`
- state: `~/.local/state/mestt/`

Recommended config responsibilities:

- selected provider/model
- audio defaults
- output defaults
- OpenAI request settings

Keep API keys separate from config.

Recommended secret storage:

- `~/.local/share/mestt/credentials.json` with `0600` permissions for MVP

This is simple, testable, and replaceable later if keychain integration is added.

## History Storage

Use SQLite from the start.

Recommended schema for MVP:

```sql
CREATE TABLE history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  source_path TEXT,
  model_id TEXT NOT NULL,
  transcript TEXT NOT NULL
);
```

Notes:

- `source_kind` can be `recording` or `file`
- `source_path` is useful for future retry/replay features
- keep schema minimal until a concrete need appears

## Output Routing

Output routing should be centralized so both `record` and `transcribe` use the same behavior.

Targets:

- stdout
- clipboard
- file

Recommended rule:

- exactly one explicit target per invocation
- default to stdout when no explicit target is set

## TUI Architecture

Bubble Tea models should manage interaction state, not external business logic.

Recording TUI state should include:

- current status: idle, recording, paused, transcribing, done, failed
- elapsed duration
- current level and recent peak
- last error message
- explicit output target summary

Recording TUI commands/messages:

- tick for timer refresh
- audio event updates
- keypresses for pause/resume, stop, cancel
- transcription completion or failure

The TUI should call service methods through a small controller or injected callbacks rather than importing low-level providers directly.

## Error Handling

Recommended approach:

- return wrapped errors with context using `fmt.Errorf("...: %w", err)`
- keep sentinel errors for user-facing branches only when necessary
- map low-level errors into concise user-facing CLI/TUI messages near the edge
- preserve original errors for logs and tests

## Logging

Use `log/slog`.

Recommended logging behavior:

- default structured logs to state directory file
- optional debug level via environment variable or config
- avoid logging transcript contents by default
- log provider/model, durations, and file paths where helpful

## Dependency Recommendations

- CLI: `spf13/cobra`
- TUI: `charmbracelet/bubbletea`
- styling: `charmbracelet/lipgloss`
- config: `pelletier/go-toml/v2`
- XDG paths: `adrg/xdg`
- SQLite: `modernc.org/sqlite`
- logging: stdlib `log/slog`
- HTTP: stdlib `net/http`

This set is enough for the MVP without adding heavy frameworks.

## Testing Strategy

Primary rule: application services should be testable without a real terminal, real `ffmpeg`, or real OpenAI calls.

Test split:

- unit tests for app services with fakes
- unit tests for config/path/history logic with temp dirs and in-memory DBs
- provider tests using `httptest.Server`
- TUI update tests for key state transitions
- integration tests for real process execution behind build tags or explicit opt-in

Recommended verification targets:

- `go test ./...`
- `go test -race ./...`

## Deliberate Non-Goals In Code Structure

Do not add now:

- background daemon abstractions
- internal event bus
- generic repository base types
- provider plugin loading
- full domain-driven design package layering
- multiple storage backends for the same data unless needed

The architecture should be clean, but still sized for the MVP.
