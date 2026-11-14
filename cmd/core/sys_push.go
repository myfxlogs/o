// sys_push.go —— 系统频富卡推面（业主令 2026-11-15：飞书双频——钱径文本报
// 走 feishu.chat_id 零改；系统运行情况富卡片走 feishu.sys_chat_id）。
// 件①在办活卡：ev_headless 敲门（fin_0183 在办/唤起/收口/果败/叙事族行落
// 即醒）→重算在办面→PATCH 同一张卡——「我现在正在做什么」=常驻活卡
// 原地刷新非消息流；msg_id 存 fin_push_ledger(sys_status/doing_card)——
// 重启续 PATCH 不重发新卡；PATCH 败（卡被删）=新发换 mid 自愈。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
)

type sysPush struct {
	pool *pgxpool.Pool
	cfg  *config.Client
	cli  *notify.FeishuClient
}

// startSysPush 装配（cli nil=凭据缺席整件降级不炸——同 alerter fail-closed 律；
// ev_headless 事件脊钦定径非轮询）。
func startSysPush(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, cli *notify.FeishuClient) {
	if cli == nil {
		return
	}
	sp := &sysPush{pool: pool, cfg: cfg, cli: cli}
	go events.ListenLoop(ctx, pool, "sys_push", []string{"ev_headless"}, sp.push)
}

// push 一敲一刷。fail-closed 两闸：enabled 关 / sys_chat_id 空=不推不炸。
func (s *sysPush) push(ctx context.Context) error {
	if en, _ := s.cfg.GetBool("notify.feishu.sys.enabled"); !en {
		return nil
	}
	chat, _ := s.cfg.GetString("feishu.sys_chat_id")
	if chat == "" {
		return nil
	}
	rows, err := queryDoingRows(ctx, s.pool)
	if err != nil {
		return err
	}
	card := notify.StatusCard(s.cardLines(ctx, rows), anyFresh(rows))
	if mid := s.cardMID(ctx); mid != "" {
		if err := s.cli.PatchCard(ctx, mid, card); err == nil {
			return nil
		} // PATCH 败=卡已删/旧mid失效→下面新发换锚
	}
	mid, err := s.cli.CreateInteractive(ctx, chat, card)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO fin_push_ledger(scope,key,msg_id)
		VALUES('sys_status','doing_card',$1)
		ON CONFLICT (scope,key) DO UPDATE SET msg_id=EXCLUDED.msg_id,pushed_at=now()`, mid)
	return err
}

// cardMID 活卡锚（fin_push_ledger 幂等锚面复用——重启首拍续 PATCH 不重发）。
func (s *sysPush) cardMID(ctx context.Context) string {
	var mid string
	_ = s.pool.QueryRow(ctx,
		`SELECT msg_id FROM fin_push_ledger WHERE scope='sys_status' AND key='doing_card'`).Scan(&mid)
	return mid
}

func anyFresh(rs []doingRowT) bool {
	for _, r := range rs {
		if r.fresh {
			return true
		}
	}
	return false
}

// cardLines 卡体行：在办明细（态/unit/声明/龄）+分隔+末场收口+末叙事+更时。
func (s *sysPush) cardLines(ctx context.Context, rs []doingRowT) []string {
	var ls []string
	if len(rs) == 0 {
		ls = append(ls, "**下班——无活在办**")
	}
	for _, r := range rs {
		tag := "在干"
		if !r.fresh {
			tag = "悬账"
		}
		ls = append(ls, fmt.Sprintf("**%s** `%s`\n%s（%s）",
			tag, r.unit, r.note, agoS(time.Since(r.ts))))
	}
	ls = append(ls, "────────")
	if t := s.lastNote(ctx, `('会话收口','唤起果','唤起败')`); t != "" {
		ls = append(ls, "**末场** "+t)
	}
	if t := s.lastNote(ctx, `('叙事')`); t != "" {
		ls = append(ls, "**叙事** "+t)
	}
	ls = append(ls, fmt.Sprintf("_更新 %s UTC_", time.Now().UTC().Format("15:04:05")))
	return ls
}

// lastNote 事件族末行 note（截 120 字防卡体臃肿）。败=空串静默不炸卡。
func (s *sysPush) lastNote(ctx context.Context, events string) string {
	var n string
	_ = s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT left(note,120) FROM ops_event WHERE event IN %s
		 ORDER BY id DESC LIMIT 1`, events)).Scan(&n)
	return strings.TrimSpace(n)
}

func agoS(d time.Duration) string {
	m := int(d.Minutes())
	if m < 60 {
		return fmt.Sprintf("%dm前", m)
	}
	return fmt.Sprintf("%dh%02dm前", m/60, m%60)
}
