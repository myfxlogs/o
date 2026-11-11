// feedstale.go —— 日历 feed 死闸（T-0410）：「拉取活性行龄」巡检件。
// 病灶原型：2026-11-06 非农夜直播 feed 断更 ~2 天零告警（md_calendar_push
// 末行=11-04，NFP 整个日历面零推送——与 dispatch-spawn-silent-failure 同族
// 「缺席无感」，靠业主人工对拍发现）。探针挂饥饿拍同槽（ProbeFamilies/
// ProbeDeps 同款隔离位）：notify.feishu.calendar.enabled 开态下
// md_calendar_pull 最新行龄超阈（票面钦建议 24h——拉环 15min 拍退避封顶
// 4h,日行实拉）零新行 →ops_event `feed断更` 行+升件（观测盲/CALENDAR-FEED
// 判例键，24h 同窗去重）。开关关态=feed 应停非病灶不探；零拉取行史
// （max=NULL）=断更极端形同报。
// 工单#10693 口径修（升件#10654 实证）：锚=拉取活性非推件产出——
// md_calendar_push.sent_at 只量送达输出,零高冲击窗=设计缺席误判
// （2026-11-11 老兵节窗实证）；md_calendar_pull 逐拍实拉（含败拍）落行,
// 行龄断=拉环死真病灶,上游拒因归读败档分守。只报不修——修归常票。
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
	Label:    "日历 feed 拉环",
	Verb:     "断更",
	TsCol:    "ts",
	MaxAge:   feedStaleHours,
	Thresh:   "24h",
	ZeroWord: "零拉取行史",
	Ditto:    "拉取活性断=拉环死/超退避窗长歇（上游拒因归读败档）——对拍归 o-core 日历段日志",
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
			`SELECT max(ts) FROM md_calendar_pull`).Scan(&last); err != nil {
			return nil, err
		}
		return last, nil
	}
	_, err = selfprobe.Probe(ctx, s.pool, face, feedSpec, s.now)
	return err
}
