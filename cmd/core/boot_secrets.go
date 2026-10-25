// boot_secrets.go —— boot 期凭据解密装配（feishu 通知面——自主拆分，行数棘轮）。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/notify"
)

// feishuSender feishu 告警通道装配（app_secrets 密文列解；缺件=nil=静默降级——
// typed-nil 主修 T-0032：接口装类型化 nil 指针时 ==nil 判 false，返真 nil 接口）。
func feishuSender(ctx context.Context, pool *pgxpool.Pool) *notify.FeishuClient {
	mk, _ := config.ReadMasterKey()
	var idCt, secCt []byte
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_id'").Scan(&idCt); err != nil {
		return nil
	}
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key='feishu.app_secret'").Scan(&secCt); err != nil {
		return nil
	}
	id, e1 := cryptoutil.Open(mk, idCt)
	sec, e2 := cryptoutil.Open(mk, secCt)
	if e1 != nil || e2 != nil {
		return nil
	}
	return notify.NewFeishu(id, sec)
}
