// Events 经营事件报（T-0061：里程碑实时可见——非告警语义独立档）。
// 三事件起底：position_opened/funding_settled/position_closed；
// 事件驱动=ev_position/ev_cashflow NOTIFY 脊（T-0048 律禁周期环）；
// 独立开关 notify.feishu.events.enabled（键缺=false fail-closed 静默）；
// 同 id+kind 一键一推；推送败=计数+log 不递归告警；与告警文案分区 [经营事件]。
package notify

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EventPush 经营事件报推送件（sender 复用 Alerter 装配面——不新造通知栈）。
type EventPush struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
	mu     sync.Mutex
	sent   map[string]bool // "kind:id" 去重键
	cursor map[string]int64
}

// NewEventPush enabled/chatID 缺席=fail-closed（on=nil/false=只 log 不发）。
func NewEventPush(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *EventPush {
	return &EventPush{pool: pool, cli: cli, chatID: chatID, on: on,
		sent: map[string]bool{}, cursor: map[string]int64{}}
}

// Scan 一拍（事件驱动调用点）：三事件源各查增推——无新行零推送。
func (e *EventPush) Scan(ctx context.Context) error {
	if e.on == nil || !e.on() {
		return nil
	}
	if cliNil(e.cli) || e.chatID == nil || e.chatID() == "" {
		return nil // 缺凭据/chat_id=静默降级（不炸推送面）
	}
	e.pushOpened(ctx)
	e.pushFunding(ctx)
	e.pushClosed(ctx)
	return nil
}

// push 发送+去重（key=kind:id 一键一推——重注入不重复推）。
func (e *EventPush) push(ctx context.Context, key, text string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sent[key] {
		return
	}
	if _, err := e.cli.SendText(ctx, e.chatID(), "[经营事件] "+text); err != nil {
		log.Printf("[经营事件] %s 推送败（下拍重试）: %v", key, err)
		return // 败=下次事件拍重试，不计 sent（重试非刷屏——同 key 单推）
	}
	e.sent[key] = true
}

func (e *EventPush) pushOpened(ctx context.Context) {
	rows, err := e.pool.Query(ctx, `SELECT p.id,p.inst_id,p.notional_usd,p.opened_at,
		s.template_ref FROM fin_position p LEFT JOIN fin_strategy s ON s.id=p.strategy_ref
		WHERE p.opened_at > now()-interval '10 minutes' ORDER BY p.id`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var inst string
		var notional float64
		var at interface{}
		var tmpl *string
		if err := rows.Scan(&id, &inst, &notional, &at, &tmpl); err != nil {
			continue // 扫行败跳过非吞——行级病灶不阻余行
		}
		ref := "?"
		if tmpl != nil && *tmpl != "" {
			ref = *tmpl
		}
		e.push(ctx, fmt.Sprintf("opened:%d", id),
			fmt.Sprintf("开仓 %s notional=$%.2f 策略=%s（venue 腿行在案）", inst, notional, ref))
	}
}

func (e *EventPush) pushFunding(ctx context.Context) {
	if _, ok := e.cursor["funding"]; !ok {
		var maxID int64
		if err := e.pool.QueryRow(ctx,
			"SELECT COALESCE(max(id),0) FROM fin_cashflow WHERE kind='funding_settle'").Scan(&maxID); err == nil {
			e.cursor["funding"] = maxID // boot 播种基线——历史行不重推（重启防护）
		}
	}
	rows, err := e.pool.Query(ctx, `SELECT id,ccy,amount,position_ref FROM fin_cashflow
		WHERE kind='funding_settle' AND id>$1 ORDER BY id`, e.cursor["funding"])
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, posRef int64
		var ccy string
		var amt float64
		_ = rows.Scan(&id, &ccy, &amt, &posRef)
		e.cursor["funding"] = id
		e.push(ctx, fmt.Sprintf("funding:%d", id),
			fmt.Sprintf("funding 结算 %s %.4f（仓ref #%d）", ccy, amt, posRef))
	}
}

func (e *EventPush) pushClosed(ctx context.Context) {
	rows, err := e.pool.Query(ctx, `SELECT p.id,p.inst_id,p.notional_usd,
		COALESCE(p.settled_amount,0) FROM fin_position p
		WHERE p.closed_at IS NOT NULL AND p.closed_at > now()-interval '10 minutes' ORDER BY p.id`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var inst string
		var notional, pnl float64
		_ = rows.Scan(&id, &inst, &notional, &pnl)
		sign := "+"
		if pnl < 0 {
			sign = "-"
		}
		e.push(ctx, fmt.Sprintf("closed:%d", id),
			fmt.Sprintf("平仓 %s notional=$%.2f 净额=%s$%.4f", inst, notional, sign, absf(pnl)))
	}
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
