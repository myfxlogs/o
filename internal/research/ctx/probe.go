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
package ctx

import (
	"context"
	"fmt"
	"time"
)

const (
	// probeSeat 升件 seat 词（research 域惯例——research/grammar、research/dcd 同形）。
	probeSeat = "research/ctx"
	// probeTicket 探针归属票号。
	probeTicket = "T-0429"
	// probeEscClass 升件类（判类册登记词——观测盲区类探针同族先例）。
	probeEscClass = "观测盲"
	// probeDedupHours 同窗去重（feedstale 同例——同面 24h 至多一报）。
	probeDedupHours = 24
)

// staleRule 源节律阈规则（scope=判例域单键——升件按源独立不撞窗）。
type staleRule struct {
	source string
	scope  string
	maxAge time.Duration
	rhythm string // 节律词（升件 note 如实标——阈判据锚）
}

// staleRules 进水口源节律阈表（新源入列同拍——源与探针同生同账）。
var staleRules = []staleRule{
	{"ffcal", "CTX-FFCAL", 10 * 24 * time.Hour, "周历滚动（新事件 slug 周级）"},
	{"fred", "CTX-FRED", 96 * time.Hour, "营业日级（周末缺口 72h 富余）"},
	{"frankfurter", "CTX-FX", 96 * time.Hour, "ECB 营业日"},
	{"cftc", "CTX-COT", 9 * 24 * time.Hour, "周五周报（缺席一周+2d 富余）"},
	{"okx_public", "CTX-OI", 48 * time.Hour, "日戳行（缺一日即报）"},
}

// ProbeStale 断供探针一拍：每源 md_ctx_fact 末行龄超阈/零行史→病灶行+升件。
// 败=上抛归 TickOnce 自报位非静默；escPool 未挂=守位不探不炸（装配缺件
// 如实缺升件——采集成果不得被观测臂拖崩）。
func (r *Runner) ProbeStale(c context.Context) error {
	if r.escPool == nil {
		return nil // 守位
	}
	for _, rule := range staleRules {
		var last *time.Time
		if err := r.pool.QueryRow(c,
			`SELECT max(received_ts) FROM md_ctx_fact WHERE source=$1`, rule.source).Scan(&last); err != nil {
			return fmt.Errorf("断供探针 %s 行龄读败: %w", rule.source, err)
		}
		var ageStr string
		if last == nil {
			ageStr = "零行史"
		} else if age := r.now().Sub(*last); age > rule.maxAge {
			ageStr = fmt.Sprintf("末行 received_ts=%s 龄=%.1fh",
				last.UTC().Format("2006-01-02 15:04Z"), age.Hours())
		} else {
			continue // 末行鲜——源在拍零病灶
		}
		// 同窗去重（feedstale 同例——同面 24h 至多一报）。
		var dup int
		if err := r.escPool.QueryRow(c, `SELECT count(*) FROM ops_event
			WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
			AND ts>now()-make_interval(hours=>$3)`,
			probeEscClass, rule.scope, probeDedupHours).Scan(&dup); err != nil {
			return fmt.Errorf("断供探针 %s 同窗去重读败: %w", rule.source, err)
		}
		if dup > 0 {
			continue
		}
		note := fmt.Sprintf("阈=%dh %s", int(rule.maxAge.Hours()), ageStr)
		if _, err := r.escPool.Exec(c,
			`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'feed断更',$2,$3)`,
			probeSeat, probeTicket,
			fmt.Sprintf("%s %s", rule.scope, note)); err != nil {
			return fmt.Errorf("feed断更行 %s: %w", rule.source, err)
		}
		if _, err := r.escPool.Exec(c,
			`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES($1,'升件',$2,$3,$4,$5)`,
			probeSeat, probeTicket,
			fmt.Sprintf("ctx 源 %s 断供：%s（节律=%s——断供形含「节律窗内无新件」两可如实标，对拍归人肉/上游活性面；探针只报不修）",
				rule.source, note, rule.rhythm),
			probeEscClass, rule.scope); err != nil {
			return fmt.Errorf("升件行 %s: %w", rule.source, err)
		}
	}
	return nil
}
