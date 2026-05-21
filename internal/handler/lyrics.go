package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/minimax"
)

type LyricsGenerateRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type LyricsGenerateResponse struct {
	SongTitle string `json:"song_title"`
	StyleTags string `json:"style_tags"`
	Lyrics    string `json:"lyrics"`
	Source    string `json:"source"` // "minimax" 或 "llm"
}

func (h *Handler) LyricsGenerate(c *gin.Context) {
	var req LyricsGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// 优先使用 MiniMax 歌词生成 API（每日 100 次），任意错误降级到大模型
	result, err := h.mm.GenerateLyrics(ctx, req.Prompt)
	if err != nil {
		log.Printf("[lyrics] MiniMax 歌词 API 失败(%v)，降级到大模型生成", err)
		result, err = h.generateLyricsWithLLM(ctx, req.Prompt)
		if err != nil {
			log.Printf("[lyrics] 大模型降级失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, LyricsGenerateResponse{
		SongTitle: result.SongTitle,
		StyleTags: result.StyleTags,
		Lyrics:    result.Lyrics,
		Source:    result.Source,
	})
}

// generateLyricsWithLLM 使用大模型生成歌词（MiniMax 配额耗尽时的降级方案）
func (h *Handler) generateLyricsWithLLM(ctx context.Context, prompt string) (*minimax.LyricsResult, error) {
	systemPrompt := `你是一位专业词曲创作者。根据用户描述生成完整的中文歌词。
格式要求：
- 使用结构标签：[Verse]、[Chorus]、[Bridge]、[Outro] 等，每段之间空一行
- 歌词自然流畅，适合配乐演唱
- 只返回歌词内容，不要包含任何解释`

	resp, err := h.mm.GenerateText(ctx, minimax.TextGenerateParams{
		Model: "MiniMax-M2.7",
		Messages: []minimax.TextMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("请根据以下描述创作歌词：%s", prompt)},
		},
		MaxTokens:   1024,
		Temperature: 0.8,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 歌词生成: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("大模型返回空结果")
	}

	return &minimax.LyricsResult{
		SongTitle: "",
		StyleTags: "",
		Lyrics:    strings.TrimSpace(resp.Choices[0].Message.Content),
		Source:    "llm",
	}, nil
}
