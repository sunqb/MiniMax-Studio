package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
)

type TextGenerateRequest struct {
	Model        string  `json:"model"`
	SystemPrompt string  `json:"system_prompt"`
	UserMessage  string  `json:"user_message" binding:"required"`
	MaxTokens    int     `json:"max_tokens"`
	Temperature  float64 `json:"temperature"`
}

type TextGenerateResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	PromptTokens int    `json:"prompt_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	HistoryID    string `json:"history_id,omitempty"`
}

func (h *Handler) TextGenerate(c *gin.Context) {
	var req TextGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages := []minimax.TextMessage{}
	if req.SystemPrompt != "" {
		messages = append(messages, minimax.TextMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, minimax.TextMessage{Role: "user", Content: req.UserMessage})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 600*time.Second)
	defer cancel()

	resp, err := h.mm.GenerateText(ctx, minimax.TextGenerateParams{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
	if err != nil {
		log.Printf("[text] generate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	histID := ""
	rec := &history.Record{
		Type:  "text",
		Title: history.Truncate(req.UserMessage, 60),
		Params: map[string]any{
			"model":       req.Model,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
		},
		Extra: map[string]any{
			"content":      content,
			"total_tokens": resp.Usage.TotalTokens,
		},
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] text: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, TextGenerateResponse{
		Content:      content,
		Model:        resp.Model,
		PromptTokens: resp.Usage.PromptTokens,
		TotalTokens:  resp.Usage.TotalTokens,
		HistoryID:    histID,
	})
}

func (h *Handler) TextGenerateStream(c *gin.Context) {
	var req TextGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages := []minimax.TextMessage{}
	if req.SystemPrompt != "" {
		messages = append(messages, minimax.TextMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, minimax.TextMessage{Role: "user", Content: req.UserMessage})

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 600*time.Second)
	defer cancel()

	resp, err := h.mm.GenerateTextStream(ctx, minimax.TextGenerateParams{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}, func(delta string) error {
		return writeTextStreamEvent(c, "delta", gin.H{"delta": delta})
	})
	if err != nil {
		log.Printf("[text] stream generate error: %v", err)
		_ = writeTextStreamEvent(c, "error", gin.H{"error": err.Error()})
		return
	}

	content := ""
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	histID := ""
	rec := &history.Record{
		Type:  "text",
		Title: history.Truncate(req.UserMessage, 60),
		Params: map[string]any{
			"model":       req.Model,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
		},
		Extra: map[string]any{
			"content":      content,
			"total_tokens": resp.Usage.TotalTokens,
		},
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] text stream: %v", err)
	} else {
		histID = rec.ID
	}

	_ = writeTextStreamEvent(c, "done", gin.H{
		"model":         resp.Model,
		"prompt_tokens": resp.Usage.PromptTokens,
		"total_tokens":  resp.Usage.TotalTokens,
		"history_id":    histID,
	})
}

func writeTextStreamEvent(c *gin.Context, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := c.Writer.WriteString("event: " + event + "\n"); err != nil {
		return err
	}
	if _, err := c.Writer.WriteString("data: " + string(data) + "\n\n"); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}
