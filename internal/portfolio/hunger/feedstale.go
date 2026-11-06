// feedstale.go —— 日历 feed 死闸（T-0410）：「md_calendar_push 行龄」巡检件。
// 病灶原型：2026-11-06 非农夜直播 feed 断更 ~2 天零告警（md_calendar_push
// 末行=11-04，NFP 整个日历面零推送——与 dispatch-spawn-silent-failure 同族
// 「缺席无感」，靠业主人工对拍发现）。探针挂饥饿拍同槽（ProbeFamilies/
// ProbeDeps 同款隔离位）：notify.feishu.calendar.enabled 开态下
// md_calendar_push 最新行 sent_at 超阈（票面钦建议 24h——feed 本应日更）
// 零新行 →ops_event `feed断更` 行+升件（观测盲/CALENDAR-FEED 判例键，
// 24h 同窗去重）。开关关态=feed 应停非病灶不探；零推送行史（max=NULL）=
// 断更极端形同报。行龄断≠fetch 死（无高冲击事件窗同形陈——note 如实两可，
// 对拍归人肉/上游活性面）。只报不修——修归常票。
package hunger

import (
	"context"
	"fmt"
	"time"
)

const (
	// feedStaleHours 行龄阈（票面钦建议 24h——feed 本应日更）。
	feedStaleHours = 24
	// feedStaleDedupHours 同窗去重（silenceDedupHours 同例——同面至多一报）。
	feedStaleDedupHours = 24
	// feedStaleEscClass 升件类（判类册登记词——观测盲区类升件同 silence 族）。
	feedStaleEscClass = "观测盲"
	// feedStaleScope 判例域（feed 面单键——与 calendar.go 读败升件同 scope
	// 不同 class 不撞窗）。
	feedStaleScope = "CALENDAR-FEED"
	// feedStaleTicket 探针归属票号。
	feedStaleTicket = "T-0410"
)

// ProbeFeedStale feed 死闸一拍：开关开+末行龄超阈/零行史→病灶行+升件。
// 败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeFeedStale(ctx context.Context) error {
	on, err := s.cfg.GetBool("notify.feishu.calendar.enabled")
	if err != nil {
		return fmt.Errorf("feed死闸 开关键读败: %w", err)
	}
	if !on {
		return nil // 关态=feed 应停——非病灶域不探
	}
	var last *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(sent_at) FROM md_calendar_push`).Scan(&last); err != nil {
		return fmt.Errorf("feed死闸 行龄读败: %w", err)
	}
	var ageStr string
	if last == nil {
		ageStr = "零推送行史"
	} else if age := s.now().Sub(*last); age > feedStaleHours*time.Hour {
		ageStr = fmt.Sprintf("末行 sent_at=%s 龄=%.1fh",
			last.UTC().Format("2006-01-02 15:04Z"), age.Hours())
	} else {
		return nil // 末行鲜——feed 在拍零病灶
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		feedStaleEscClass, feedStaleScope, feedStaleDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("feed死闸 同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil // 同窗已报——24h 内同面至多一报
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'feed断更',$2,$3)`,
		hungerSeat, feedStaleTicket,
		fmt.Sprintf("阈=%dh %s", feedStaleHours, ageStr)); err != nil {
		return fmt.Errorf("feed断更行: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, feedStaleTicket,
		fmt.Sprintf("日历 feed 断更：%s（阈%dh——断更形含「无高冲击事件窗」两可如实标，对拍归人肉/上游活性面；探针只报不修）",
			ageStr, feedStaleHours),
		feedStaleEscClass, feedStaleScope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
