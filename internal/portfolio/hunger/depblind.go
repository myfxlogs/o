// depblind.go —— 依赖对活性枚举探针（T-0394 自我迭代发现层封顶件：
// 「还有类似的问题，又由谁来发现呢？」——发现缺陷形状第④层=不变量
// 机械枚举，carry_diff 盲/现货腿盲/未见的同类全是本族成员）。
//
// 语义：枚举系统全部「产者↔消费者」依赖对→逐对活性对账——产者活着
// （新鲜写入）而消费者视角零行/缺席=失明即自报，不管该盲叫什么形状。
// 探针挂饥饿拍同槽（ProbeFamilies/ProbeLegCoverage 同位省协程），只报
// 不修（修归常票）；败=上抛归 hungerTick 自报位非静默。
//
// 枚举面（票面钦四类起，①②优先③④ deferred）：
//
//	①信号对：md_signal kind 写者↔research/策略消费臂——kind 面不硬
//	  编码，复用族沉默探针同款两链派生（假想卡 data_needs 注册表链+
//	  template 命名规约链）枚举 kind→template 对；消费册外 kind（两
//	  链皆不属）=「产而活消费缺席」对（orphan——消费者缺席形失明）。
//	②腿-行情对：策略/提案各腿所需行情键↔md_ticks——腿键集=
//	  legset.ActiveSetUnion 同源表达式（订阅集/镜像谓词同串，禁自造
//	  第二套归一链）∩订阅宇宙（okx.symbols+IsUSDTInst 闸——虚构键
//	  宇宙外不求供价不炸）。
//	③表读写对/④配置对：票面钦可标 deferred——本件枚举器面预留
//	  （deferred 落地，界碑见票面结果回填）。
//
// 枚举器自身 fail-closed：全谱扫零对=报错升件不静默（扫不到对=枚举
// 面病灶非「零失明」）。
//
// 界碑（票面钦写明）：0384 族沉默探针=template 聚合特例先登（驱动
// kind 对拍族评估），0392 腿覆盖探针=单键覆盖特例；本件=产者↔消费者
// 对粒度一般化面——重复报由签名去重吸收（同窗 24h 对签名+产者/消费
// 者自然键三键去重：特例探针已报同对象即吸收不重复升）。
package hunger

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	// depblindSignalDays 信号对窗（天）——族沉默探针同款 7d 口径（md_signal
	// 状态表同键最新赢，窗内行数=窗内鲜键数非流水量）。
	depblindSignalDays = 7
	// depblindLegFreshMin 腿行情新鲜窗（分钟）——腿覆盖同款 10min 口径：
	// 判单键覆盖非面死，低流动腿安静间隙免误报。
	depblindLegFreshMin = 10
	// depblindDedupHours 同窗去重——同对签名 24h 至多一报（esc_class+
	// esc_scope 等值键，结构性限度同族沉默探针先例）。
	depblindDedupHours = 24
	// depblindEscClass 升件类（判类册登记词——观测盲区类升件，同族先例）。
	depblindEscClass = "观测盲"
	// depblindTicket 探针归属票号（例行行携环票号同 T-0384 例）。
	depblindTicket = "T-0394"
)

// depPair 依赖对（枚举器产物——产者↔消费者活性断言对象）。
type depPair struct {
	sig            string // 对签名（dep_blind key/esc_scope——逐对唯一）
	family         string // 枚举族（signal|leg——③④ deferred 未入）
	producer       string // 产者键（kind/inst）
	consumer       string // 消费者键（template/inst）
	consumerAbsent bool   // 消费册缺席形（orphan kind——产而活零读）
}

// depBlindWhitelist 注册式豁免面——合法「产而未读」基线落位点：首跑全谱
// 人工过一次后，合法未读对在此登记。key=对签名（'…*' 尾=前缀通配——如
// 'signal:macro_*→∅' 一族）；value=理由键（每豁免附理由——注册式不静默：
// 豁免对同窗落「依赖对豁免」行留账，非暗吞）。
var depBlindWhitelist = map[string]string{}

// depBlindExempt 豁免判定（精确签名优先，'*' 尾前缀通配次——理由键随行）。
func depBlindExempt(sig string) (string, bool) {
	if r, ok := depBlindWhitelist[sig]; ok {
		return r, true
	}
	for k, r := range depBlindWhitelist {
		if strings.HasSuffix(k, "*") && strings.HasPrefix(sig, k[:len(k)-1]) {
			return r, true
		}
	}
	return "", false
}

// ProbeDeps 依赖对活性枚举探针一拍：枚举全谱对→逐对活性对账→失明对落
// 「依赖对失明」行+观测盲升件。枚举空集=fail-closed 报错（上抛自报）。
func (s *Sensor) ProbeDeps(ctx context.Context) error {
	pairs, err := s.enumerateDeps(ctx)
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		return errors.New("依赖对枚举器空集——fail-closed（扫不到对=枚举面病灶非零失明）")
	}
	if err := s.depCensus(ctx, pairs); err != nil {
		return fmt.Errorf("依赖对清单行: %w", err)
	}
	var errs []error
	for _, p := range pairs {
		if err := s.probeDep(ctx, p); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.sig, err))
		}
	}
	return errors.Join(errs...)
}

// probeDep 单对活性对账：豁免面先判（注册式——豁免对同窗落「依赖对豁免」
// 行非暗吞）→产者/消费者窗内行数→失明判定→签名去重→失明行+升件。
func (s *Sensor) probeDep(ctx context.Context, p depPair) error {
	if reason, ok := depBlindExempt(p.sig); ok {
		return s.depExemptRow(ctx, p, reason)
	}
	prodN, consN, err := s.depCounts(ctx, p)
	if err != nil {
		return err
	}
	blind := prodN > 0 && (p.consumerAbsent || consN == 0)
	switch p.family {
	case "leg", "scan":
		blind = prodN == 0 // 产者死（行情行/粗扫行零新鲜）而需求在册=失明
	case "subreq":
		blind = prodN > 0 && consN == 0 // 请求超龄在册而同键 tick 零新鲜=晋级滞留
	}
	if !blind {
		return nil
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope=ANY($1)
		AND ts>now()-make_interval(hours=>$2)`,
		p.dedupScopes(), depblindDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil // 本对/特例探针同窗已报同对象——签名去重吸收
	}
	cons := p.consumer
	consNStr := fmt.Sprintf("%d", consN)
	if p.consumerAbsent {
		cons = "∅（消费册缺席——两链派生零命中）"
		consNStr = "缺席"
	}
	win := fmt.Sprintf("%dd", depblindSignalDays)
	if p.family == "leg" || p.family == "scan" || p.family == "subreq" {
		win = fmt.Sprintf("%dmin", depblindLegFreshMin)
	}
	note := fmt.Sprintf("kind=dep_blind key=%s family=%s 产者=%s 产者窗内行=%d 消费者=%s 消费者窗内行=%s 窗=%s",
		p.sig, p.family, p.producer, prodN, cons, consNStr, win)
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'依赖对失明',$2,$3)`,
		hungerSeat, depblindTicket, note); err != nil {
		return fmt.Errorf("依赖对失明行: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, depblindTicket,
		fmt.Sprintf("依赖对失明 %s：产者 %s 窗内行=%d 而消费者 %s 行=%s——产者活消费者零视野依赖对失明（探针只报不修，修归常票/饥饿响应）",
			p.sig, p.producer, prodN, cons, consNStr),
		depblindEscClass, p.sig); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}

// dedupScopes 签名去重吸收面：本对签名+产者键+消费者自然键三键——
// 0384（esc_scope=template）/0392（esc_scope=inst）特例探针同窗已报同
// 对象即吸收不重复升；缺席消费者对零消费者键不入集。
func (p depPair) dedupScopes() []string {
	out := []string{p.sig, p.producer}
	if p.consumer != "" && !p.consumerAbsent {
		out = append(out, p.consumer)
	}
	return out
}

// depCounts 对窗内行数（信号对=md_signal kind×fin_opp_eval template 7d
// 同款口径；腿对=md_ticks inst 10min 新鲜行×消费者在册恒 1——腿对消费者
// =activeSet 枚举本体，在册即需求）。缺席消费者对 consN 恒 -1。
func (s *Sensor) depCounts(ctx context.Context, p depPair) (prodN, consN int, err error) {
	switch p.family {
	case "signal":
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
			WHERE kind=$1 AND ts>now()-make_interval(days=>$2)`,
			p.producer, depblindSignalDays).Scan(&prodN); err != nil {
			return 0, 0, fmt.Errorf("产者信号行数读败: %w", err)
		}
		if p.consumerAbsent {
			return prodN, -1, nil
		}
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
			WHERE template=$1 AND ts>now()-make_interval(days=>$2)`,
			p.consumer, depblindSignalDays).Scan(&consN); err != nil {
			return 0, 0, fmt.Errorf("消费者评估行数读败: %w", err)
		}
	case "leg":
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM md_ticks
			WHERE inst_id=$1 AND ts>now()-make_interval(mins=>$2)`,
			p.producer, depblindLegFreshMin).Scan(&prodN); err != nil {
			return 0, 0, fmt.Errorf("产者行情行数读败: %w", err)
		}
		consN = 1 // 消费者在册恒真（枚举本体=需求方）
	case "scan": // T-0401③粗扫活性——产者=md_tick_scan 全谱新鲜行数
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan
			WHERE ts>now()-make_interval(mins=>$1)`,
			depblindLegFreshMin).Scan(&prodN); err != nil {
			return 0, 0, fmt.Errorf("产者粗扫行数读败: %w", err)
		}
		consN = 1
	case "subreq": // T-0401③晋级滞留——产者=超龄 open 请求在册，消费者=同键新鲜 tick
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_sub_request
			WHERE inst_id=$1 AND state='open' AND ts<now()-make_interval(mins=>$2)`,
			p.producer, depblindLegFreshMin).Scan(&prodN); err != nil {
			return 0, 0, fmt.Errorf("产者晋级请求读败: %w", err)
		}
		if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM md_ticks
			WHERE inst_id=$1 AND ts>now()-make_interval(mins=>$2)`,
			p.consumer, depblindLegFreshMin).Scan(&consN); err != nil {
			return 0, 0, fmt.Errorf("消费者晋级 tick 读败: %w", err)
		}
	default:
		return 0, 0, fmt.Errorf("未知依赖对族: %s", p.family)
	}
	return prodN, consN, nil
}

// depExemptRow 豁免对落账（注册式不静默——「依赖对豁免」行同窗 24h 至多
// 一行携理由键；豁免疫判本身不入升件面）。
func (s *Sensor) depExemptRow(ctx context.Context, p depPair, reason string) error {
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='依赖对豁免' AND note LIKE $1
		AND ts>now()-make_interval(hours=>$2)`,
		"key="+p.sig+" %", depblindDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("豁免去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'依赖对豁免',$2,$3)`,
		hungerSeat, depblindTicket,
		fmt.Sprintf("key=%s 理由键=%s——合法产而未读注册豁免（人工基线落位）", p.sig, reason))
	return err
}
