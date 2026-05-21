package handler

import (
	"encoding/base64"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UploadRequest struct {
	Data      string `json:"data" binding:"required"`
	FileType  string `json:"file_type" binding:"required"` // text / tts / music
	Format    string `json:"format" binding:"required"`    // mp3 / wav / txt
	HistoryID string `json:"history_id"`                   // 可选，填写后同步更新历史记录的 R2 URL
}

func (h *Handler) Upload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 data: " + err.Error()})
		return
	}

	result, err := h.r2.Upload(c.Request.Context(), req.FileType, req.Format, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	// 有 history_id 且上传拿到公开 URL 时，同步更新历史记录
	if h.hist != nil && req.HistoryID != "" && result.PublicURL != "" {
		if err := h.hist.UpdateAudioURL(c.Request.Context(), req.HistoryID, result.PublicURL); err != nil {
			log.Printf("[history] update audio_url: %v", err)
		}
	}

	c.JSON(http.StatusOK, result)
}
