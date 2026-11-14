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
	return feishuKeySender(ctx, pool, "feishu.app_id", "feishu.app_secret")
}

// feishuSysSender 系统频第二机器人装配（feishu_sys.*——业主令 2026-11-15
// 双频：meta 飞书挪作系统通道；缺件=nil=系统频降级主频兜底零漏件）。
func feishuSysSender(ctx context.Context, pool *pgxpool.Pool) *notify.FeishuClient {
	return feishuKeySender(ctx, pool, "feishu_sys.app_id", "feishu_sys.app_secret")
}

func feishuKeySender(ctx context.Context, pool *pgxpool.Pool, idKey, secKey string) *notify.FeishuClient {
	mk, _ := config.ReadMasterKey()
	var idCt, secCt []byte
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", idKey).Scan(&idCt); err != nil {
		return nil
	}
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", secKey).Scan(&secCt); err != nil {
		return nil
	}
	id, e1 := cryptoutil.Open(mk, idCt)
	sec, e2 := cryptoutil.Open(mk, secCt)
	if e1 != nil || e2 != nil {
		return nil
	}
	return notify.NewFeishu(id, sec)
}
