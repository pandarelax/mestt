package transcribe

import "fmt"

type ProviderID string

const (
	ProviderOpenAI ProviderID = "openai"
	ProviderLocal  ProviderID = "local"
)

type ModelID string

const (
	ModelGPT4oMiniTranscribe ModelID = "gpt-4o-mini-transcribe"
	ModelGPT4oTranscribe     ModelID = "gpt-4o-transcribe"
	ModelWhisper1            ModelID = "whisper-1"
	ModelLargeV3TurboLocal   ModelID = "large-v3-turbo"
	ModelDistilLargeV3Local  ModelID = "distil-large-v3"
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
	{ID: ModelLargeV3TurboLocal, Provider: ProviderLocal, Label: "Local Whisper Large V3 Turbo", APIName: "large-v3-turbo"},
	{ID: ModelDistilLargeV3Local, Provider: ProviderLocal, Label: "Local Distil Whisper Large V3", APIName: "distil-large-v3"},
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
