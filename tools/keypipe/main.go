// keypipe —— T-0184 私钥管道化直传件（一次性运维）：
// stdin=nox 库 app_secrets TRX_VOTE_PRIVATE_KEY 密文 hex → nox mk 解密
// → O master key 重封 → O app_secrets 键 trx.vote_private_key。
// 红线：明文只进 DB/内存，零落 stdout/log/文件（存在性输出不含值）。
// T-0192 返修③：nox mk 改文件径（-noxkey）——env 键不入白名单面。
// 用法：echo <hex> | go run tools/keypipe -dst <DSN> -key <O mk 文件> -noxkey <nox mk 文件>
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/cryptoutil"
)

func main() {
	dst := flag.String("dst", "", "O 库 DSN")
	keyf := flag.String("key", "/opt/o/secrets/o_master_key", "O master key 文件")
	skeyf := flag.String("noxkey", "", "nox master key 文件径")
	skey := flag.String("skey", "trx.vote_private_key", "目标 app_secrets 键名")
	flag.Parse()
	if *dst == "" || *skeyf == "" {
		fmt.Fprintln(os.Stderr, "-dst/-noxkey 必填")
		os.Exit(2)
	}
	raw, _ := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	hexStr := strings.TrimSpace(string(raw))
	hexStr = strings.TrimPrefix(hexStr, "\\x")
	enc, err := hex.DecodeString(hexStr)
	if err != nil || len(enc) == 0 {
		die(fmt.Errorf("stdin hex 解败"))
	}
	nkb, err := os.ReadFile(*skeyf)
	if err != nil {
		die(fmt.Errorf("nox mk 文件读败"))
	}
	plain, err := cryptoutil.Open([]byte(strings.TrimSpace(string(nkb))), enc)
	if err != nil {
		die(fmt.Errorf("nox 解密败（key/密文不配）"))
	}
	mk, err := os.ReadFile(*keyf)
	if err != nil {
		die(err)
	}
	reEnc, err := cryptoutil.Seal([]byte(strings.TrimSpace(string(mk))), plain)
	plain = ""
	if err != nil {
		die(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dst)
	if err != nil {
		die(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
		ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext,updated_at=now()`,
		*skey, reEnc); err != nil {
		die(err)
	}
	fmt.Println("OK: app_secrets 写入存在性验（值零回显）")
}
func die(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
