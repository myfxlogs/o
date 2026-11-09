// mt5wrap —— app_secrets mt5.* 取密喂 mt5verify（凭据不出进程不落盘）。
// 用法: mt5wrap -verify /tmp/mt5verify [-realmut -mutsym XXXm]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

func main() {
	verify := flag.String("verify", "/tmp/mt5verify", "mt5verify 二进制径")
	show := flag.Bool("show", false, "印 mt5.server/host+login 尾位（密件不明文）")
	dsn := flag.String("dsn", "", "PG DSN（缺省 env FIN_PG_DSN/DB_DSN）")
	extra := flag.String("extra", "", "mt5verify 追加参（空格隔）")
	flag.Parse()
	d := *dsn
	if d == "" {
		d = os.Getenv("FIN_PG_DSN")
	}
	if d == "" {
		d = os.Getenv("DB_DSN")
	}
	if d == "" {
		fmt.Fprintln(os.Stderr, "dsn 缺（-dsn 或 FIN_PG_DSN/DB_DSN）")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, d)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pg 连败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	c := config.New(pool)
	get := func(k string) string {
		v, err := c.Secret(ctx, k)
		if err != nil || len(v) == 0 {
			fmt.Fprintf(os.Stderr, "secret %s 读败: %v\n", k, err)
			os.Exit(1)
		}
		return string(v)
	}
	if *show {
		lg := get("mt5.login")
		if len(lg) > 2 {
			lg = "***" + lg[len(lg)-2:]
		}
		pw := get("mt5.password")
		pwm := "(empty)"
		if len(pw) > 0 {
			pwm = fmt.Sprintf("%s…%s (len=%d)", pw[:1], pw[len(pw)-1:], len(pw))
		}
		fmt.Printf("mt5.login=%s mt5.password=%s mt5.server=%s mt5.host=%s\n",
			get("mt5.login"), pwm, get("mt5.server"), get("mt5.host"))
		return
	}
	args := []string{
		"-login", get("mt5.login"),
		"-pass", get("mt5.password"),
		"-server", get("mt5.server"),
		"-host", get("mt5.host"),
	}
	if *extra != "" {
		args = append(args, strings.Fields(*extra)...)
	}
	cmd := exec.Command(*verify, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mt5verify 退非零: %v\n", err)
		os.Exit(1)
	}
}
