package ollama

type PullModelRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure"`
	Stream   bool   `json:"stream"`
}

type PullModelResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
}

type ListModelsResponse struct {
	Models []ModelInformation `json:"models"`
}

type ModelInformation struct {
	Name       string                  `json:"name"`
	ModifiedAt string                  `json:"modified_at"`
	Size       int64                   `json:"size"`
	Digest     string                  `json:"digest"`
	Details    ModelInformationDetails `json:"details"`
}

type ModelInformationDetails struct {
	Format            string `json:"format"`
	Family            string `json:"family"`
	Families          any    `json:"families"`
	ParameterSize     string `json:"parameter_size"`
	QuantizationLevel string `json:"quantization_level"`
}

type GetModelRequest struct {
	Name string `json:"name"`
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

type ConverseRequest struct {
	Model    string                `json:"model"`
	Messages []ConversationMessage `json:"messages"`
	Stream   bool                  `json:"stream"`
}

type ConversationActor string

const (
	User      ConversationActor = "user"
	Assistant ConversationActor = "assistant"
	System    ConversationActor = "system"
)

type ConversationMessage struct {
	Role    ConversationActor `json:"role"`
	Content string            `json:"content"`
}

type ConverseResponse struct {
	Model              string              `json:"model"`
	CreatedAt          string              `json:"created_at"`
	Message            ConversationMessage `json:"message"`
	Done               bool                `json:"done"`
	TotalDuration      int64               `json:"total_duration"`
	LoadDuration       int64               `json:"load_duration"`
	PromptEvalCount    int64               `json:"prompt_eval_count"`
	PromptEvalDuration int64               `json:"prompt_eval_duration"`
	EvalCount          int64               `json:"eval_count"`
	EvalDuration       int64               `json:"eval_duration"`
}
