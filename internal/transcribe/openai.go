package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Request struct {
	AudioPath string
	APIKey    string
	Model     Model
	BaseURL   string
	Timeout   time.Duration
}

type Result struct {
	Text     string
	ModelID  string
	Provider string
}

type OpenAIClient struct {
	HTTPClient *http.Client
}

type openAIResponse struct {
	Text string `json:"text"`
}

func (c OpenAIClient) Transcribe(ctx context.Context, req Request) (Result, error) {
	file, err := os.Open(req.AudioPath)
	if err != nil {
		return Result{}, fmt.Errorf("open audio file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(req.AudioPath))
	if err != nil {
		return Result{}, fmt.Errorf("create multipart file field: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return Result{}, fmt.Errorf("copy audio file into request: %w", err)
	}

	if err := writer.WriteField("model", req.Model.APIName); err != nil {
		return Result{}, fmt.Errorf("write model field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return Result{}, fmt.Errorf("close multipart writer: %w", err)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: req.Timeout}
	}

	endpoint := req.BaseURL + "/audio/transcriptions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return Result{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return Result{}, fmt.Errorf("send transcription request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return Result{}, fmt.Errorf("openai transcription failed: status=%d body=%s", resp.StatusCode, errBody.String())
	}

	var parsed openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Result{}, fmt.Errorf("decode transcription response: %w", err)
	}

	return Result{
		Text:     parsed.Text,
		ModelID:  string(req.Model.ID),
		Provider: string(req.Model.Provider),
	}, nil
}
