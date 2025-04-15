package models

// GenerateContentResponse представляет структуру ответа от API
type GenerateContentResponse struct {
	Candidates     []Candidate    `json:"candidates"`
	PromptFeedback PromptFeedback `json:"promptFeedback"`
	UsageMetadata  UsageMetadata  `json:"usageMetadata"`
	ModelVersion   string         `json:"modelVersion"`
}

// Candidate представляет сгенерированный контент
type Candidate struct {
	Content       Content        `json:"content"`
	FinishReason  string         `json:"finishReason"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

// Content содержит части сгенерированного контента
type Content struct {
	Parts []Part `json:"parts"`
}

// Part представляет часть контента (например, текст)
type Part struct {
	Text string `json:"text"`
}

// SafetyRating представляет оценку безопасности
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// PromptFeedback содержит обратную связь по запросу
type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

// UsageMetadata содержит статистику использования
type UsageMetadata struct {
	PromptTokenCount     int32 `json:"promptTokenCount"`
	CandidatesTokenCount int32 `json:"candidatesTokenCount"`
	TotalTokenCount      int32 `json:"totalTokenCount"`
}

type GenerativeContentRequest struct {
	Contents []Content `json:"contents"`
}
