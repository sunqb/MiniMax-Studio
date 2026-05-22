package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/config"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
	"github.com/sunqb/minimax-studio/internal/storage"
)

type Handler struct {
	cfg        *config.Config
	mm         *minimax.Client
	r2         *storage.R2Client
	hist       *history.Store // nil 表示历史功能禁用
	taskResults sync.Map      // taskID → *TaskResult（临时缓存音频等大数据）
	musicSem   chan struct{}  // 音乐生成并发信号量
}

func New(cfg *config.Config, mm *minimax.Client, r2 *storage.R2Client, hist *history.Store) *Handler {
	return &Handler{
		cfg:      cfg,
		mm:       mm,
		r2:       r2,
		hist:     hist,
		musicSem: make(chan struct{}, 3), // 同时最多 3 个音乐生成任务
	}
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 文本生成
		api.POST("/text/generate", h.TextGenerate)
		api.POST("/text/generate-stream", h.TextGenerateStream)

		// 语音合成
		api.POST("/tts/synthesize", h.TTSSynthesize)
		api.GET("/tts/voices", h.TTSVoices)

		// 音乐合成
		api.POST("/music/generate", h.MusicGenerate)
		api.GET("/music/task/:id", h.MusicTaskStatus)
		api.GET("/music/models", h.MusicModels)
		api.POST("/lyrics/generate", h.LyricsGenerate)

		// 图像生成
		api.POST("/image/generate", h.ImageGenerate)
		api.POST("/image/save-r2", h.ImageSaveR2)
		api.GET("/image/proxy", h.ImageProxy)

		// 声音复刻
		api.POST("/voice/clone", h.VoiceClone)
		api.GET("/voice/cloned", h.ClonedVoiceList)

		// MiniMax 文件管理
		api.GET("/files", h.FilesList)
		api.POST("/files/delete", h.FilesDelete)

		// R2 上传
		api.POST("/upload", h.Upload)

		// 历史记录
		api.GET("/history", h.HistoryList)
		api.POST("/history/delete", h.HistoryDelete)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}
