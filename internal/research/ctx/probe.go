// probe.go —— ctx 进水口断供探针（T-0429）：「md_ctx_fact 源末行龄」巡检件。
// 形=T-0410 feedstale.go 钦定样板（row-age 巡检+ops_event `feed断更` 病灶行+
// 观测盲升件 esc_scope=CTX-<源>+24h 同窗去重——禁自创新形）。病灶原型：
// FRED 止 id976 零升件静默死+ffcal 零行史死（判词六维饥荒实证 2026-11-07——
// 源死无人知=「缺席无感」数据源层版本）。
// 阈按源节律定（feedstale 24h 阈=日更 feed 同理外推）：营业日级源 96h
// （周末缺口 72h 富余——48h 每周末必假报）；周五周报 9d（缺席一周+2d）；
// 日戳源 48h（缺一日即报）；周历滚动 10d（新事件 slug 周级）。零行史=
// 断供极端形同报（源死在出生时可见——feedstale 形④同例）。断供形含
// 「节律窗内无新件」两可如实标（feedstale note 同款——对拍归人肉/上游
// 活性面）。消费归裁定席 ops_event 既有径，禁孤儿 sink；探针只报不修。
// T-0434：判/去重/升件四拍归 selfprobe 公共模子（第三实例成族抽取）——
// 本件只剩源节律阈表+读缝。
package ctx

import (
	"context"
	"fmt"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// probeSeat 升件 seat 词（research 域惯例——research/grammar、research/dcd 同形）。
	probeSeat = "research/ctx"
	// probeTicket 探针归属票号。
	probeTicket = "T-0429"
)

// staleRule 源节律阈规则（scope=判例域单键——升件按源独立不撞窗）。
type staleRule struct {
	source string
	scope  string
	maxAge time.Duration
	thr    string // 阈人读形（note 锚）
	rhythm string // 节律词（升件 note 如实标——阈判据锚）
}

// staleRules 进水口源节律阈表（新源入列同拍——源与探针同生同账）。
var staleRules = []staleRule{
	{"ffcal", "CTX-FFCAL", 10 * 24 * time.Hour, "240h", "周历滚动（新事件 slug 周级）"},
	{"fred", "CTX-FRED", 96 * time.Hour, "96h", "营业日级（周末缺口 72h 富余）"},
	{"frankfurter", "CTX-FX", 96 * time.Hour, "96h", "ECB 营业日"},
	{"cftc", "CTX-COT", 9 * 24 * time.Hour, "216h", "周五周报（缺席一周+2d 富余）"},
	{"okx_public", "CTX-OI", 48 * time.Hour, "48h", "日戳行（缺一日即报）"},
}

// ruleSpec 源规格→公共模子形。
func ruleSpec(r staleRule) selfprobe.Spec {
	return selfprobe.Spec{
		Site:     probeSeat,
		Ticket:   probeTicket,
		Scope:    r.scope,
		Label:    fmt.Sprintf("ctx 源 %s", r.source),
		Verb:     "断供",
		TsCol:    "received_ts",
		MaxAge:   r.maxAge,
		Thresh:   r.thr,
		Rhythm:   r.rhythm,
		ZeroWord: "零行史",
		Ditto:    "断供形含「节律窗内无新件」两可如实标，对拍归人肉/上游活性面",
	}
}

// ProbeStale 断供探针一拍：每源 md_ctx_fact 末行龄超阈/零行史→病灶行+升件。
// 败=上抛归 TickOnce 自报位非静默；escPool 未挂=守位不探不炸（装配缺件
// 如实缺升件——采集成果不得被观测臂拖崩）。
func (r *Runner) ProbeStale(c context.Context) error {
	if r.escPool == nil {
		return nil // 守位
	}
	for _, rule := range staleRules {
		source := rule.source
		face := func(ctx context.Context) (*time.Time, error) {
			var last *time.Time
			if err := r.pool.QueryRow(ctx,
				`SELECT max(received_ts) FROM md_ctx_fact WHERE source=$1`, source).Scan(&last); err != nil {
				return nil, err
			}
			return last, nil
		}
		if _, err := selfprobe.Probe(c, r.escPool, face, ruleSpec(rule), r.now); err != nil {
			return err
		}
	}
	return nil
}
