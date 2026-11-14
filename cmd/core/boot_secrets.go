// boot_secrets.go —— boot 期凭据解密装配（feishu 通知面——自主拆分，行数棘轮）。
package main

import (
	"context"
	"log"

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

// feishuKeySender 双轨共享装配——降级必留声（坑#16928 实证：封钥归一错→
// nil 静默→sys_push 哑死 40min 零痕迹；败因分记=下次同类病灶秒见）。
func feishuKeySender(ctx context.Context, pool *pgxpool.Pool, idKey, secKey string) *notify.FeishuClient {
	mk, err := config.ReadMasterKey()
	if err != nil {
		log.Printf("[boot] 主权钥读败 %v——sender(%s) 降级缺席", err, idKey)
		return nil
	}
	var idCt, secCt []byte
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", idKey).Scan(&idCt); err != nil {
		log.Printf("[boot] %s 密件读败 %v——sender 降级缺席", idKey, err)
		return nil
	}
	if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", secKey).Scan(&secCt); err != nil {
		log.Printf("[boot] %s 密件读败 %v——sender 降级缺席", secKey, err)
		return nil
	}
	id, e1 := cryptoutil.Open(mk, idCt)
	sec, e2 := cryptoutil.Open(mk, secCt)
	if e1 != nil || e2 != nil {
		log.Printf("[boot] %s/%s 解封败(%v / %v)——sender 降级缺席（封钥归一疑）", idKey, secKey, e1, e2)
		return nil
	}
	return notify.NewFeishu(id, sec)
}
