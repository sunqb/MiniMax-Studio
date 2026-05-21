package minimax

import (
	"context"
	"encoding/json"
	"fmt"
)

var ImageModelList = []string{"image-01", "image-01-live"}

var ImageAspectRatioList = []string{
	"1:1", "16:9", "4:3", "3:2", "2:3", "3:4", "9:16", "21:9",
}

type imageSubjectRef struct {
	Type      string `json:"type"`       // "character"
	ImageFile string `json:"image_file"` // URL 或 base64 data URL
}

type imageGenerateReq struct {
	Model            string            `json:"model"`
	Prompt           string            `json:"prompt"`
	AspectRatio      string            `json:"aspect_ratio,omitempty"`
	ResponseFormat   string            `json:"response_format,omitempty"`
	N                int               `json:"n,omitempty"`
	PromptOptimizer  bool              `json:"prompt_optimizer,omitempty"`
	SubjectReference []imageSubjectRef `json:"subject_reference,omitempty"`
}

type imageGenerateResp struct {
	ID   string `json:"id"`
	Data struct {
		ImageURLs []string `json:"image_urls"`
	} `json:"data"`
	Metadata struct {
		FailedCount  string `json:"failed_count"`
		SuccessCount string `json:"success_count"`
	} `json:"metadata"`
	BaseResp BaseResp `json:"base_resp"`
}

type ImageParams struct {
	Model           string
	Prompt          string
	AspectRatio     string
	N               int
	PromptOptimizer bool
	SubjectRefImage string // I2I：参考图 URL 或 base64 data URL，空则为 T2I
}

// GenerateImage 调用 /v1/image_generation，返回图片 URL 列表（有效期 24 小时）
func (c *Client) GenerateImage(ctx context.Context, params ImageParams) ([]string, error) {
	if params.Model == "" {
		params.Model = "image-01"
	}
	if params.N == 0 {
		params.N = 1
	}
	if params.AspectRatio == "" {
		params.AspectRatio = "1:1"
	}

	req := imageGenerateReq{
		Model:           params.Model,
		Prompt:          params.Prompt,
		AspectRatio:     params.AspectRatio,
		ResponseFormat:  "url",
		N:               params.N,
		PromptOptimizer: params.PromptOptimizer,
	}
	if params.SubjectRefImage != "" {
		req.SubjectReference = []imageSubjectRef{
			{Type: "character", ImageFile: params.SubjectRefImage},
		}
	}

	body, err := c.post(ctx, "/v1/image_generation", "", req)
	if err != nil {
		return nil, fmt.Errorf("image generation: %w", err)
	}

	var resp imageGenerateResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse image response: %w", err)
	}
	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("image API error %d: %s", resp.BaseResp.StatusCode, resp.BaseResp.StatusMsg)
	}
	return resp.Data.ImageURLs, nil
}
