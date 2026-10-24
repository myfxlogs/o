// feishusend —— 通知面发送微件（T-0025 段三执行配套：核对点/晋升结果推业主飞书；
// feishuprobe 同族=T-0028 通知面工具族延伸）。用法：
//
//	DB_DSN=postgres://... PHASE_MASTER_KEY_FILE=./secrets/phase_master_key \
//	  go run ./tools/feishusend "文本"
//
// chat_id 读 app_config.feishu.chat_id（绑定态）；缺绑定/缺键=fail-closed 退非零。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/platform/notify"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, `用法: go run ./tools/feishusend "文本"（env: DB_DSN, PHASE_MASTER_KEY_FILE）`)
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dsn:", err)
		os.Exit(1)
	}
	defer pool.Close()
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	getSecret := func(k string) string {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			fmt.Fprintln(os.Stderr, k, err)
			os.Exit(1)
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			fmt.Fprintln(os.Stderr, k, err)
			os.Exit(1)
		}
		return p
	}
	var chatID string
	if err := pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key='feishu.chat_id'").Scan(&chatID); err != nil || chatID == "" {
		fmt.Fprintln(os.Stderr, "feishu.chat_id 未绑定（fail-closed 不推）")
		os.Exit(1)
	}
	cli := notify.NewFeishu(getSecret("feishu.app_id"), getSecret("feishu.app_secret"))
	msgID, err := cli.SendText(ctx, chatID, os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "send:", err)
		os.Exit(1)
	}
	fmt.Println("sent", msgID)
}
