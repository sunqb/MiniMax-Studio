package minimax

import (
	"context"
	"encoding/json"
	"fmt"
)

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TextRequest struct {
	Model       string        `json:"model"`
	Messages    []TextMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

type TextChoice struct {
	Message      TextMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
	Index        int         `json:"index"`
}

type TextUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type BaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type TextResponse struct {
	ID       string       `json:"id"`
	Choices  []TextChoice `json:"choices"`
	Usage    TextUsage    `json:"usage"`
	Model    string       `json:"model"`
	BaseResp BaseResp     `json:"base_resp"`
}

type TextGenerateParams struct {
	Model       string
	Messages    []TextMessage
	MaxTokens   int
	Temperature float64
}

func (c *Client) GenerateText(ctx context.Context, params TextGenerateParams) (*TextResponse, error) {
	if params.Model == "" {
		params.Model = "MiniMax-M2.7"
	}
	if params.MaxTokens == 0 {
		params.MaxTokens = 2048
	}
	if params.Temperature == 0 {
		params.Temperature = 0.7
	}

	req := TextRequest{
		Model:       params.Model,
		Messages:    params.Messages,
		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,
		Stream:      false,
	}

	// 使用 OpenAI 兼容接口（推荐）
	body, err := c.post(ctx, "/v1/chat/completions", "", req)
	if err != nil {
		return nil, fmt.Errorf("text generation: %w", err)
	}

	var resp TextResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("API error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}

	return &resp, nil
}
