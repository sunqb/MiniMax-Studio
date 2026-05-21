package minimax

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type VoiceSetting struct {
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed"`
	Vol     float64 `json:"vol"`
	Pitch   int     `json:"pitch"`
}

type AudioSetting struct {
	SampleRate int    `json:"sample_rate"` // 官方字段名是 sample_rate
	Bitrate    int    `json:"bitrate"`
	Format     string `json:"format"`
	Channel    int    `json:"channel"`
}

type TTSRequest struct {
	Model        string       `json:"model"`
	Text         string       `json:"text"`
	Stream       bool         `json:"stream"`
	VoiceSetting VoiceSetting `json:"voice_setting"`
	AudioSetting AudioSetting `json:"audio_setting"`
}

type TTSData struct {
	Audio  string `json:"audio"` // hex encoded
	Status int    `json:"status"`
}

type TTSResponse struct {
	Data     TTSData  `json:"data"`
	TraceID  string   `json:"trace_id"`
	BaseResp BaseResp `json:"base_resp"`
}

type TTSParams struct {
	Text    string
	VoiceID string
	Speed   float64
	Vol     float64
	Pitch   int
	Format  string
	Model   string
}

// VoiceList 常用音色列表
var VoiceList = []map[string]string{
	{"id": "male-qn-qingse", "name": "青涩青年音色"},
	{"id": "male-qn-jingying", "name": "精英青年音色"},
	{"id": "male-qn-badao", "name": "霸道青年音色"},
	{"id": "male-qn-daxuesheng", "name": "青年大学生音色"},
	{"id": "female-shaonv", "name": "少女音色"},
	{"id": "female-yujie", "name": "御姐音色"},
	{"id": "female-chengshu", "name": "成熟女性音色"},
	{"id": "female-tianmei", "name": "甜美女性音色"},
	{"id": "presenter_male", "name": "男性主持人"},
	{"id": "presenter_female", "name": "女性主持人"},
	{"id": "audiobook_male_1", "name": "男性有声书1"},
	{"id": "audiobook_male_2", "name": "男性有声书2"},
	{"id": "audiobook_female_1", "name": "女性有声书1"},
	{"id": "audiobook_female_2", "name": "女性有声书2"},
	{"id": "male-qn-qingse-jingpin", "name": "青涩青年音色-beta"},
	{"id": "male-qn-jingying-jingpin", "name": "精英青年音色-beta"},
	{"id": "male-qn-badao-jingpin", "name": "霸道青年音色-beta"},
	{"id": "male-qn-daxuesheng-jingpin", "name": "青年大学生音色-beta"},
	{"id": "female-shaonv-jingpin", "name": "少女音色-beta"},
	{"id": "female-yujie-jingpin", "name": "御姐音色-beta"},
}

func (c *Client) SynthesizeSpeech(ctx context.Context, params TTSParams) ([]byte, error) {
	if params.Model == "" {
		params.Model = "speech-2.8-hd"
	}
	if params.VoiceID == "" {
		params.VoiceID = "female-shaonv"
	}
	if params.Speed == 0 {
		params.Speed = 1.0
	}
	if params.Vol == 0 {
		params.Vol = 1.0
	}
	if params.Format == "" {
		params.Format = "mp3"
	}

	req := TTSRequest{
		Model: params.Model,
		Text:  params.Text,
		VoiceSetting: VoiceSetting{
			VoiceID: params.VoiceID,
			Speed:   params.Speed,
			Vol:     params.Vol,
			Pitch:   params.Pitch,
		},
		AudioSetting: AudioSetting{
			SampleRate: 32000,
			Bitrate:    128000,
			Format:     params.Format,
			Channel:    1,
		},
	}

	body, err := c.post(ctx, "/v1/t2a_v2", "", req)
	if err != nil {
		return nil, fmt.Errorf("TTS request: %w", err)
	}

	var resp TTSResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse TTS response: %w", err)
	}

	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("TTS API error: %s", resp.BaseResp.StatusMsg)
	}

	// 解码 hex 音频数据
	audioBytes, err := hex.DecodeString(resp.Data.Audio)
	if err != nil {
		return nil, fmt.Errorf("decode audio hex: %w", err)
	}

	return audioBytes, nil
}
