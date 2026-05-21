package minimax

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type MusicAudioSetting struct {
	SampleRate int    `json:"sample_rate"`
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
}

type MusicRequest struct {
	Model           string            `json:"model"`
	Prompt          string            `json:"prompt,omitempty"`
	Lyrics          string            `json:"lyrics,omitempty"`
	Stream          bool              `json:"stream"`
	IsInstrumental  bool              `json:"is_instrumental,omitempty"`
	LyricsOptimizer bool              `json:"lyrics_optimizer,omitempty"`
	AudioSetting    MusicAudioSetting `json:"audio_setting"`
}

type MusicData struct {
	Audio  string `json:"audio"`
	Status int    `json:"status"`
}

type MusicResponse struct {
	Data     MusicData `json:"data"`
	TraceID  string    `json:"trace_id"`
	BaseResp BaseResp  `json:"base_resp"`
}

type MusicParams struct {
	Model           string
	Prompt          string // 风格描述，如"流行音乐,忧郁,适合在下雨的晚上"
	Lyrics          string
	IsInstrumental  bool // 纯音乐（无人声）
	LyricsOptimizer bool // 根据 prompt 自动生成歌词
	Format          string
}

// MusicModelList 可用模型
// music-2.6 / music-cover：仅限 Token Plan 用户，RPM 较高
// music-2.6-free / music-cover-free：所有用户可用，RPM 较低
var MusicModelList = []map[string]string{
	{"id": "music-2.6", "name": "music-2.6（Token Plan）"},
	{"id": "music-2.6-free", "name": "music-2.6-free（免费可用）"},
}

func (c *Client) GenerateMusic(ctx context.Context, params MusicParams) ([]byte, error) {
	if params.Model == "" {
		params.Model = "music-2.6"
	}
	if params.Format == "" {
		params.Format = "mp3"
	}

	req := MusicRequest{
		Model:           params.Model,
		Prompt:          params.Prompt,
		Lyrics:          params.Lyrics,
		Stream:          false,
		IsInstrumental:  params.IsInstrumental,
		LyricsOptimizer: params.LyricsOptimizer,
		AudioSetting: MusicAudioSetting{
			SampleRate: 44100,
			Bitrate:    256000,
			Format:     params.Format,
		},
	}

	body, err := c.post(ctx, "/v1/music_generation", "", req)
	if err != nil {
		return nil, fmt.Errorf("music generation request: %w", err)
	}

	var resp MusicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse music response: %w", err)
	}

	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("music API error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}

	if len(resp.Data.Audio) == 0 {
		return nil, fmt.Errorf("empty audio data")
	}

	return decodeAudio(resp.Data.Audio)
}

// decodeAudio 先尝试 hex，失败则尝试 base64
func decodeAudio(s string) ([]byte, error) {
	if b, err := hex.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return nil, fmt.Errorf("unable to decode audio data (not hex or base64)")
}
