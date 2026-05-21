package history

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Record 代表一次生成操作的历史记录
type Record struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`       // music / tts / text / clone
	CreatedAt int64          `json:"created_at"` // Unix毫秒
	Title     string         `json:"title"`
	Params    map[string]any `json:"params,omitempty"`
	AudioURL  string         `json:"audio_url,omitempty"` // R2 公开URL
	Size      int64          `json:"size,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

// Store 是基于 SQLite 的历史记录存储，DB 文件放在 data/ 目录下
type Store struct {
	db *sql.DB
}

// New 打开（或创建）SQLite 数据库，dbPath 为文件路径，如 "data/history.db"
func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// 单写连接，避免 SQLITE_BUSY
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS history (
			id         TEXT    PRIMARY KEY,
			type       TEXT    NOT NULL,
			created_at INTEGER NOT NULL,
			title      TEXT    NOT NULL,
			params     TEXT,
			audio_url  TEXT,
			size       INTEGER,
			extra      TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_hist_type_time ON history(type, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_hist_time      ON history(created_at DESC);
	`)
	return err
}

// Add 插入一条记录，自动生成 ID 和 CreatedAt
func (s *Store) Add(ctx context.Context, r *Record) error {
	if r.ID == "" {
		id, err := randomID()
		if err != nil {
			return err
		}
		r.ID = id
	}
	if r.CreatedAt == 0 {
		r.CreatedAt = time.Now().UnixMilli()
	}

	params, _ := json.Marshal(r.Params)
	extra, _ := json.Marshal(r.Extra)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO history (id, type, created_at, title, params, audio_url, size, extra)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Type, r.CreatedAt, r.Title,
		string(params), r.AudioURL, r.Size, string(extra),
	)
	return err
}

// ListParams 查询参数
type ListParams struct {
	Type string // 空 = 所有类型
	Page int    // 1-based
	Size int    // 每页条数，默认20，最大100
}

// List 按时间倒序分页返回记录，同时返回总数
func (s *Store) List(ctx context.Context, p ListParams) ([]*Record, int64, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 || p.Size > 100 {
		p.Size = 20
	}
	offset := (p.Page - 1) * p.Size

	var (
		countQuery string
		listQuery  string
		args       []any
	)
	if p.Type != "" {
		countQuery = `SELECT COUNT(*) FROM history WHERE type = ?`
		listQuery = `SELECT id, type, created_at, title, params, audio_url, size, extra
		             FROM history WHERE type = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{p.Type}
	} else {
		countQuery = `SELECT COUNT(*) FROM history`
		listQuery = `SELECT id, type, created_at, title, params, audio_url, size, extra
		             FROM history ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{}
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	var queryArgs []any
	if p.Type != "" {
		queryArgs = []any{p.Type, p.Size, offset}
	} else {
		queryArgs = []any{p.Size, offset}
	}

	rows, err := s.db.QueryContext(ctx, listQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		var r Record
		var params, extra string
		if err := rows.Scan(&r.ID, &r.Type, &r.CreatedAt, &r.Title,
			&params, &r.AudioURL, &r.Size, &extra); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(params), &r.Params)
		_ = json.Unmarshal([]byte(extra), &r.Extra)
		records = append(records, &r)
	}
	return records, total, rows.Err()
}

// UpdateAudioURL 更新指定记录的 R2 音频地址（手动上传 R2 后调用）
func (s *Store) UpdateAudioURL(ctx context.Context, id, audioURL string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE history SET audio_url = ? WHERE id = ?`, audioURL, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("record %s not found", id)
	}
	return nil
}

// GetAudioURL 返回指定记录的 audio_url（删除前用于联动清理 R2）
func (s *Store) GetAudioURL(ctx context.Context, id string) (string, error) {
	var url string
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(audio_url,'') FROM history WHERE id = ?`, id).Scan(&url)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("record %s not found", id)
	}
	return url, err
}

// Delete 按 ID 删除记录
func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM history WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("record %s not found", id)
	}
	return nil
}

// Truncate 截断字符串，防止标题过长
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func randomID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
