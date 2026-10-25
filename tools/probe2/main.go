package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/trading/okxexec"
)

// probe2 —— dcd currency-pair/products 真参取证探针（T-0102 修补：拉密文
// 解密直打端点——uly 伪参 50014 vs baseCcy+quoteCcy+optType 三参实证对照）。
func main() {
	ctx := context.Background()
	mk, err := config.ReadMasterKeyFrom("/tmp/o_mk")
	die(err)
	var cred okxexec.Credentials
	b, _ := os.ReadFile("/tmp/okx_ct.txt")
	for _, ln := range strings.Split(string(b), "\n") {
		kv := strings.SplitN(ln, "|", 2)
		if len(kv) != 2 {
			continue
		}
		ct, _ := hex.DecodeString(strings.TrimSpace(kv[1]))
		p, err := cryptoutil.Open(mk, ct)
		die(err)
		switch strings.TrimSpace(kv[0]) {
		case "okx.api_key":
			cred.Key = p
		case "okx.api_secret":
			cred.Secret = p
		case "okx.passphrase":
			cred.Passphrase = p
		}
	}
	cli := okxexec.NewClient(cred, false)
	for _, path := range []string{
		"/api/v5/finance/sfp/dcd/products?baseCcy=BTC&quoteCcy=USDT&optType=C",
	} {
		raw, err := cli.Get(ctx, path)
		fmt.Println("==", path, "err:", err)
		if raw != nil {
			fmt.Println(string(raw)[:600])
		}
	}
}
func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
}
