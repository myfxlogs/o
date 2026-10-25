// feishumig —— T-0028 凭据迁移件：nox app_secrets FEISHU 行解密→o app_secrets
// feishu.app_id/app_secret 密文落册（密文进出，值不入会话/不入仓）。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

func main() {
	ctx := context.Background()
	src, _ := pgxpool.New(ctx, os.Getenv("SRC_DSN"))
	dst, _ := pgxpool.New(ctx, os.Getenv("DST_DSN"))
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	moved := 0
	for _, pair := range [][2]string{
		{"NOX_FEISHU_APP_ID", "feishu.app_id"},
		{"NOX_FEISHU_APP_SECRET", "feishu.app_secret"},
	} {
		var ct []byte
		if err := src.QueryRow(ctx, "SELECT value_enc FROM app_secrets WHERE key=$1", pair[0]).Scan(&ct); err != nil {
			fmt.Fprintln(os.Stderr, pair[0], "src read:", err)
			continue
		}
		plain, err := cryptoutil.Open(mk, ct)
		if err != nil {
			fmt.Fprintln(os.Stderr, pair[0], "decrypt:", err)
			continue
		}
		sealed, err := cryptoutil.Seal(mk, plain)
		if err != nil {
			fmt.Fprintln(os.Stderr, pair[0], "seal:", err)
			continue
		}
		if _, err := dst.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
			ON CONFLICT(key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, pair[1], sealed); err != nil {
			fmt.Fprintln(os.Stderr, pair[1], "dst write:", err)
			continue
		}
		moved++
	}
	fmt.Printf("moved=%d\n", moved)
}
