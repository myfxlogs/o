// klinestale.go —— 日K数据面双死闸（工单#13065 切片②）：「日K面活性」巡检件。
// 病灶原型：md_klines 面立后无人盯活——business 臂断连/收线语义病灶
// （形成中帧永不确定）皆「缺席无感」同族，靠下轮会话蹭到才知。
// 探针挂饥饿拍同槽（ProbeFeedStale 同款隔离位）：
//
//	面①KLINE-DAILY-RECV：max(recv_ts) 龄阈 30min——business 臂 candle1D
//	  推帧连续（活跃集 42 inst×形成中帧秒级更新），龄>30min=采集臂死；
//	  零行史（建表以来零落行）=面未产极端形同报。
//	面②KLINE-DAILY-CLOSE：收线语义龄=末条确认棒收线时刻（max(ts)+24h,
//	  bar='1D'）龄阈 30h——日K 收线帧一日一落（16:00Z 族），超时=
//	  收线语义死（confirm 位不写/收线帧被节流吞）。语义列必须用 bar
//	  时刻非 recv_ts——upsert 每拍覆写 recv_ts，收线停摆期收帧仍被
//	  形成中帧触碰=旧闸恒鲜（O-38640 实证：收线停 41.5h 而 recv_ts
//	  1.5min 前仍被写）。冷启防误报：表内无「到期该收线」bar
//	  （ts<now-25h=开点已超一根日K时长,收线帧应已落）则本面不探——
//	  部署初 25h 内零收线行是设计态非病灶。
//
// 只报不修——修归常票（采集臂/upsert 语义/订阅面）。
package hunger

import (
	"context"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// klineRecvMaxAge 收帧龄阈（WS candle 推帧连续——30min 无帧=臂死）。
	klineRecvMaxAge = 30 * time.Minute
	// klineCloseMaxAge 收线龄阈（日K 收线一拍/日——30h 无收线=语义死）。
	klineCloseMaxAge = 30 * time.Hour
	// klineCloseDueHours 「到期该收线」判据——bar 开点龄>25h=该收未收。
	klineCloseDueHours = 25
)

var (
	klineRecvSpec = selfprobe.Spec{
		Site:     hungerSeat,
		Ticket:   "#13065",
		Scope:    "KLINE-DAILY-RECV",
		Label:    "日K candle 收帧面",
		Verb:     "断更",
		TsCol:    "recv_ts",
		MaxAge:   klineRecvMaxAge,
		Thresh:   "30m",
		ZeroWord: "零落行史（建表以来零采）",
		Ditto:    "business 臂断连/订阅面拒集全灭——对拍归 o-core okxws business 段日志",
	}
	klineCloseSpec = selfprobe.Spec{
		Site:     hungerSeat,
		Ticket:   "#13065",
		Scope:    "KLINE-DAILY-CLOSE",
		Label:    "日K candle 收线面",
		Verb:     "断更",
		TsCol:    "收线时刻(ts+24h)",
		MaxAge:   klineCloseMaxAge,
		Thresh:   "30h",
		ZeroWord: "零收线行史（形成中帧永不确定）",
		Ditto:    "confirm 位不写/收线帧吞节流——对拍归 md_klines confirm 位分布",
	}
)

// ProbeKlineStale 日K死闸一拍：面①收帧龄/面②收线龄（到期才探）→病灶行+升件。
func (s *Sensor) ProbeKlineStale(ctx context.Context) error {
	// 面①：收帧活性（零行史同报——建表面未产=极端断更）。
	if _, err := selfprobe.Probe(ctx, s.pool, func(ctx context.Context) (*time.Time, error) {
		var last *time.Time
		if err := s.pool.QueryRow(ctx,
			`SELECT max(recv_ts) FROM md_klines`).Scan(&last); err != nil {
			return nil, err
		}
		return last, nil
	}, klineRecvSpec, s.now); err != nil {
		return err
	}
	// 面②门控：无到期该收线 bar=冷启窗（部署初 25h）不探防误报。
	var due int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM md_klines WHERE ts<now()-make_interval(hours=>$1)`,
		klineCloseDueHours).Scan(&due); err != nil {
		return err
	}
	if due == 0 {
		return nil
	}
	_, err := selfprobe.Probe(ctx, s.pool, func(ctx context.Context) (*time.Time, error) {
		var ts *time.Time
		if err := s.pool.QueryRow(ctx,
			`SELECT max(ts) FROM md_klines WHERE confirm AND bar='1D'`).Scan(&ts); err != nil {
			return nil, err
		}
		if ts == nil {
			return nil, nil
		}
		// 语义列=收线时刻：1D 棒收线≈开点+24h（okx 16:00Z 族日界）。
		// 用 bar 时刻而非 recv_ts——upsert 覆写 recv_ts 会把收线停摆
		// 期被形成中帧触碰的假鲜掩盖（O-38640）。
		closeAt := ts.Add(24 * time.Hour)
		return &closeAt, nil
	}, klineCloseSpec, s.now)
	return err
}
