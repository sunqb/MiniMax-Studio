package minimax

import (
	"context"
	"encoding/json"
	"fmt"
)

// UploadFileResponse 文件上传响应
type UploadFileResponse struct {
	File struct {
		FileID    int64  `json:"file_id"`
		Bytes     int64  `json:"bytes"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
		CreatedAt int64  `json:"created_at"`
	} `json:"file"`
	BaseResp BaseResp `json:"base_resp"`
}

// VoiceCloneRequest 声音复刻请求
type VoiceCloneRequest struct {
	FileID                  int64        `json:"file_id"`
	VoiceID                 string       `json:"voice_id"`
	ClonePrompt             *ClonePrompt `json:"clone_prompt,omitempty"`
	Text                    string       `json:"text,omitempty"`
	Model                   string       `json:"model,omitempty"`
	NeedNoiseReduction      bool         `json:"need_noise_reduction"`
	NeedVolumeNormalization bool         `json:"need_volume_normalization"`
}

type ClonePrompt struct {
	PromptAudio int64  `json:"prompt_audio"`
	PromptText  string `json:"prompt_text"`
}

// VoiceCloneResponse 声音复刻响应
type VoiceCloneResponse struct {
	DemoAudio      string   `json:"demo_audio"`       // 试听音频 URL（有效期 24h）
	InputSensitive bool     `json:"input_sensitive"`
	BaseResp       BaseResp `json:"base_resp"`
}

// VoiceCloneParams 调用参数
type VoiceCloneParams struct {
	FileData                []byte
	Filename                string
	VoiceID                 string
	PreviewText             string // 可选，提供后返回试听 URL
	NeedNoiseReduction      bool
	NeedVolumeNormalization bool
}

// UploadVoiceFile 上传待复刻音频，返回 file_id
func (c *Client) UploadVoiceFile(ctx context.Context, filename string, data []byte) (int64, error) {
	body, err := c.postMultipart(ctx, "/v1/files/upload",
		map[string]string{"purpose": "voice_clone"},
		"file", filename, data,
	)
	if err != nil {
		return 0, fmt.Errorf("upload file: %w", err)
	}

	var resp UploadFileResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("parse upload response: %w", err)
	}
	if resp.BaseResp.StatusCode != 0 {
		return 0, fmt.Errorf("upload error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}
	return resp.File.FileID, nil
}

// CloneVoice 声音快速复刻
func (c *Client) CloneVoice(ctx context.Context, params VoiceCloneParams) (*VoiceCloneResponse, error) {
	fileID, err := c.UploadVoiceFile(ctx, params.Filename, params.FileData)
	if err != nil {
		return nil, err
	}

	req := VoiceCloneRequest{
		FileID:                  fileID,
		VoiceID:                 params.VoiceID,
		NeedNoiseReduction:      params.NeedNoiseReduction,
		NeedVolumeNormalization: params.NeedVolumeNormalization,
	}
	if params.PreviewText != "" {
		req.Text = params.PreviewText
		req.Model = "speech-2.8-hd"
	}

	body, err := c.post(ctx, "/v1/voice_clone", "", req)
	if err != nil {
		return nil, fmt.Errorf("voice clone: %w", err)
	}

	var resp VoiceCloneResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse clone response: %w", err)
	}
	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("clone error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}
	return &resp, nil
}
