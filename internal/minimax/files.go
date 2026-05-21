package minimax

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type FileObject struct {
	FileID    int64  `json:"file_id"`
	Bytes     int64  `json:"bytes"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
	CreatedAt int64  `json:"created_at"`
}

type listFilesResponse struct {
	Files    []FileObject `json:"files"`
	BaseResp BaseResp     `json:"base_resp"`
}

type deleteFileResponse struct {
	FileID   int64    `json:"file_id"`
	BaseResp BaseResp `json:"base_resp"`
}

// ListFiles 列出文件，purpose 可选（voice_clone / prompt_audio / t2a_async_input）
func (c *Client) ListFiles(ctx context.Context, purpose string) ([]FileObject, error) {
	u := c.baseURL + "/v1/files/list"
	if purpose != "" {
		u += "?purpose=" + purpose
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result listFilesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if result.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("list files error %d: %s", result.BaseResp.StatusCode, result.BaseResp.StatusMsg)
	}
	return result.Files, nil
}

// DeleteFile 删除文件，需要 file_id 和 purpose
func (c *Client) DeleteFile(ctx context.Context, fileID int64, purpose string) error {
	body, err := c.post(ctx, "/v1/files/delete", "", map[string]any{
		"file_id": fileID,
		"purpose": purpose,
	})
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}

	var result deleteFileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if result.BaseResp.StatusCode != 0 {
		return fmt.Errorf("delete file error %d: %s", result.BaseResp.StatusCode, result.BaseResp.StatusMsg)
	}
	return nil
}
