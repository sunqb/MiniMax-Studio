package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunqb/minimax-studio/internal/history"
	"github.com/sunqb/minimax-studio/internal/minimax"
)

type ImageGenerateRequest struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt" binding:"required"`
	AspectRatio     string `json:"aspect_ratio"`
	N               int    `json:"n"`
	PromptOptimizer bool   `json:"prompt_optimizer"`
	SubjectRefImage string `json:"subject_ref_image"` // I2I：URL 或 base64 data URL
}

type ImageGenerateResponse struct {
	ImageURLs []string `json:"image_urls"`
	Count     int      `json:"count"`
	HistoryID string   `json:"history_id,omitempty"`
}

func (h *Handler) ImageGenerate(c *gin.Context) {
	var req ImageGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Model == "" {
		req.Model = "image-01"
	}
	if req.N < 1 {
		req.N = 1
	}
	if req.N > 9 {
		req.N = 9
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	urls, err := h.mm.GenerateImage(ctx, minimax.ImageParams{
		Model:           req.Model,
		Prompt:          req.Prompt,
		AspectRatio:     req.AspectRatio,
		N:               req.N,
		PromptOptimizer: req.PromptOptimizer,
		SubjectRefImage: req.SubjectRefImage,
	})
	if err != nil {
		log.Printf("[image] generate error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	mode := "t2i"
	if req.SubjectRefImage != "" {
		mode = "i2i"
	}

	histID := ""
	rec := &history.Record{
		Type:  "image",
		Title: history.Truncate(req.Prompt, 60),
		Params: map[string]any{
			"model":            req.Model,
			"prompt":           req.Prompt,
			"aspect_ratio":     req.AspectRatio,
			"n":                req.N,
			"prompt_optimizer": req.PromptOptimizer,
			"mode":             mode,
		},
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] image: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, ImageGenerateResponse{
		ImageURLs: urls,
		Count:     len(urls),
		HistoryID: histID,
	})
}

// ImageSaveR2 将 MiniMax 返回的图片 URL 下载后上传到 R2，避免 24h 过期
func (h *Handler) ImageSaveR2(c *gin.Context) {
	var body struct {
		URL       string `json:"url" binding:"required"`
		HistoryID string `json:"history_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	imgData, ext, err := fetchImageURL(body.URL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "fetch image: " + err.Error()})
		return
	}

	result, err := h.r2.Upload(c.Request.Context(), "image", ext, imgData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed: " + err.Error()})
		return
	}

	if h.hist != nil && body.HistoryID != "" && result.PublicURL != "" {
		if err := h.hist.UpdateAudioURL(c.Request.Context(), body.HistoryID, result.PublicURL); err != nil {
			log.Printf("[history] image update url: %v", err)
		}
	}

	c.JSON(http.StatusOK, result)
}

// ImageProxy 代理 MiniMax 图片 URL，供前端直接下载（绕过 CORS）
func (h *Handler) ImageProxy(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required"})
		return
	}

	imgData, ext, err := fetchImageURL(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	ct := contentTypeForExt(ext)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="image.%s"`, ext))
	c.Data(http.StatusOK, ct, imgData)
}

func fetchImageURL(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}

	ext := "jpg"
	ct := resp.Header.Get("Content-Type")
	switch ct {
	case "image/png":
		ext = "png"
	case "image/webp":
		ext = "webp"
	case "image/gif":
		ext = "gif"
	}
	return data, ext, nil
}

func contentTypeForExt(ext string) string {
	switch ext {
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}
