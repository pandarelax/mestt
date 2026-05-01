package transcribe

import "fmt"

type ProviderID string

const ProviderOpenAI ProviderID = "openai"

type ModelID string

const (
	ModelGPT4oMiniTranscribe ModelID = "gpt-4o-mini-transcribe"
	ModelGPT4oTranscribe     ModelID = "gpt-4o-transcribe"
	ModelWhisper1            ModelID = "whisper-1"
)

type Model struct {
	ID       ModelID
	Provider ProviderID
	Label    string
	APIName  string
}

var models = []Model{
	{ID: ModelGPT4oMiniTranscribe, Provider: ProviderOpenAI, Label: "GPT-4o Mini Transcribe", APIName: "gpt-4o-mini-transcribe"},
	{ID: ModelGPT4oTranscribe, Provider: ProviderOpenAI, Label: "GPT-4o Transcribe", APIName: "gpt-4o-transcribe"},
	{ID: ModelWhisper1, Provider: ProviderOpenAI, Label: "Whisper 1", APIName: "whisper-1"},
}

func Models() []Model {
	copyModels := make([]Model, len(models))
	copy(copyModels, models)
	return copyModels
}

func LookupModel(id string) (Model, error) {
	for _, model := range models {
		if string(model.ID) == id {
			return model, nil
		}
	}
	return Model{}, fmt.Errorf("unknown model %q", id)
}
