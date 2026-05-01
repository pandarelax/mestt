package transcribe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAIClientTranscribe(t *testing.T) {
	tempDir := t.TempDir()
	audioPath := filepath.Join(tempDir, "sample.wav")
	if err := os.WriteFile(audioPath, []byte("fake-audio"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization header = %q", got)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("model"); got != "whisper-1" {
			t.Fatalf("model form value = %q", got)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile() error = %v", err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if string(data) != "fake-audio" {
			t.Fatalf("uploaded file = %q", string(data))
		}
		_, _ = w.Write([]byte(`{"text":"hello from test"}`))
	}))
	defer server.Close()

	client := OpenAIClient{}
	result, err := client.Transcribe(context.Background(), Request{
		AudioPath: audioPath,
		APIKey:    "test-key",
		Model:     Model{ID: ModelWhisper1, Provider: ProviderOpenAI, APIName: "whisper-1"},
		BaseURL:   server.URL,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}

	if result.Text != "hello from test" {
		t.Fatalf("Text = %q, want %q", result.Text, "hello from test")
	}
}
