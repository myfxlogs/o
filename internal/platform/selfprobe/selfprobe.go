// Package selfprobe —— 自报表陈旧→升件公共模子（T-0434 抽取）。
// 「盯人的探头没人盯」族三实例成族钦定归一：feedstale（T-0410 日历 feed）、
// ctx 断供探针（T-0429 md_ctx_fact 源）、hostmetrics 停拍探针（T-0434
// fin_host_metrics）同踩一模——row-age 巡检 + 24h 同窗去重 + ops_event
// `feed断更` 病灶行 + 观测盲升件，四拍在模子里，各面只留规格与读缝。
// 病灶族谱：缺席无感（dispatch-spawn-silent-failure → feed 断更 → ctx
// 断供 → 自报停拍）——「静默死第 N 条实证」的机械对治即本模子。
// 探针只报不修——修归常票；消费归裁定席 ops_event 既有径，禁孤儿 sink。
package selfprobe

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	// dedupHours 同窗去重（族 norm——同面 24h 至多一报）。
	dedupHours = 24
	// EscClass 升件类（观测盲区族判类册登记词——feedstale/ctx 同款）。
	EscClass = "观测盲"
)

// Spec 单面巡检规格（scope 单键——升件按面独立不撞窗）。
type Spec struct {
	Site     string        // ops_event seat 词（上报组件身份）
	Ticket   string        // 归属票号
	Scope    string        // esc_scope 判例域单键（CALENDAR-FEED / CTX-FRED / HOSTMETRICS-SG）
	Label    string        // 病灶面名（note 主语——「日历 feed」「ctx 源 fred」「主机自报 sg」）
	Verb     string        // 病灶动词（「断更」「断供」「停拍」——升件 note 谓语）
	TsCol    string        // 末行时间列名（ageStr 列锚——「sent_at」「received_ts」「ts」）
	MaxAge   time.Duration // 阈（判据本值）
	Thresh   string        // 阈人读形（「24h」「10m」——note 阈判据锚）
	Rhythm   string        // 节律词（阈判据外推锚；空=不显）
	ZeroWord string        // 零行史词（各面语义词——「零推送行史」「零行史」）
	Ditto    string        // 两可如实标语（陈形含良态窗的同款如实标——对拍归人肉/上游活性面）
}

// Face 单面末行龄读缝（nil=零行史；各面自持查询与池——模子只管判/去重/升件）。
type Face func(ctx context.Context) (*time.Time, error)

// EscPool 升件写面最小缝（*pgxpool.Pool 同形——桩测可覆写）。
type EscPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

// Probe 一拍：末行龄超阈/零行史→同窗去重→ops_event `feed断更` 行+观测盲升件。
// 返 reported=本拍落账与否；败上抛归宿主 tick 自报位非静默。
// escPool nil=守位不探（装配缺件如实缺升件——采集成果不得被观测臂拖崩）。
func Probe(ctx context.Context, escPool EscPool, face Face, spec Spec, now func() time.Time) (bool, error) {
	if escPool == nil {
		return false, nil // 守位
	}
	last, err := face(ctx)
	if err != nil {
		return false, fmt.Errorf("自报探针 %s 行龄读败: %w", spec.Scope, err)
	}
	var ageStr string
	if last == nil {
		ageStr = spec.ZeroWord
	} else if age := now().Sub(*last); age > spec.MaxAge {
		ageStr = fmt.Sprintf("末行 %s=%s 龄=%.1fh",
			spec.TsCol, last.UTC().Format("2006-01-02 15:04Z"), age.Hours())
	} else {
		return false, nil // 末行鲜——面在拍零病灶
	}
	// 同窗去重（族 norm——同面 24h 至多一报）。
	var dup int
	if err := escPool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		EscClass, spec.Scope, dedupHours).Scan(&dup); err != nil {
		return false, fmt.Errorf("自报探针 %s 同窗去重读败: %w", spec.Scope, err)
	}
	if dup > 0 {
		return false, nil // 同窗已报
	}
	note := fmt.Sprintf("阈=%s %s", spec.Thresh, ageStr)
	if _, err := escPool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'feed断更',$2,$3)`,
		spec.Site, spec.Ticket, fmt.Sprintf("%s %s", spec.Scope, note)); err != nil {
		return false, fmt.Errorf("自报探针 %s 病灶行: %w", spec.Scope, err)
	}
	rhythm := ""
	if spec.Rhythm != "" {
		rhythm = fmt.Sprintf(" 节律=%s", spec.Rhythm)
	}
	if _, err := escPool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		spec.Site, spec.Ticket,
		fmt.Sprintf("%s%s：%s（阈=%s%s——%s；探针只报不修）",
			spec.Label, spec.Verb, note, spec.Thresh, rhythm, spec.Ditto),
		EscClass, spec.Scope); err != nil {
		return false, fmt.Errorf("自报探针 %s 升件行: %w", spec.Scope, err)
	}
	return true, nil
}
