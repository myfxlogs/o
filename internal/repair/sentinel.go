// sentinel.go —— OCI 宿主感知哨（O-38640）：md 侧 venue='mt5' 日K
// 双面活性 + 外部器官产出账。挂 sweeper 既有拍槽（同宿主同数据面
// =细胞层零新器官——docs/resident_units.txt 无需登记）。
//
//	面①KLINE-MT5-RECV：max(recv_ts) 龄阈 8h——collecto klineEvery=4h
//	  扫拍覆写全行 recv_ts，龄>8h=两拍断扫（会话死/扫败静默）。
//	面②KLINE-MT5-CLOSE：收线语义龄=末条确认棒收线时刻（max(ts)+24h,
//	  bar='1D' venue='mt5'）龄阈 96h——周末闭市+经纪商日界漂移
//	  （EET 系 ±3h）+扫拍滞后全吸进阈内；周棒污染/确认停摆超 4d
//	  必报。零行史同报（面未产=极端断更）。
//
// 只报不修——修归常票/采集臂。
package repair

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/selfprobe"
)

const (
	// mt5KlineRecvMaxAge 收帧龄阈（4h 扫拍×2=两拍断扫判死）。
	mt5KlineRecvMaxAge = 8 * time.Hour
	// mt5KlineCloseMaxAge 收线龄阈（周末闭市最长 ~52h+日界漂移——
	// 96h=4d 兜住一切良态窗，超即真停摆）。
	mt5KlineCloseMaxAge = 96 * time.Hour
	// sentinelBeat 哨兵拍距（30min——8h 阈探针细查无意义，半时足）。
	sentinelBeat = 30 * time.Minute
)

var (
	mt5RecvSpec = selfprobe.Spec{
		Site:     "sweeper",
		Ticket:   "O-38640",
		Scope:    "KLINE-MT5-RECV",
		Label:    "mt5 日K 收帧面",
		Verb:     "断更",
		TsCol:    "recv_ts",
		MaxAge:   mt5KlineRecvMaxAge,
		Thresh:   "8h",
		ZeroWord: "零落行史（venue='mt5' 建表以来零采）",
		Ditto:    "collecto kline 扫拍断/会话死——对拍归 journalctl o-collecto [kline] 段",
	}
	mt5CloseSpec = selfprobe.Spec{
		Site:     "sweeper",
		Ticket:   "O-38640",
		Scope:    "KLINE-MT5-CLOSE",
		Label:    "mt5 日K 收线面",
		Verb:     "断更",
		TsCol:    "收线时刻(ts+24h)",
		MaxAge:   mt5KlineCloseMaxAge,
		Thresh:   "96h",
		ZeroWord: "零收线行史（confirm 位无落行）",
		Ditto:    "粒度污染/确认停摆——对拍归 md_klines venue='mt5' bar 分布与 dow 签名",
	}
)

// Sentinel OCI 宿主感知哨（md 读面+SG 升件面+md app_config 投影面）。
type Sentinel struct {
	md      *pgxpool.Pool
	esc     selfprobe.EscPool
	cfg     *config.Client
	now     func() time.Time
	sysLast func(context.Context, string) (*time.Time, error)
}

// New 装配哨兵（esc nil=守位不探同 selfprobe 律）。
func New(md *pgxpool.Pool, esc selfprobe.EscPool, cfg *config.Client) *Sentinel {
	return &Sentinel{md: md, esc: esc, cfg: cfg, now: time.Now, sysLast: sysLastTrigger}
}

// Run 哨兵环（首拍即时+30min 拍——sched-exempt-ok：感知面非业务轮询）。
func (s *Sentinel) Run(ctx context.Context) {
	s.beat(ctx)
	t := time.NewTicker(sentinelBeat) // sched-exempt-ok：感知面陈龄巡检期界非业务轮询（同 repair.Watch 60s 律）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.beat(ctx)
		}
	}
}

// beat 一拍：md_klines venue='mt5' 双面 + 器官死活账（manifest 空=不探）。
func (s *Sentinel) beat(ctx context.Context) {
	if _, err := selfprobe.Probe(ctx, s.esc, func(ctx context.Context) (*time.Time, error) {
		var last *time.Time
		if err := s.md.QueryRow(ctx,
			`SELECT max(recv_ts) FROM md_klines WHERE venue='mt5'`).Scan(&last); err != nil {
			return nil, err
		}
		return last, nil
	}, mt5RecvSpec, s.now); err != nil {
		log.Printf("WARN [sentinel] mt5 收帧面读败: %v", err)
	}
	if _, err := selfprobe.Probe(ctx, s.esc, func(ctx context.Context) (*time.Time, error) {
		var ts *time.Time
		if err := s.md.QueryRow(ctx,
			`SELECT max(ts) FROM md_klines WHERE venue='mt5' AND confirm AND bar='1D'`).Scan(&ts); err != nil {
			return nil, err
		}
		if ts == nil {
			return nil, nil
		}
		// 收线语义龄=bar 开点+24h（同 hunger 面②律——写触碰不算产出）。
		closeAt := ts.Add(24 * time.Hour)
		return &closeAt, nil
	}, mt5CloseSpec, s.now); err != nil {
		log.Printf("WARN [sentinel] mt5 收线面读败: %v", err)
	}
	s.probeOrgans(ctx)
}
