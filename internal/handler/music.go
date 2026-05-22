package handler

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
)

type MusicGenerateRequest struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	Lyrics          string `json:"lyrics"`
	IsInstrumental  bool   `json:"is_instrumental"`
	LyricsOptimizer bool   `json:"lyrics_optimizer"`
	Format          string `json:"format"`
}

type MusicGenerateResponse struct {
	Audio     string `json:"audio"` // base64
	Format    string `json:"format"`
	Size      int    `json:"size"`
	HistoryID string `json:"history_id,omitempty"`
}

func (h *Handler) MusicGenerate(c *gin.Context) {
	var req MusicGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "mp3"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 180*time.Second)
	defer cancel()

	audioData, err := h.mm.GenerateMusic(ctx, minimax.MusicParams{
		Model:           req.Model,
		Prompt:          req.Prompt,
		Lyrics:          req.Lyrics,
		IsInstrumental:  req.IsInstrumental,
		LyricsOptimizer: req.LyricsOptimizer,
		Format:          req.Format,
	})
	if err != nil {
		log.Printf("[music] generate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 同步写历史，拿到 ID 返回给前端
	histID := ""
	rec := &history.Record{
		Type:  "music",
		Title: history.Truncate(req.Prompt, 60),
		Params: map[string]any{
			"model":            req.Model,
			"prompt":           req.Prompt,
			"is_instrumental":  req.IsInstrumental,
			"lyrics_optimizer": req.LyricsOptimizer,
			"format":           req.Format,
		},
		Size: int64(len(audioData)),
	}
	if req.Lyrics != "" {
		rec.Extra = map[string]any{"lyrics": req.Lyrics}
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] music: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, MusicGenerateResponse{
		Audio:     base64.StdEncoding.EncodeToString(audioData),
		Format:    req.Format,
		Size:      len(audioData),
		HistoryID: histID,
	})
}

func (h *Handler) MusicModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"models": minimax.MusicModelList})
}
