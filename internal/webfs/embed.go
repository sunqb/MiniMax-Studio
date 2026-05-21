package webfs

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed static
var embedFS embed.FS

// FS 返回嵌入的前端静态文件系统
func FS() fs.FS {
	sub, err := fs.Sub(embedFS, "static")
	if err != nil {
		log.Fatalf("webfs: %v", err)
	}
	return sub
}
