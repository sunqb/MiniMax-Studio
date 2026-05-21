package minimax

import (
	"context"
	"encoding/json"
	"fmt"
)

// LyricsRequest 歌词生成请求
type LyricsRequest struct {
	Mode   string `json:"mode"`
	Prompt string `json:"prompt,omitempty"`
	Lyrics string `json:"lyrics,omitempty"`
	Title  string `json:"title,omitempty"`
}

// LyricsResponse 歌词生成响应
type LyricsResponse struct {
	SongTitle string   `json:"song_title"`
	StyleTags string   `json:"style_tags"`
	Lyrics    string   `json:"lyrics"`
	BaseResp  BaseResp `json:"base_resp"`
}

// LyricsResult 歌词生成结果
type LyricsResult struct {
	SongTitle string
	StyleTags string
	Lyrics    string
	Source    string // "minimax" 或 "llm"
}

// LyricsQuotaError 表示歌词生成配额不足错误
type LyricsQuotaError struct {
	Code    int
	Message string
}

func (e *LyricsQuotaError) Error() string {
	return fmt.Sprintf("lyrics quota error %d: %s", e.Code, e.Message)
}

// isLyricsQuotaCode 判断 status_code 是否为配额/限额错误
func isLyricsQuotaCode(code int) bool {
	// 1008: 余额不足
	// 1002: 访问频率超限（RPM）
	// 1039: 日配额已耗尽（Token Plan 每日上限）
	switch code {
	case 1002, 1008, 1039:
		return true
	}
	return false
}

// GenerateLyrics 调用 MiniMax /v1/lyrics_generation 生成歌词。
// 当配额不足时返回 *LyricsQuotaError，调用方可据此降级到大模型。
func (c *Client) GenerateLyrics(ctx context.Context, prompt string) (*LyricsResult, error) {
	req := LyricsRequest{
		Mode:   "write_full_song",
		Prompt: prompt,
	}

	body, err := c.post(ctx, "/v1/lyrics_generation", "", req)
	if err != nil {
		return nil, fmt.Errorf("lyrics generation request: %w", err)
	}

	var resp LyricsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse lyrics response: %w", err)
	}

	if resp.BaseResp.StatusCode != 0 {
		if isLyricsQuotaCode(resp.BaseResp.StatusCode) {
			return nil, &LyricsQuotaError{
				Code:    resp.BaseResp.StatusCode,
				Message: resp.BaseResp.StatusMsg,
			}
		}
		return nil, fmt.Errorf("lyrics API error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}

	return &LyricsResult{
		SongTitle: resp.SongTitle,
		StyleTags: resp.StyleTags,
		Lyrics:    resp.Lyrics,
		Source:    "minimax",
	}, nil
}
