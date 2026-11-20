// deploywatch.go —— deploy_watch 心跳看门狗探针（工单#25803）：「盯人的
// 探头没人盯」族新实例——deploy_watch 是部署唯一自动径
// （o-deploy-watch.timer 15min 拍），自身无心跳无看门狗：服务死/
// timer 挂/锁卡死=部署静默停摆无告警（本纪元实证：RSI 停后
// land_queue timer 死多日无告警，支积压靠人工发现）。
// deploy_watch 每拍写 ops_event seat=DEPLOY event=心跳（判词随行）；
// 本探针读末行龄>45min（3×拍距）→病灶行+观测盲升件
// esc_scope=deploy_watch_silent（票面钦定键）。
// 看门狗与看守对象不同件（hunger 探针族读 ops_event vs systemd timer
// 拉起 deploy_watch.sh）——独立失效面；timer 死=心跳停=升件，
// 服务崩/锁卡死同覆无需另检死因。只报不修——拉起归 systemd 域，
// 禁自愈重启（派工单禁线）。判/去重/升件归 selfprobe 公共模子
// （T-0434 族抽取实例——feed/ctx/hostmetrics 同款第四面）。
package hunger

import (
	"context"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// deployWatchSilentMaxAge 心跳末行龄阈（票面钦定 3×拍距=45min——
	// deploy_watch.timer OnCalendar 15min 拍）。
	deployWatchSilentMaxAge = 45 * time.Minute
	// deployWatchSilentScope 升件域键（票面钦定 deploy_watch_silent）。
	deployWatchSilentScope = "deploy_watch_silent"
	// deployWatchSilentTicket 探针归属票号。
	deployWatchSilentTicket = "#25803"
)

// deployWatchSpec 看门狗巡检规格（自报表陈旧→升件公共模子形）。
var deployWatchSpec = selfprobe.Spec{
	Site:     hungerSeat,
	Ticket:   deployWatchSilentTicket,
	Scope:    deployWatchSilentScope,
	Label:    "deploy_watch 心跳",
	Verb:     "静默",
	TsCol:    "ts",
	MaxAge:   deployWatchSilentMaxAge,
	Thresh:   "45min",
	Rhythm:   "15min",
	ZeroWord: "零心跳行史",
	Ditto:    "末行龄断=timer死/服务崩/锁卡死三因同覆（看门狗与看守不同件——探针只报不修，拉起归systemd域）——对拍归 systemctl --user status o-deploy-watch",
}

// ProbeDeployWatch deploy_watch 心跳死闸一拍：末行龄超阈/零行史→病灶行+升件。
// 败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeDeployWatch(ctx context.Context) error {
	face := func(ctx context.Context) (*time.Time, error) {
		var last *time.Time
		if err := s.pool.QueryRow(ctx,
			`SELECT max(ts) FROM ops_event WHERE seat='DEPLOY' AND event='心跳'`).Scan(&last); err != nil {
			return nil, err
		}
		return last, nil
	}
	_, err := selfprobe.Probe(ctx, s.pool, face, deployWatchSpec, s.now)
	return err
}
