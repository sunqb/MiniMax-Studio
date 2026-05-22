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
	AutoUploadR2    bool   `json:"auto_upload_r2"`
}

type MusicAsyncResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

type TaskResult struct {
	Audio     string `json:"audio"` // base64
	Format    string `json:"format"`
	Size      int    `json:"size"`
	HistoryID string `json:"history_id,omitempty"`
	PublicURL string `json:"public_url,omitempty"`
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

	// 创建 pending 状态的历史记录作为任务
	rec := &history.Record{
		Type:   "music",
		Title:  history.Truncate(req.Prompt, 60),
		Status: "pending",
		Params: map[string]any{
			"model":            req.Model,
			"prompt":           req.Prompt,
			"is_instrumental":  req.IsInstrumental,
			"lyrics_optimizer": req.LyricsOptimizer,
			"format":           req.Format,
			"auto_upload_r2":   req.AutoUploadR2,
		},
	}
	if req.Lyrics != "" {
		rec.Extra = map[string]any{"lyrics": req.Lyrics}
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[music] create task error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}

	taskID := rec.ID

	// 立即返回 task_id
	c.JSON(http.StatusOK, MusicAsyncResponse{
		TaskID: taskID,
		Status: "pending",
	})

	// 后台 goroutine 执行生成
	go h.runMusicTask(taskID, req)
}

func (h *Handler) runMusicTask(taskID string, req MusicGenerateRequest) {
	// 获取信号量（限制并发数）
	h.musicSem <- struct{}{}
	defer func() { <-h.musicSem }()

	// 更新为 running
	_ = h.hist.UpdateStatus(context.Background(), taskID, "running", "")

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
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
		log.Printf("[music] generate error (task %s): %v", taskID, err)
		_ = h.hist.UpdateStatus(context.Background(), taskID, "failed", err.Error())
		return
	}

	publicURL := ""
	if req.AutoUploadR2 && h.r2 != nil {
		uploadCtx, uploadCancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer uploadCancel()
		uploadResult, err := h.r2.Upload(uploadCtx, "music", req.Format, audioData)
		if err != nil {
			log.Printf("[music] auto upload R2 error (task %s): %v", taskID, err)
		} else if uploadResult.PublicURL != "" {
			publicURL = uploadResult.PublicURL
			if err := h.hist.UpdateAudioURL(context.Background(), taskID, publicURL); err != nil {
				log.Printf("[history] update music audio_url (task %s): %v", taskID, err)
			}
		}
	}

	// 缓存音频数据到内存（base64），前端当前页面轮询时取走
	result := &TaskResult{
		Audio:     base64.StdEncoding.EncodeToString(audioData),
		Format:    req.Format,
		Size:      len(audioData),
		HistoryID: taskID,
		PublicURL: publicURL,
	}
	h.taskResults.Store(taskID, result)

	// 更新历史记录为 completed
	extra := map[string]any{}
	if req.Lyrics != "" {
		extra["lyrics"] = req.Lyrics
	}
	_ = h.hist.UpdateResult(context.Background(), taskID, extra, int64(len(audioData)))
}

// MusicTaskStatus 查询音乐生成任务状态
func (h *Handler) MusicTaskStatus(c *gin.Context) {
	taskID := c.Param("id")

	rec, err := h.hist.GetByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	resp := gin.H{
		"id":         rec.ID,
		"type":       rec.Type,
		"status":     effectiveStatus(rec.Status),
		"created_at": rec.CreatedAt,
	}
	if rec.AudioURL != "" {
		resp["audio_url"] = rec.AudioURL
	}

	if rec.Status == "failed" {
		resp["error"] = rec.Error
	}

	if rec.Status == "completed" || rec.Status == "" {
		// 从内存缓存取结果
		if val, ok := h.taskResults.Load(taskID); ok {
			resp["result"] = val
		}
	}

	c.JSON(http.StatusOK, resp)
}

// effectiveStatus 兼容旧数据：空 status 等价 completed
func effectiveStatus(s string) string {
	if s == "" {
		return "completed"
	}
	return s
}

func (h *Handler) MusicModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"models": minimax.MusicModelList})
}
