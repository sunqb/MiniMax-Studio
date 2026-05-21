package handler

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
)

type VoiceCloneResponse struct {
	VoiceID   string `json:"voice_id"`
	DemoAudio string `json:"demo_audio"`
	HistoryID string `json:"history_id,omitempty"`
}

func (h *Handler) VoiceClone(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	voiceID := c.PostForm("voice_id")
	if voiceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voice_id is required"})
		return
	}

	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	resp, err := h.mm.CloneVoice(ctx, minimax.VoiceCloneParams{
		FileData:                data,
		Filename:                fh.Filename,
		VoiceID:                 voiceID,
		PreviewText:             c.PostForm("preview_text"),
		NeedNoiseReduction:      c.PostForm("noise_reduction") == "true",
		NeedVolumeNormalization: c.PostForm("volume_normalization") == "true",
	})
	if err != nil {
		log.Printf("[clone] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	histID := ""
	rec := &history.Record{
		Type:  "clone",
		Title: voiceID,
		Params: map[string]any{
			"voice_id": voiceID,
			"filename": fh.Filename,
		},
		AudioURL: resp.DemoAudio, // MiniMax 试听链接直接作为 audio_url
	}
	if resp.DemoAudio != "" {
		rec.Extra = map[string]any{"demo_audio": resp.DemoAudio}
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] clone: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, VoiceCloneResponse{
		VoiceID:   voiceID,
		DemoAudio: resp.DemoAudio,
		HistoryID: histID,
	})
}
