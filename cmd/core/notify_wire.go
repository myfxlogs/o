// notify_wire.go —— 飞书告警装配件（棘轮拆出：sender/alerter/双频/系统活卡
// 一线牵——main.go 棘轮基线守）。双频=业主令 2026-11-15：钱径照走
// feishu.chat_id，系统类告警+在办富卡走 feishu.sys_chat_id。
package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
)

// wireNotify 装 sender/alerter/sys_push（凭据缺席=sender nil 降级——
// typed-nil 主修 T-0032：缺件传真 nil 接口非 (*FeishuClient)(nil)）。
func wireNotify(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (notify.Sender, *notify.Alerter) {
	var sender notify.Sender
	if c := feishuSender(ctx, pool); c != nil {
		sender = c
	}
	sysCli := feishuSysSender(ctx, pool) // 第二机器人（meta 挪用——nil=降级主cli）
	al := notify.NewAlerter(sender, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v }).
		WithSysChat(func() string { // 系统类告警/在办富卡分频
			v, _ := cfg.GetString("feishu.sys_chat_id")
			return v
		}).WithSysSender(sysCli)
	startSysPush(ctx, pool, cfg, sysCli)  // 系统频在办活卡（ev_headless 脊——sys_push.go）
	startFeedPush(ctx, pool, cfg, sysCli) // 系统频动作流水（ev_feed 脊——feed_push.go）
	return sender, al
}
