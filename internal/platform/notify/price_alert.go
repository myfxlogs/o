// price_alert.go —— T-0382 价格提醒拍（业主令 2026-11-06：「预设一个价格，
// 飞书通知我」）：fin_price_alert active 行×md_ticks 最新价命中→飞书推→
// state 翻 fired+ops_event 落行。一发即锁=UPDATE ... WHERE state='active'
// 条件翻位（重拍/并发天然幂等——命中集再扫零行）。单向告警非条件单——
// 票面红线零交易挂钩。
// 败径照 T-0369 容错律：推送败=行不翻 fired（票不丢，下帧重试）+升件自报
// 行（24h 同窗去重不暴——hungerEscalate 同构）；Scan 查询败=自报+上抛由
// 拍壳 log。sender 缺席（凭据未落）按关态处理——只翻位不发不空指针。
package notify

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PriceAlert 价格提醒拍件（sender/chatID/on 复用 venue_req 同款装配面）。
type PriceAlert struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
}

// NewPriceAlert enabled/chatID 缺席=fail-closed（on=nil/false=只翻位不发——
// venue_req 关态落锚同语义；cli=nil 凭据缺席同按关态）。
func NewPriceAlert(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *PriceAlert {
	return &PriceAlert{pool: pool, cli: cli, chatID: chatID, on: on}
}

// 拍行席别/环票号（hunger 拍同款口径——ops_event 落行可溯）。
const (
	priceAlertSeat   = "提醒拍"
	priceAlertEvent  = "价格提醒"
	priceAlertTicket = "T-0382"
)

type alertHit struct {
	id     int64
	symbol string
	dir    string
	target float64
	last   float64
}

// Scan 一拍（ev_ticks 帧调用点）：active 行 JOIN 每 symbol 最新 tick 集
// （DISTINCT ON 防 venue 扩面重行——现单 venue okx），命中即逐行 fire——
// 零命中零动作。
func (p *PriceAlert) Scan(ctx context.Context) error {
	rows, err := p.pool.Query(ctx, `SELECT a.id,a.symbol,a.direction,a.target::float8,t.last::float8
		FROM fin_price_alert a
		JOIN (SELECT DISTINCT ON (inst_id) inst_id,last FROM md_ticks
			WHERE last IS NOT NULL ORDER BY inst_id,ts DESC) t
			ON t.inst_id=a.symbol
		WHERE a.state='active'`)
	if err != nil {
		p.escalate(ctx, "scan", err)
		return err
	}
	defer rows.Close()
	var hits []alertHit
	for rows.Next() {
		var h alertHit
		if err := rows.Scan(&h.id, &h.symbol, &h.dir, &h.target, &h.last); err != nil {
			return err
		}
		if (h.dir == ">=" && h.last >= h.target) || (h.dir == "<=" && h.last <= h.target) {
			hits = append(hits, h)
		}
	}
	if err := rows.Err(); err != nil {
		p.escalate(ctx, "scan", err)
		return err
	}
	for _, h := range hits {
		p.fire(ctx, h)
	}
	return nil
}

// fire 单行命中：推（关态/凭据缺=不发）→条件翻位 fired→ops_event 行。
// 推送败=自报+不翻位（票不丢——下帧重试）；翻位零行=并发/重拍已锁（不重响）。
func (p *PriceAlert) fire(ctx context.Context, h alertHit) {
	msg := fmt.Sprintf("【价格提醒】%s 现价 %s %s 目标价 %s",
		h.symbol, numStr(h.last), h.dir, numStr(h.target))
	pushed := false
	if p.cli == nil || p.on == nil || !p.on() {
		log.Printf("[price_alert] 推送关/凭据缺——%s 命中（不发只翻位）", h.symbol)
	} else if _, err := p.cli.SendText(ctx, p.chatID(), msg); err != nil {
		log.Printf("[price_alert] 推送败（行不翻 fired——下帧重试）: %v", err)
		p.escalate(ctx, "push", err)
		return
	} else {
		pushed = true
	}
	tag, err := p.pool.Exec(ctx, `UPDATE fin_price_alert
		SET state='fired',fired_at=now() WHERE id=$1 AND state='active'`, h.id)
	if err != nil {
		p.escalate(ctx, "fired 翻位", err)
		return
	}
	if tag.RowsAffected() == 0 {
		return // 并发先响/手翻——一发即锁语义不重复落行
	}
	note := fmt.Sprintf("#%d %s 现价 %s %s 目标价 %s——%s",
		h.id, h.symbol, numStr(h.last), h.dir, numStr(h.target),
		map[bool]string{true: "已推送", false: "推送关只翻位"}[pushed])
	if _, err := p.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,$3,$4)`, priceAlertSeat, priceAlertEvent, priceAlertTicket, note); err != nil {
		log.Printf("[price_alert] ops_event 行写败（翻位已成——自报痕缺）: %v", err)
	}
}

// escalate 败情自报升件行（T-0369 容错必报——hungerEscalate 同构）：写败/
// NOTIFY-PRICE-ALERT 判例键，note 前缀同窗 24h 至多一行不暴；自报行自身
// 写败=静默退（升件不能再升——结构性限度在案）。
func (p *PriceAlert) escalate(ctx context.Context, phase string, cause error) {
	prefix := "price_alert " + phase + " 败"
	if _, err := p.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'写败','NOTIFY-PRICE-ALERT'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE $4 AND ts>now()-interval '24 hours')`,
		priceAlertSeat, priceAlertTicket, fmt.Sprintf("%s: %v", prefix, cause), prefix+"%"); err != nil {
		log.Printf("[price_alert] 自报行落败（结构性限度静默退）: %v（原败: %v）", err, cause)
	}
}

// numStr 价格人串（整数去尾零、小数保精度——拒科学计数法黑话）。
func numStr(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
