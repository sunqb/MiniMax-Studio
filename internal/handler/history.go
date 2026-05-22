package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/history"
)

func (h *Handler) HistoryList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	records, total, err := h.hist.List(c.Request.Context(), history.ListParams{
		Type:   c.Query("type"),
		Source: c.Query("source"),
		Page:   page,
		Size:   size,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if records == nil {
		records = []*history.Record{}
	}
	c.JSON(http.StatusOK, gin.H{
		"records": records,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

type deleteHistoryRequest struct {
	ID string `json:"id" binding:"required"`
}

func (h *Handler) HistoryDelete(c *gin.Context) {
	var req deleteHistoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 先查 audio_url，若有则联动删除 R2 上的文件
	audioURL, err := h.hist.GetAudioURL(c.Request.Context(), req.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if audioURL != "" && h.r2 != nil {
		if key := h.r2.KeyFromURL(audioURL); key != "" {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 600*time.Second)
			defer cancel()
			if err := h.r2.Delete(ctx, key); err != nil {
				log.Printf("[history] delete R2 key=%s: %v", key, err)
				// R2 删除失败不阻断，继续删历史记录
			} else {
				log.Printf("[history] deleted R2 key=%s", key)
			}
		}
	}

	if err := h.hist.Delete(c.Request.Context(), req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": req.ID})
}
