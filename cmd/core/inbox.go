// inbox.go —— 业主投件道（业主令 11-12：文件传前端落到可读盒）。
// POST /inbox multipart field=file → O_INBOX_DIR(缺省 /opt/o-inbox,部署树外
// 净区——防脏树闸)/<UTCts>_<净名>；ops_event 簿记行审计。盒外路=dev 直读/
// sg ssh 拉——落径回包业主知悉。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	inboxMaxBytes = 64 << 20 // 64MB 上限(图档/日志/dump 谱全盖)
	inboxMaxName  = 120
)

func inboxDir() string {
	if d := os.Getenv("O_INBOX_DIR"); d != "" {
		return d
	}
	return "/opt/o-inbox"
}

// 文件名净:basename 剥径+白名单(字母/数字/CJK/./-/_)+截长——
// 路径注入与怪字符在此绝。
func safeInboxName(raw string) string {
	base := filepath.Base(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range base {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	name := strings.Trim(b.String(), ".") // 剥前导点防藏档
	if name == "" {
		name = "file"
	}
	if utf8.RuneCountInString(name) > inboxMaxName {
		rs := []rune(name)
		name = string(rs[:inboxMaxName])
	}
	return name
}

func mountInbox(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("POST /inbox", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, inboxMaxBytes)
		if err := r.ParseMultipartForm(inboxMaxBytes); err != nil {
			inboxJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"ok": false, "err": "超限或形错"})
			return
		}
		f, hdr, err := r.FormFile("file")
		if err != nil {
			inboxJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "err": "file 域缺"})
			return
		}
		defer f.Close()
		dir := inboxDir()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			inboxJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "err": "inbox 目录造败"})
			return
		}
		name := time.Now().UTC().Format("20060102-150405") + "_" + safeInboxName(hdr.Filename)
		dst := filepath.Join(dir, name)
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			inboxJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "err": "落件败"})
			return
		}
		n, cpErr := io.Copy(out, f)
		cErr := out.Close()
		if cpErr != nil || cErr != nil {
			inboxJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "err": "写件断"})
			return
		}
		// 簿记行审计(tolerate:账败不阻落件)
		if pool != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, _ = pool.Exec(ctx,
				`INSERT INTO ops_event(seat,event,ticket,note) VALUES('业主','投件','',$1)`,
				fmt.Sprintf("%s → %s (%dB)", safeInboxName(hdr.Filename), dst, n))
		}
		inboxJSON(w, http.StatusOK, map[string]any{"ok": true, "path": dst, "size": n})
	})
}

func inboxJSON(w http.ResponseWriter, code int, v map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
