// feishu-notify —— 调度环事件推飞书薄 CLI（T-0061 同栈复用：app_secrets 密文
// 解凭据 + FeishuClient SendText，零新凭据面零新密钥）。relay/bash 调用：
//
//	feishu-notify [-env env文件] [-mk 主权钥文件] "文本"
//
// 任何一环缺（DSN/钥/凭据/chat_id）=stderr 记因退出非零，调用方 best-effort
// 语义（推送败不炸环——devin-relay nf() 已兜）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"
)

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "feishu-notify:", err)
		os.Exit(1)
	}
}

// envDSN 从 env 文件取 KEY=VALUE（只读不回显值——凭据零入日志零入 stderr）。
func envDSN(path, key string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("env 读败 %s: %w", path, err)
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(ln, key+"=") {
			return strings.TrimPrefix(ln, key+"="), nil
		}
	}
	return "", fmt.Errorf("env %s 缺 %s", path, key)
}

func main() {
	env := flag.String("env", "/opt/o-research/env", "env 文件（取 FIN_PG_DSN）")
	mkf := flag.String("mk", "/opt/o/secrets/o_master_key", "主权钥文件")
	tmo := flag.Duration("t", 20*time.Second, "总时限")
	flag.Parse()
	text := strings.Join(flag.Args(), " ")
	if text == "" {
		fmt.Fprintln(os.Stderr, "用法: feishu-notify 文本")
		os.Exit(2)
	}
	dsn, err := envDSN(*env, "FIN_PG_DSN")
	die(err)
	mk, err := config.ReadMasterKeyFrom(*mkf)
	die(err)
	ctx, cancel := context.WithTimeout(context.Background(), *tmo)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	die(err)
	defer pool.Close()
	var idCt, secCt []byte
	die(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt))
	die(pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt))
	id, err := cryptoutil.Open(mk, idCt)
	die(err)
	sec, err := cryptoutil.Open(mk, secCt)
	die(err)
	cfg := config.New(pool)
	die(cfg.Load(ctx)) // 快照装载（boot 步——GetString 只读快照）
	chatID, err := cfg.GetString("feishu.chat_id")
	die(err)
	if chatID == "" {
		die(fmt.Errorf("feishu.chat_id 空（未绑定）"))
	}
	cli := notify.NewFeishu(id, sec)
	if _, err := cli.SendText(ctx, chatID, text); err != nil {
		die(err)
	}
	fmt.Println("sent")
}
