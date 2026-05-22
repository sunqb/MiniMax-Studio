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
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Record 代表一次生成操作的历史记录
type Record struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`       // music / tts / text / clone / image
	CreatedAt int64          `json:"created_at"` // Unix毫秒
	Title     string         `json:"title"`
	Params    map[string]any `json:"params,omitempty"`
	AudioURL  string         `json:"audio_url,omitempty"` // R2 公开URL
	Size      int64          `json:"size,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
	AppSource string         `json:"app_source,omitempty"` // 来源应用，空=原子功能直接调用
	Status    string         `json:"status,omitempty"`     // pending / running / completed / failed，空=completed（兼容旧数据）
	Error     string         `json:"error,omitempty"`      // 失败原因
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
	// 1. 创建基础表（不含 app_source，兼容旧库）
	if _, err := db.Exec(`
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
	`); err != nil {
		return err
	}
	// 2. 兼容旧数据库：若 app_source 列不存在则添加（已有则忽略）
	_, _ = db.Exec(`ALTER TABLE history ADD COLUMN app_source TEXT NOT NULL DEFAULT ''`)
	// 3. 创建 app_source 索引（表与列已确保存在）
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_hist_app ON history(app_source, created_at DESC)`); err != nil {
		return err
	}
	// 4. 异步任务：添加 status 和 error 列（兼容旧库）
	_, _ = db.Exec(`ALTER TABLE history ADD COLUMN status TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE history ADD COLUMN error TEXT NOT NULL DEFAULT ''`)
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_hist_status ON history(status)`); err != nil {
		return err
	}
	return nil
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
		`INSERT INTO history (id, type, created_at, title, params, audio_url, size, extra, app_source, status, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Type, r.CreatedAt, r.Title,
		string(params), r.AudioURL, r.Size, string(extra), r.AppSource,
		r.Status, r.Error,
	)
	return err
}

// ListParams 查询参数
type ListParams struct {
	Type   string // 空 = 所有类型
	Source string // "app"=仅应用市集, "atomic"=仅原子功能, ""=全部
	Page   int    // 1-based
	Size   int    // 每页条数，默认20，最大100
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

	// 构建 WHERE 子句
	var conditions []string
	var args []any
	if p.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, p.Type)
	}
	switch p.Source {
	case "app":
		conditions = append(conditions, "app_source != ''")
	case "atomic":
		conditions = append(conditions, "app_source = ''")
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM history"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := "SELECT id, type, created_at, title, params, audio_url, size, extra, app_source, status, error FROM history" +
		where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, p.Size, offset)

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
			&params, &r.AudioURL, &r.Size, &extra, &r.AppSource, &r.Status, &r.Error); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(params), &r.Params)
		_ = json.Unmarshal([]byte(extra), &r.Extra)
		records = append(records, &r)
	}
	return records, total, rows.Err()
}

// GetByID 按 ID 查询单条记录（轮询任务状态用）
func (s *Store) GetByID(ctx context.Context, id string) (*Record, error) {
	var r Record
	var params, extra string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, type, created_at, title, params, audio_url, size, extra, app_source, status, error
		 FROM history WHERE id = ?`, id).Scan(
		&r.ID, &r.Type, &r.CreatedAt, &r.Title,
		&params, &r.AudioURL, &r.Size, &extra, &r.AppSource,
		&r.Status, &r.Error,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("record %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(params), &r.Params)
	_ = json.Unmarshal([]byte(extra), &r.Extra)
	return &r, nil
}

// UpdateStatus 更新任务状态
func (s *Store) UpdateStatus(ctx context.Context, id, status, errMsg string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE history SET status = ?, error = ? WHERE id = ?`, status, errMsg, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("record %s not found", id)
	}
	return nil
}

// UpdateResult 完成时回写 extra 和 size
func (s *Store) UpdateResult(ctx context.Context, id string, extra map[string]any, size int64) error {
	extraJSON, _ := json.Marshal(extra)
	res, err := s.db.ExecContext(ctx,
		`UPDATE history SET extra = ?, size = ?, status = 'completed' WHERE id = ?`,
		string(extraJSON), size, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("record %s not found", id)
	}
	return nil
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
