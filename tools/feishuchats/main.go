// feishuchats —— 群列微件（双频绑定位：业主建新群拉机器人后跑本件拿
// chat_id 写 feishu.sys_chat_id；ListChats 在册 API 面零新增）。
//
//	DB_DSN=postgres://... O_MASTER_KEY_FILE=./secrets/o_master_key \
//	  go run ./tools/feishuchats
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
	get := func(k string) string {
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
	cli := notify.NewFeishu(get("feishu.app_id"), get("feishu.app_secret"))
	chats, err := cli.ListChats(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "list:", err)
		os.Exit(1)
	}
	for _, c := range chats {
		fmt.Printf("%s\t%s\n", c.ID, c.Name)
	}
}
