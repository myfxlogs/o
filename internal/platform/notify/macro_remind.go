// macro_remind.go —— T-0410 seed→提醒件：md_macro_event 今/明日 event_date
// 行日拍扫→飞书提醒（venue_req/price_alert 同款装配面——Sender+chatID+
// enabled 热键，禁旁路裸写）。病灶原型：seed 行在库（NFP 占位）而零件把它
// 转成「今晚有事件」推送——知识在库里提醒径断。时刻列缺=「日级精度」如实标。
// 已提醒锚=ops_event `宏观提醒` 行（键=event_date——送达才记账 T-0093 序：
// 推败/凭据缺不占位，下拍重试）；败径 escalate 自报 24h 同窗去重不暴。
// 关态=不扫不记零行（复归后窗内行照推——不占位语义同 price_alert 关态翻位
// 异构：提醒是时敏件，关态期窗过即作废不补）。
package notify

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MacroRemind 宏观事件提醒拍件（sender/chatID/on 复用 price_alert 同装配面）。
type MacroRemind struct {
	pool   *pgxpool.Pool
	cli    Sender
	chatID func() string
	on     func() bool
}

// NewMacroRemind enabled/chatID 缺席=fail-closed（on=nil/false=不扫不记；
// cli/chatID 缺席形=凭据缺升件自报不记账——下拍重试不占位）。
func NewMacroRemind(pool *pgxpool.Pool, cli Sender, chatID func() string, on func() bool) *MacroRemind {
	return &MacroRemind{pool: pool, cli: cli, chatID: chatID, on: on}
}

// 拍行席别/事件/环票号/判例域（price_alert 同款口径——ops_event 落行可溯）。
const (
	macroRemindSeat   = "提醒拍"
	macroRemindEvent  = "宏观提醒"
	macroRemindTicket = "T-0410"
	macroRemindScope  = "NOTIFY-MACRO-REMIND"
)

// Scan 一拍（日拍 ticker 调用点）：md_macro_event event_date IN(今日,明日)
// 行逐条——已提醒锚在=跳过；未锚=推→锚落账。零命中零动作。
func (m *MacroRemind) Scan(ctx context.Context) error {
	if m.on == nil || !m.on() {
		return nil // 关态不扫不记——复归后窗内行照推
	}
	rows, err := m.pool.Query(ctx, `SELECT event_date,kind,title,(event_date=current_date) FROM md_macro_event
		WHERE event_date IN (current_date,current_date+1) ORDER BY event_date`)
	if err != nil {
		m.escalate(ctx, "scan", err)
		return err
	}
	defer rows.Close()
	type ev struct {
		date  time.Time
		kind  string
		title string
		today bool
	}
	var evs []ev
	for rows.Next() {
		var e ev
		if err := rows.Scan(&e.date, &e.kind, &e.title, &e.today); err != nil {
			return err
		}
		evs = append(evs, e)
	}
	if err := rows.Err(); err != nil {
		m.escalate(ctx, "scan", err)
		return err
	}
	for _, e := range evs {
		m.remind(ctx, e.date, e.kind, e.title, e.today)
	}
	return nil
}

// remind 单行提醒：锚查→推→锚落账。推送败/凭据缺=自报+不落锚（票不丢
// 下拍重试——T-0093 送达才记账同序）。
func (m *MacroRemind) remind(ctx context.Context, date time.Time, kind, title string, today bool) {
	key := date.Format("2006-01-02")
	var exists bool
	if err := m.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event=$1 AND note LIKE $2)`,
		macroRemindEvent, "键="+key+"|%").Scan(&exists); err != nil {
		m.escalate(ctx, "锚查", err)
		return
	}
	if exists {
		return // 一事件一提醒幂等锚——日拍/重启不重推
	}
	when := "明日"
	if today {
		when = "今日"
	}
	// 时刻列缺=日级精度如实标（票面钦）；标签=risk-event 窗标记供策略层
	// 消费挂 note（可选延拓——策略侧消费改造不在本票）。
	msg := fmt.Sprintf("[宏观提醒] %s %s %s %s——日级精度（种子历无时刻）",
		when, key, kind, title)
	if m.cli == nil || m.chatID == nil || m.chatID() == "" {
		m.escalate(ctx, "凭据缺", errors.New("feishu sender/chat_id 缺席——不落锚下拍重试"))
		return
	}
	if _, err := m.cli.SendText(ctx, m.chatID(), msg); err != nil {
		log.Printf("[宏观提醒] %s|%s 推送败（不落锚下拍重试）: %v", key, kind, err)
		m.escalate(ctx, "推送", err)
		return
	}
	if _, err := m.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,$3,$4)`, macroRemindSeat, macroRemindEvent, macroRemindTicket,
		fmt.Sprintf("键=%s|%s %s %s（日级精度——种子历无时刻）标签=risk-event 已推送",
			key, kind, when, title)); err != nil {
		log.Printf("[宏观提醒] %s|%s 锚行写败（已送达——重拍或重推，幂等锚缺结构性限度）: %v", key, kind, err)
	}
}

// escalate 败情自报升件行（T-0369 容错必报——price_alert 同构）：读败/
// NOTIFY-MACRO-REMIND 判例键，note 前缀同窗 24h 至多一行不暴；自报行自身
// 写败=静默退（升件不能再升——结构性限度在案）。
func (m *MacroRemind) escalate(ctx context.Context, phase string, cause error) {
	prefix := "macro_remind " + phase + " 败"
	if _, err := m.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'读败',$4
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE $5 AND ts>now()-interval '24 hours')`,
		macroRemindSeat, macroRemindTicket, fmt.Sprintf("%s: %v", prefix, cause),
		macroRemindScope, prefix+"%"); err != nil {
		log.Printf("[宏观提醒] 自报行落败（结构性限度静默退）: %v（原败: %v）", err, cause)
	}
}
