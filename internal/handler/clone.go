package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 600*time.Second)
	defer cancel()

	resp, err := h.cloneMM.CloneVoice(ctx, minimax.VoiceCloneParams{
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

	demoAudioURL := resp.DemoAudio
	demoAudioSize := int64(0)
	if resp.DemoAudio != "" && h.r2 != nil {
		audioData, ext, err := fetchAudioURL(resp.DemoAudio)
		if err != nil {
			log.Printf("[clone] fetch demo audio for R2: %v", err)
		} else {
			demoAudioSize = int64(len(audioData))
			uploadCtx, uploadCancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
			defer uploadCancel()
			result, err := h.r2.Upload(uploadCtx, "clone", ext, audioData)
			if err != nil {
				log.Printf("[clone] upload demo audio to R2: %v", err)
			} else if result.PublicURL != "" {
				demoAudioURL = result.PublicURL
			}
		}
	}

	histID := ""
	rec := &history.Record{
		Type:  "clone",
		Title: voiceID,
		Params: map[string]any{
			"voice_id": voiceID,
			"filename": fh.Filename,
		},
		AudioURL: demoAudioURL,
		Size:     demoAudioSize,
	}
	if resp.DemoAudio != "" {
		rec.Extra = map[string]any{
			"demo_audio": resp.DemoAudio,
			"demo_r2":    demoAudioURL,
		}
	}
	if err := h.hist.Add(c.Request.Context(), rec); err != nil {
		log.Printf("[history] clone: %v", err)
	} else {
		histID = rec.ID
	}

	c.JSON(http.StatusOK, VoiceCloneResponse{
		VoiceID:   voiceID,
		DemoAudio: demoAudioURL,
		HistoryID: histID,
	})
}

func fetchAudioURL(url string) ([]byte, string, error) {
	client := &http.Client{Timeout: 600 * time.Second}
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

	ext := audioExtFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		ext = audioExtFromURL(url)
	}
	if ext == "" {
		ext = "mp3"
	}
	return data, ext, nil
}

func audioExtFromContentType(ct string) string {
	ct = strings.ToLower(strings.Split(ct, ";")[0])
	switch ct {
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/x-wav", "audio/wave":
		return "wav"
	case "audio/ogg":
		return "ogg"
	case "audio/flac":
		return "flac"
	case "audio/mp4", "audio/x-m4a":
		return "m4a"
	}
	return ""
}

func audioExtFromURL(url string) string {
	u := strings.ToLower(strings.Split(url, "?")[0])
	for _, ext := range []string{"mp3", "wav", "ogg", "flac", "m4a"} {
		if strings.HasSuffix(u, "."+ext) {
			return ext
		}
	}
	return ""
}

// ClonedVoiceList 从历史记录中提取所有已复刻的音色 ID（去重，按时间倒序）
func (h *Handler) ClonedVoiceList(c *gin.Context) {
	if h.hist == nil {
		c.JSON(http.StatusOK, gin.H{"voices": []string{}})
		return
	}
	// 取最近 100 条 clone 记录足够覆盖所有音色
	records, _, err := h.hist.List(c.Request.Context(), history.ListParams{
		Type: "clone",
		Page: 1,
		Size: 100,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	seen := map[string]bool{}
	voices := []string{}
	for _, r := range records {
		if id, ok := r.Params["voice_id"].(string); ok && id != "" && !seen[id] {
			seen[id] = true
			voices = append(voices, id)
		}
	}
	c.JSON(http.StatusOK, gin.H{"voices": voices})
}
