// alertprobe —— T-0030 实证件：凭据解密→Alerter 真推一条告警→飞书 message_id 回执。
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
	ctx := context.Background()
	pool, _ := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var idCt, secCt []byte
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	id, _ := cryptoutil.Open(mk, idCt)
	sec, _ := cryptoutil.Open(mk, secCt)
	cli := notify.NewFeishu(id, sec)
	var chat string
	_ = pool.QueryRow(ctx, "SELECT value::text FROM app_config WHERE key='feishu.chat_id'").Scan(&chat)
	if len(chat) >= 2 {
		chat = chat[1 : len(chat)-1]
	}
	mid, err := cli.SendText(ctx, chat, "[PHASE告警] alert_probe | T-0030 投递实证行（五事件源接源前实证）")
	if err != nil {
		fmt.Fprintln(os.Stderr, "send:", err)
		os.Exit(1)
	}
	fmt.Printf("ALERT-SENT msg=%s\n", mid)
}
