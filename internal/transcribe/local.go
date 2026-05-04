package transcribe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pandarelax/mestt/internal/paths"
)

type LocalRequest struct {
	AudioPath       string
	Model           Model
	Command         string
	DownloadCommand string
	ModelPath       string
	Language        string
	UseGPU          bool
}

type LocalClient struct{}

type localResponse struct {
	Transcription []localSegment `json:"transcription"`
}

type localSegment struct {
	Text string `json:"text"`
}

type knownModel struct {
	Size   int64
	SHA256 string
}

var knownModelMetadata = map[string]knownModel{
	string(ModelLargeV3TurboLocal): {
		Size:   1624555275,
		SHA256: "1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69",
	},
	string(DefaultLocalModelID): {
		Size:   574041195,
		SHA256: "394221709cd5ad1f40c46e6031ca61bce88931e6e088c188294c6d5a55ffa7e2",
	},
}

func (c LocalClient) Prepare(ctx context.Context, req LocalRequest) error {
	if _, err := resolveExecutablePath(req.Command, "whisper-cli"); err != nil {
		return err
	}
	_, err := resolveLocalModelPath(ctx, req)
	return err
}

func (c LocalClient) Transcribe(ctx context.Context, req LocalRequest) (Result, error) {
	command, err := resolveExecutablePath(req.Command, "whisper-cli")
	if err != nil {
		return Result{}, err
	}
	modelPath, err := resolveLocalModelPath(ctx, req)
	if err != nil {
		return Result{}, err
	}

	outputDir, err := os.MkdirTemp("", "mestt-whispercpp-*")
	if err != nil {
		return Result{}, fmt.Errorf("create whisper.cpp temp directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	outputBase := filepath.Join(outputDir, "transcript")
	args := []string{
		"-m", modelPath,
		"-f", req.AudioPath,
		"-oj",
		"-of", outputBase,
		"-np",
		"-nt",
		"-bs", "1",
		"-l", defaultString(strings.TrimSpace(req.Language), "auto"),
	}
	if !req.UseGPU {
		args = append(args, "-ng")
	}

	cmd := exec.CommandContext(ctx, command, args...)
	var stderr bytes.Buffer
	cmd.Stdout = ioDiscard{}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return Result{}, fmt.Errorf("run whisper.cpp transcription: %w: %s", err, stderrText)
		}
		return Result{}, fmt.Errorf("run whisper.cpp transcription: %w", err)
	}

	data, err := os.ReadFile(outputBase + ".json")
	if err != nil {
		return Result{}, fmt.Errorf("read whisper.cpp json output: %w", err)
	}

	var parsed localResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return Result{}, fmt.Errorf("parse whisper.cpp json output: %w", err)
	}

	var text strings.Builder
	for _, segment := range parsed.Transcription {
		text.WriteString(segment.Text)
	}

	return Result{Text: strings.TrimSpace(text.String()), ModelID: string(req.Model.ID), Provider: string(req.Model.Provider)}, nil
}

func resolveLocalModelPath(ctx context.Context, req LocalRequest) (string, error) {
	if modelPath := strings.TrimSpace(req.ModelPath); modelPath != "" {
		if _, err := os.Stat(modelPath); err != nil {
			return "", fmt.Errorf("stat configured local model path: %w", err)
		}
		if err := validateKnownModelFile(req.Model.APIName, modelPath); err != nil {
			return "", err
		}
		return modelPath, nil
	}

	p := paths.Resolve()
	if err := p.Ensure(); err != nil {
		return "", fmt.Errorf("ensure local data directories: %w", err)
	}

	modelDir := filepath.Join(p.DataDir, "models")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		return "", fmt.Errorf("create local model directory: %w", err)
	}
	cleanupStaleDownloadDirs(modelDir)

	modelPath := filepath.Join(modelDir, "ggml-"+req.Model.APIName+".bin")
	if _, err := os.Stat(modelPath); err == nil {
		if err := validateKnownModelFile(req.Model.APIName, modelPath); err != nil {
			if removeErr := os.Remove(modelPath); removeErr != nil {
				return "", fmt.Errorf("remove invalid local model file: %w", removeErr)
			}
		} else {
			return modelPath, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("stat local model file: %w", err)
	}

	if err := downloadKnownModel(ctx, req.DownloadCommand, req.Model.APIName, modelDir); err != nil {
		return "", err
	}

	if err := validateKnownModelFile(req.Model.APIName, modelPath); err != nil {
		return "", err
	}
	return modelPath, nil
}

func cleanupStaleDownloadDirs(modelDir string) {
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "download-") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(modelDir, entry.Name()))
	}
}

func downloadKnownModel(ctx context.Context, downloadCommand, modelName, modelDir string) error {
	downloadDir, err := os.MkdirTemp(modelDir, "download-*")
	if err != nil {
		return fmt.Errorf("create local model download directory: %w", err)
	}
	defer os.RemoveAll(downloadDir)

	downloadCommand, err = resolveExecutablePath(downloadCommand, "whisper-cpp-download-ggml-model")
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, downloadCommand, modelName, downloadDir)
	var stderr bytes.Buffer
	cmd.Stdout = ioDiscard{}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if stderrText != "" {
			return fmt.Errorf("download whisper.cpp model %q: %w: %s", modelName, err, stderrText)
		}
		return fmt.Errorf("download whisper.cpp model %q: %w", modelName, err)
	}

	sourcePath := filepath.Join(downloadDir, "ggml-"+modelName+".bin")
	if err := validateKnownModelFile(modelName, sourcePath); err != nil {
		return err
	}
	targetPath := filepath.Join(modelDir, filepath.Base(sourcePath))
	if err := os.Rename(sourcePath, targetPath); err != nil {
		return fmt.Errorf("move downloaded local model file into place: %w", err)
	}
	return nil
}

func validateKnownModelFile(modelName, modelPath string) error {
	meta, ok := knownModelMetadata[modelName]
	if !ok {
		return nil
	}
	return validateModelFile(modelPath, meta)
}

func validateModelFile(modelPath string, meta knownModel) error {
	info, err := os.Stat(modelPath)
	if err != nil {
		return fmt.Errorf("stat local model file: %w", err)
	}
	if info.Size() != meta.Size {
		return fmt.Errorf("local model file is incomplete or corrupt: %s (expected %d bytes, got %d)", modelPath, meta.Size, info.Size())
	}

	file, err := os.Open(modelPath)
	if err != nil {
		return fmt.Errorf("open local model file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash local model file: %w", err)
	}
	actual := fmt.Sprintf("%x", hasher.Sum(nil))
	if actual != meta.SHA256 {
		return fmt.Errorf("local model file is incomplete or corrupt: %s (expected sha256 %s, got %s)", modelPath, meta.SHA256, actual)
	}
	return nil
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func resolveExecutablePath(value, fallback string) (string, error) {
	command := defaultString(value, fallback)
	path, err := exec.LookPath(command)
	if err == nil {
		return path, nil
	}
	if fallback == command {
		return "", fmt.Errorf("%s not found in PATH; run inside 'nix develop' or set [local].%s to an absolute command path", command, localCommandConfigKey(command))
	}
	return "", fmt.Errorf("configured local command %q not found in PATH; set [local].%s to a valid absolute path or run inside 'nix develop'", command, localCommandConfigKey(fallback))
}

func localCommandConfigKey(command string) string {
	if command == "whisper-cpp-download-ggml-model" {
		return "download_command"
	}
	return "command"
}
