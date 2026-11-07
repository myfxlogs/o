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
// T-0434：判/去重/升件四拍归 selfprobe 公共模子（第三实例成族抽取）——
// 本件只剩开关位+读缝+规格。
package hunger

import (
	"context"
	"fmt"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// feedStaleHours 行龄阈（票面钦建议 24h——feed 本应日更）。
	feedStaleHours = 24 * time.Hour
	// feedStaleScope 判例域（feed 面单键——与 calendar.go 读败升件同 scope
	// 不同 class 不撞窗）。
	feedStaleScope = "CALENDAR-FEED"
	// feedStaleTicket 探针归属票号。
	feedStaleTicket = "T-0410"
)

// feedSpec feed 面巡检规格（自报表陈旧→升件公共模子形）。
var feedSpec = selfprobe.Spec{
	Site:     hungerSeat,
	Ticket:   feedStaleTicket,
	Scope:    feedStaleScope,
	Label:    "日历 feed",
	Verb:     "断更",
	TsCol:    "sent_at",
	MaxAge:   feedStaleHours,
	Thresh:   "24h",
	ZeroWord: "零推送行史",
	Ditto:    "断更形含「无高冲击事件窗」两可如实标，对拍归人肉/上游活性面",
}

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
	face := func(ctx context.Context) (*time.Time, error) {
		var last *time.Time
		if err := s.pool.QueryRow(ctx,
			`SELECT max(sent_at) FROM md_calendar_push`).Scan(&last); err != nil {
			return nil, err
		}
		return last, nil
	}
	_, err = selfprobe.Probe(ctx, s.pool, face, feedSpec, s.now)
	return err
}
