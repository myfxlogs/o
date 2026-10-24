// okxlev — OKX 杠杆写面（set-leverage/leverage-info）真户实测探针。
//
// 测试谱：基线读→cross 往返→isolated posSide 方言→超档拒码→复原。
// 账户全平态执行，杠杆改档位对保证金影响=配置面非持仓面，逐步实读仲裁。
// 用法：okxlev -dsn <dsn> -key <keyfile> [-inst BTC-USDT-SWAP]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/trading/okxexec"
)

func jstr(raw json.RawMessage, keys ...string) string {
	var v any
	if json.Unmarshal(raw, &v) != nil || len(keys) == 0 {
		return ""
	}
	m, ok := v.([]any)
	if !ok || len(m) == 0 {
		return ""
	}
	mm, _ := m[0].(map[string]any)
	for _, k := range keys {
		if s, ok := mm[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func main() {
	dsn := flag.String("dsn", "", "phase DSN")
	keyf := flag.String("key", config.MasterKeyPath(), "")
	inst := flag.String("inst", "BTC-USDT-SWAP", "标的合约")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	mk, err := config.ReadMasterKeyFrom(*keyf)
	if err != nil {
		log.Fatalf("master key: %v", err)
	}
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	var creds okxexec.Credentials
	for k, dst := range map[string]*string{"okx.api_key": &creds.Key, "okx.api_secret": &creds.Secret, "okx.passphrase": &creds.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			log.Fatalf("secret %s: %v", k, err)
		}
		plain, err := cryptoutil.Open(mk, ct)
		if err != nil {
			log.Fatalf("decrypt %s: %v", k, err)
		}
		*dst = plain
	}
	cli := okxexec.NewClient(creds, false)
	fmt.Println("== 凭据解密 ok（值零回显）==")

	lvInfo := func(mgn, pos string) string {
		q := "/api/v5/account/leverage-info?instId=" + *inst + "&mgnMode=" + mgn
		if pos != "" {
			q += "&posSide=" + pos
		}
		raw, err := cli.Get(ctx, q)
		if err != nil {
			return "ERR:" + err.Error()
		}
		return jstr(raw, "lever") + "|pos=" + jstr(raw, "posSide")
	}
	set := func(mgn, pos string, lev int) string {
		body := map[string]any{"instId": *inst, "mgnMode": mgn, "lever": strconv.Itoa(lev)}
		if pos != "" {
			body["posSide"] = pos
		}
		b, _ := json.Marshal(body)
		raw, err := cli.Post(ctx, "/api/v5/account/set-leverage", b)
		if err != nil {
			return "ERR:" + err.Error()
		}
		return "ok lever=" + jstr(raw, "lever")
	}

	fmt.Printf("基线: cross=%s isolated=%s\n", lvInfo("cross", ""), lvInfo("isolated", ""))
	fmt.Println("--- cross 往返 3→5→3 ---")
	fmt.Println("set cross 5  :", set("cross", "", 5))
	fmt.Println("读回         :", lvInfo("cross", ""))
	fmt.Println("set cross 3  :", set("cross", "", 3))
	fmt.Println("读回         :", lvInfo("cross", ""))
	fmt.Println("--- isolated posSide 方言 ---")
	fmt.Println("set iso 10 无posSide:", set("isolated", "", 10))
	fmt.Println("set iso 10 long   :", set("isolated", "long", 10))
	fmt.Println("读回 iso          :", lvInfo("isolated", "long"))
	fmt.Println("set iso 3 long    :", set("isolated", "long", 3))
	fmt.Println("set iso 10 short  :", set("isolated", "short", 10))
	fmt.Println("读回 iso short    :", lvInfo("isolated", "short"))
	fmt.Println("set iso 3 short   :", set("isolated", "short", 3))
	fmt.Println("--- 边界 ---")
	fmt.Println("set cross 200(超档):", set("cross", "", 200))
	fmt.Println("set cross 0        :", set("cross", "", 0))
	fmt.Printf("终态: cross=%s isoLong=%s isoShort=%s\n",
		lvInfo("cross", ""), lvInfo("isolated", "long"), lvInfo("isolated", "short"))
	os.Exit(0)
}
