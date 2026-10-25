// feishuprobe —— T-0028 实证件：凭据解密→ListChats 自动发现 chat_id→
// 落 feishu.chat_id 键→SendText 摘要实证（飞书侧 message_id 回执印行）。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"
)

func main() {
	ctx := context.Background()
	pool, _ := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	mk, err := config.ReadMasterKey()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	get := func(k string) string {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			fmt.Fprintln(os.Stderr, k, err)
			os.Exit(1)
		}
		p, _ := cryptoutil.Open(mk, ct)
		return p
	}
	cli := notify.NewFeishu(get("feishu.app_id"), get("feishu.app_secret"))
	chats, err := cli.ListChats(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chats:", err)
		os.Exit(1)
	}
	fmt.Printf("chats=%d\n", len(chats))
	for _, c := range chats {
		fmt.Printf("  chat=%s name=%q\n", c.ID, c.Name)
	}

	// chat_id 考古自 nox 备份 feishu_push_target 行（落 dev app_config 已先行）。
	var chatID string
	_ = pool.QueryRow(ctx, "SELECT value::text FROM app_config WHERE key='feishu.chat_id'").Scan(&chatID)
	chatID = trimJSONStr(chatID)
	mid, err := cli.SendText(ctx, chatID, "【O 推送通道实证】T-0028——chat_id 绑定+SendText 径走通（本行为 dev 实证消息）")
	if err != nil {
		fmt.Fprintln(os.Stderr, "send:", err)
		os.Exit(1)
	}
	fmt.Printf("SENT chat=%s msg=%s\n", chatID, mid)
}

func trimJSONStr(s string) string {
	if len(s) >= 2 && s[0] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
