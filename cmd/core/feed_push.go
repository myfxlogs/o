// feed_push.go —— 系统频动作流水推面（业主令 2026-11-15④：O 每个动作更新
// 都发飞书=「我在做什么」消息流；在办活卡=此刻快照，本件=逐条时间线）。
// ev_feed 事件脊钦定径非轮询（fin_0185：动作行落库即醒）。突发合流：
// 醒后短窗再收一轮——会话簇写多行合一发防刷屏；水印 fin_push_ledger
// (sys_status/action_feed)——重启续水位不重发不漏发；首启锚定当下不回放史。
package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/events"
	"o/internal/platform/notify"
)

type feedPush struct {
	pool     *pgxpool.Pool
	cfg      *config.Client
	cli      *notify.FeishuClient
	lastPush time.Time // 攒批窗锚——送成才记（发败不锚,下醒重泄同批）
}

type feedRow struct {
	id    int64
	seat  string
	event string
	note  string
}

// startFeedPush 装配（cli nil=凭据缺席整件降级不炸——同 sys_push 律）。
func startFeedPush(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, cli *notify.FeishuClient) {
	if cli == nil {
		log.Printf("[feed_push] 系统频凭据缺席——动作流水降级不启（活卡/告警径不受影响）")
		return
	}
	fp := &feedPush{pool: pool, cfg: cfg, cli: cli}
	go events.ListenLoop(ctx, pool, "feed_push", []string{"ev_feed"}, fp.push)
}

// feedQuietGap 攒批窗——非穿透件攒 10min 一批（事驱推吵坑 2026-11-15:
// 写账即推,会话活跃=刷屏;警报/判词/部署/自愈穿透窗即时推,余攒批）。
const feedQuietGap = 10 * time.Minute

// feedPierce 穿透事件集——窗内到即推不等攒（要事不延迟）。
var feedPierce = "('升件','判词','部署','恢复')"

// push 一醒一泄：短窗合流→攒批闸→水印后新动作行逐条人话渲染合一发→送成才过水印。
func (s *feedPush) push(ctx context.Context) error {
	if en, _ := s.cfg.GetBool("notify.feishu.sys.enabled"); !en {
		return nil
	}
	chat, _ := s.cfg.GetString("feishu.sys_chat_id")
	if chat == "" {
		return nil
	}
	time.Sleep(6 * time.Second) // 突发合流窗——簇写行聚一批（sched-exempt-ok 事驱背压窗）
	wm, err := s.watermark(ctx)
	if err != nil {
		return err
	}
	if time.Since(s.lastPush) < feedQuietGap {
		pierce, err := s.hasPierce(ctx, wm)
		if err != nil {
			return err
		}
		if !pierce {
			return nil // 攒批窗内非穿透件=攒着不过水印,下醒合批泄
		}
	}
	if wm == 0 { // 首启不回放史——锚定当下 max(id)
		return s.initWM(ctx)
	}
	for drain := 0; drain < 5; drain++ { // 每拍至多泄 5 批防长尾阻塞听环
		rows, err := s.pending(ctx, wm)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		var b strings.Builder
		b.WriteString("[O·动]\n")
		if f := s.feedFrame(ctx); f != "" { // 在办帧冠首——「此刻谁在干什么」白话解说
			b.WriteString(f + "\n———\n")
		}
		for _, r := range rows {
			fmt.Fprintf(&b, "%s #%d %s\n", feedHead(r.seat, r.event), r.id, feedTidy(r.note))
		}
		if _, err := s.cli.SendText(ctx, chat, strings.TrimRight(b.String(), "\n")); err != nil {
			return err // 发败=不过水印，下醒重泄同批（不漏件）
		}
		s.lastPush = time.Now()
		wm = rows[len(rows)-1].id
		if err := s.setWM(ctx, wm); err != nil {
			return err
		}
	}
	return nil
}

// feedExclude 与 fin_0185 触发器 WHEN 同集（双写一致件——剔集单边改=断流坑）。
const feedExclude = "('量表','唤起','唤起果','唤起败','条件请求','求助值守')"

func (s *feedPush) pending(ctx context.Context, wm int64) ([]feedRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id,seat,event,left(note,500) FROM ops_event
		 WHERE id>$1 AND event NOT IN `+feedExclude+`
		 ORDER BY id LIMIT 40`, wm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []feedRow
	for rows.Next() {
		var r feedRow
		if err := rows.Scan(&r.id, &r.seat, &r.event, &r.note); err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// hasPierce 攒批窗穿透判——水印后有要事件=真（窗内不等攒即时泄）。
func (s *feedPush) hasPierce(ctx context.Context, wm int64) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event
		 WHERE id>$1 AND event IN `+feedPierce+`
		 AND event NOT IN `+feedExclude, wm).Scan(&n)
	return n > 0, err
}

func (s *feedPush) watermark(ctx context.Context) (int64, error) {
	var v string
	err := s.pool.QueryRow(ctx,
		`SELECT msg_id FROM fin_push_ledger WHERE scope='sys_status' AND key='action_feed'`).Scan(&v)
	if err != nil {
		return 0, nil // 无锚=首启
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, nil
	}
	return n, nil
}

func (s *feedPush) initWM(ctx context.Context) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO fin_push_ledger(scope,key,msg_id)
		 SELECT 'sys_status','action_feed',COALESCE((SELECT max(id) FROM ops_event),0)::text
		 ON CONFLICT (scope,key) DO NOTHING`)
	return err
}

func (s *feedPush) setWM(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO fin_push_ledger(scope,key,msg_id) VALUES('sys_status','action_feed',$1)
		 ON CONFLICT (scope,key) DO UPDATE SET msg_id=EXCLUDED.msg_id,pushed_at=now()`,
		strconv.FormatInt(id, 10))
	return err
}

var feedWS = strings.NewReplacer("\n", " ", "\r", " ")

// feedTidy 动作行人话渲染（k=v 头段中化+unit 短名同活卡层；换行拍平保一行一件）。
func feedTidy(note string) string {
	return feedWS.Replace(feedGloss.Replace(strings.TrimSpace(humanizeKV(note))))
}
