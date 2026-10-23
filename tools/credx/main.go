// credx —— E-1 凭据提取工具（一次性运维件）：nox dump 库 exchange_configs okx 行
// *_enc 密文列解密 → 同构 Seal → 写 PHASE app_secrets + exchange_configs。
// 禁线：任何明文只进 DB/内存，零落 stdout/log/文件。
// 用法：credx -src <noxdump DSN> -dst <phase DSN> -key <master key 文件>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
)

func main() {
	src := flag.String("src", "", "nox dump DSN")
	dst := flag.String("dst", "", "phase DSN")
	keyf := flag.String("key", config.MasterKeyPath(), "master key 文件路径")
	flag.Parse()
	if *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "-src/-dst 必填")
		os.Exit(2)
	}
	mk, err := os.ReadFile(*keyf)
	if err != nil {
		die(err)
	}
	mk = []byte(string(mk[:len(mk)-1])) // 尾换行
	ctx := context.Background()
	sp, err := pgxpool.New(ctx, *src)
	if err != nil {
		die(err)
	}
	defer sp.Close()
	dp, err := pgxpool.New(ctx, *dst)
	if err != nil {
		die(err)
	}
	defer dp.Close()

	var ke, se, pe []byte
	err = sp.QueryRow(ctx, `SELECT api_key_enc, api_secret_enc, passphrase_enc
		FROM exchange_configs WHERE venue='okx' AND venue_type='unified' LIMIT 1`).Scan(&ke, &se, &pe)
	if err != nil {
		// unified 行缺→任一 enc 非空行
		err = sp.QueryRow(ctx, `SELECT api_key_enc, api_secret_enc, passphrase_enc
			FROM exchange_configs WHERE venue='okx' AND api_key_enc IS NOT NULL LIMIT 1`).Scan(&ke, &se, &pe)
	}
	if err != nil {
		die(fmt.Errorf("okx enc 行读取: %w", err))
	}
	plain := map[string]string{}
	for k, blob := range map[string][]byte{"okx.api_key": ke, "okx.api_secret": se, "okx.passphrase": pe} {
		if len(blob) == 0 {
			die(fmt.Errorf("%s 密文列为空", k))
		}
		p, err := cryptoutil.Open(mk, blob)
		if err != nil {
			die(fmt.Errorf("%s 解密: %w", k, err))
		}
		plain[k] = p
	}
	for k, p := range plain {
		ct, err := cryptoutil.Seal(mk, p)
		if err != nil {
			die(err)
		}
		if _, err := dp.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext)
			VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, k, ct); err != nil {
			die(fmt.Errorf("app_secrets 写 %s: %w", k, err))
		}
		fmt.Printf("extracted %s -> app_secrets（密文 %dB，零明文凭据出径）\n", k, len(ct))
	}
	fmt.Println("E-1 提取完成——下一步探针翻已验")
}

func die(err error) { fmt.Fprintln(os.Stderr, "FAIL:", err); os.Exit(1) }
