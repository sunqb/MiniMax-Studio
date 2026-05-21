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

type TTSSynthesizeRequest struct {
	Text      string  `json:"text" binding:"required"`
	VoiceID   string  `json:"voice_id"`
	Speed     float64 `json:"speed"`
	Vol       float64 `json:"vol"`
	Pitch     int     `json:"pitch"`
	Format    string  `json:"format"`
	Model     string  `json:"model"`
	AppSource string  `json:"app_source"` // 来源应用，如 "音色复刻配音"
}

type TTSSynthesizeResponse struct {
	Audio     string `json:"audio"` // base64
	Format    string `json:"format"`
	Size      int    `json:"size"`
	HistoryID string `json:"history_id,omitempty"`
}

func (h *Handler) TTSSynthesize(c *gin.Context) {
	var req TTSSynthesizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Format == "" {
		req.Format = "mp3"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	audioData, err := h.mm.SynthesizeSpeech(ctx, minimax.TTSParams{
		Text:    req.Text,
		VoiceID: req.VoiceID,
		Speed:   req.Speed,
		Vol:     req.Vol,
		Pitch:   req.Pitch,
		Format:  req.Format,
		Model:   req.Model,
	})
	if err != nil {
		log.Printf("[tts] synthesize error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	histID := ""
	rec := &history.Record{
		Type:  "tts",
		Title: history.Truncate(req.Text, 60),
		Params: map[string]any{
			"model":      req.Model,
			"voice_id":   req.VoiceID,
			"speed":      req.Speed,
			"vol":        req.Vol,
			"pitch":      req.Pitch,
			"format":     req.Format,
			"app_source": req.AppSource,
		},
		Size: int64(len(audioData)),
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] tts: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, TTSSynthesizeResponse{
		Audio:     base64.StdEncoding.EncodeToString(audioData),
		Format:    req.Format,
		Size:      len(audioData),
		HistoryID: histID,
	})
}

func (h *Handler) TTSVoices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"voices": minimax.VoiceList})
}
