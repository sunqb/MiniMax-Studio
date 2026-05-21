package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) FilesList(c *gin.Context) {
	purpose := c.Query("purpose") // 可选：voice_clone / prompt_audio

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	files, err := h.mm.ListFiles(ctx, purpose)
	if err != nil {
		log.Printf("[files] list error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

type deleteFileRequest struct {
	FileID  int64  `json:"file_id" binding:"required"`
	Purpose string `json:"purpose" binding:"required"`
}

func (h *Handler) FilesDelete(c *gin.Context) {
	var req deleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.mm.DeleteFile(ctx, req.FileID, req.Purpose); err != nil {
		log.Printf("[files] delete error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true, "file_id": req.FileID})
}
