// secput —— 秘件入库工具（运维件）：stdin 单行明文 → master key Seal →
// app_secrets upsert。红线：明文只进 stdin/内存/DB——零落 argv/stdout/
// log/文件（ps 面、shell history、日志面全不可见；输出只含键名+长度）。
// 用法：printf '%s' <值> | secput -key <app_secrets.key> [-dst <DSN>]
// （容器内径：docker exec -i o-core secput -key k；DSN 取 env DB_DSN 缺省）
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
)

func main() {
	key := flag.String("key", "", "app_secrets 键名（必填）")
	dst := flag.String("dst", "", "O 库 DSN（缺省取 env DB_DSN）")
	keyf := flag.String("mk", config.MasterKeyPath(), "master key 文件路径")
	flag.Parse()
	if *key == "" {
		fmt.Fprintln(os.Stderr, "-key 必填")
		os.Exit(2)
	}
	dsn := *dst
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DSN 缺（-dst 或 env DB_DSN）")
		os.Exit(2)
	}
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<16))
	if err != nil {
		die(err)
	}
	plain := strings.TrimSpace(string(raw))
	if plain == "" {
		fmt.Fprintln(os.Stderr, "stdin 空——秘件不入空值")
		os.Exit(2)
	}
	mk, err := os.ReadFile(*keyf)
	if err != nil {
		die(err)
	}
	mk = []byte(strings.TrimSpace(string(mk)))
	ct, err := cryptoutil.Seal(mk, plain)
	if err != nil {
		die(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		die(err)
	}
	defer pool.Close()
	tag, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
		ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, *key, ct)
	if err != nil {
		die(err)
	}
	fmt.Printf("sealed key=%s rows=%d\n", *key, tag.RowsAffected())
}

func die(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
