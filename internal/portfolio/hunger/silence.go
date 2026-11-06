// silence.go —— 族级沉默探针（T-0384）：「信号活而评估零」观测盲区自报件。
// 病灶原型：T-0164 carry_diff 库内鲜行在流而 fin_opp_eval xvenue-carry 族
// N 天零行——上游活性监控（ops_face 分件节拍 warn/dead）测「上游活不活」、
// 饥饿 Detect 测「提案/paper 零不零」，中间断档「模板族源信号鲜而评估零」
// 两不测。探针挂饥饿拍同槽（Detect 同拍省协程）：对每模板族
// （fin_opp_eval.template 全集）两数对拍——其源信号 md_signal 对应 kind
// 近窗鲜行数 × fin_opp_eval 该 template 近窗行数；信号行>0 且评估行=0
// → 沉默病灶（ops_event `族沉默` 行 + 观测盲升件，24h 同窗去重）。
// 只报不修——探针升件不自动修（修归常票/饥饿响应）；不重造监控——
// 复用本拍窗/去重面，信号死非本探针域（源死归 ops_face 活性面）。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// silenceWindowDays 对拍近窗（票面钦 7d——md_signal 状态表同键最新赢，
// 窗内行数=该 kind 窗内有鲜键数非流水量）。
const silenceWindowDays = 7

// silenceDedupHours 同窗去重（票面钦 24h——esc_class+esc_scope 等值键，
// 同族同窗至多一报；结构性限度同 hungerEscalate 先例）。
const silenceDedupHours = 24

// silenceEscClass 升件类（判类册登记词——观测盲区类升件）。
const silenceEscClass = "观测盲"

// silenceTicket 探针归属票号（T-0384 立件——例行行携环票号同 T-0376 例）。
const silenceTicket = "T-0384"

// ProbeFamilies 族级沉默探针一拍：模板全集两数对拍→病灶行+升件。
// kind↔template 映射零硬编码——票面钦「现成语义链优先」两链派生：
//
//	①假想卡族注册表链——fin_opp_hypothesis.requirements.data_needs 声明
//	  的族信号需求与 md_signal kind 实集求交（surface↔template 名归一
//	  〔-↔_ 折叠〕，实例级面 stake_carry-TRX→stake-carry 前缀形同认）；
//	  首命中 kind=族驱动信号（卡 data_needs 序=起草钦定主源序）；
//	②template 命名规约链——首词元作 kind 前缀（funding-hedge→funding_
//	  view；xvenue-carry→xvenue_spread 仅入从件，驱动仍①链 carry_diff）。
//
// 两链皆空=族源非 md_signal（ticks/dcd/能力面直读族）——出探针域不报。
func (s *Sensor) ProbeFamilies(ctx context.Context) error {
	kindSet, err := s.signalKindSet(ctx)
	if err != nil {
		return fmt.Errorf("族沉默探针 kind 谱读败: %w", err)
	}
	needs, err := s.hypothesisNeeds(ctx)
	if err != nil {
		return fmt.Errorf("族沉默探针 假想卡需求读败: %w", err)
	}
	templates, err := s.evalTemplates(ctx)
	if err != nil {
		return fmt.Errorf("族沉默探针 模板全集读败: %w", err)
	}
	var errs []error
	for _, t := range templates {
		kinds := deriveKinds(t, kindSet, needs)
		if len(kinds) == 0 {
			continue // 族源非 md_signal——出探针域
		}
		if err := s.probeFamily(ctx, t, kinds); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", t, err))
		}
	}
	return errors.Join(errs...)
}

// signalKindSet md_signal kind 实集（kind 谱=蒸馏侧注册行——探针不自造词表）。
func (s *Sensor) signalKindSet(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT kind FROM md_signal`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := map[string]bool{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		set[k] = true
	}
	return set, rows.Err()
}

// hypothesisNeeds 假想卡信号需求面（族注册表链——surface→kind 词集，
// requirements.data_needs 声明词与 kind 实集求交后的净集；卡无
// data_needs 键=该卡零信号需求贡献）。
func (s *Sensor) hypothesisNeeds(ctx context.Context) (map[string][]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT surface, requirements->'data_needs'
		FROM fin_opp_hypothesis WHERE requirements ? 'data_needs'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var surface string
		var raw []byte
		if err := rows.Scan(&surface, &raw); err != nil {
			return nil, err
		}
		var needs []string
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		if err := json.Unmarshal(raw, &needs); err != nil {
			continue // data_needs 非词组——该卡让位如实
		}
		out[normName(surface)] = needs
	}
	return out, rows.Err()
}

// evalTemplates 模板全集（票面钦 fin_opp_eval.template 全集——评估留痕族
// 才是观测对象；fin_opp_template 候选册不入（未产评估行的族零基线））。
func (s *Sensor) evalTemplates(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT template FROM fin_opp_eval ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// normName 名归一（-↔_ 折叠+小写——假想官 family/surface 下划线规约
// 与 eval template 横线规约同族同认：xvenue_carry↔xvenue-carry）。
func normName(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", "_"))
}

// surfaceMatch 实例级面↔模板匹配：归一等值或 surface 以 template 为族前缀
// （stake_carry-TRX→stake-carry——实例面归族；'*_template' 假想卡面同式）。
func surfaceMatch(surface, template string) bool {
	sn, tn := normName(surface), normName(template)
	return sn == tn || strings.HasPrefix(sn, tn+"_")
}

// deriveKinds 模板→md_signal kind 序（首元=驱动 kind——①链卡册声明序
// 优先，②命名规约补齐从件；保序去重）。
func deriveKinds(template string, kindSet map[string]bool, needs map[string][]string) []string {
	var out []string
	seen := map[string]bool{}
	push := func(k string) {
		if kindSet[k] && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	// ①假想卡族注册表链：data_needs 词与 kind 实集求交（'|' 替代表
	// 逐元求交——cmc_realized|mt5_swap|funding_view→后两元为 kind）。
	// 面序定排——多卡命中同族时驱动 kind 须稳定（map 序不赖）。
	var surfs []string
	for surface := range needs {
		if surfaceMatch(surface, template) {
			surfs = append(surfs, surface)
		}
	}
	sort.Strings(surfs)
	for _, surface := range surfs {
		for _, entry := range needs[surface] {
			for _, tok := range strings.Split(entry, "|") {
				push(strings.TrimSpace(tok))
			}
		}
	}
	// ②template 命名规约链：首词元作 kind 前缀（funding-hedge→funding_view；
	// 无卡册声明族的唯一链）。kind 序定排同上（多命中时首元稳定）。
	if i := strings.IndexAny(template, "-_"); i > 0 {
		head := template[:i]
		var ks []string
		for k := range kindSet {
			if k == head || strings.HasPrefix(k, head+"_") {
				ks = append(ks, k)
			}
		}
		sort.Strings(ks)
		for _, k := range ks {
			push(k)
		}
	}
	return out
}

// probeFamily 单族两数对拍：驱动 kind 近窗鲜行>0 且同族评估行=0→病灶。
// 病灶行+升件同窗去重（esc_class+esc_scope 等值键——同族 24h 至多一报）。
func (s *Sensor) probeFamily(ctx context.Context, template string, kinds []string) error {
	driver := kinds[0]
	var sigN, evalN int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
		WHERE kind=$1 AND ts>now()-make_interval(days=>$2)`,
		driver, silenceWindowDays).Scan(&sigN); err != nil {
		return fmt.Errorf("信号行数读败: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE template=$1 AND ts>now()-make_interval(days=>$2)`,
		template, silenceWindowDays).Scan(&evalN); err != nil {
		return fmt.Errorf("评估行数读败: %w", err)
	}
	if sigN == 0 || evalN > 0 {
		return nil // 信号死（源死归活性面非本域）/ 评估有——零病灶
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		silenceEscClass, template, silenceDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil // 同窗已报——24h 内同族至多一报
	}
	anchor, err := s.kindCounts(ctx, kinds)
	if err != nil {
		return err
	}
	note := fmt.Sprintf("template=%s 信号kind=%s 信号行数=%d 评估行数=0 窗=%dd 锚=%s",
		template, driver, sigN, silenceWindowDays, anchor)
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'族沉默',$2,$3)`,
		hungerSeat, silenceTicket, note); err != nil {
		return fmt.Errorf("族沉默行: %w", err)
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, silenceTicket,
		fmt.Sprintf("族沉默 %s：源信号 %s 近%dd 鲜行=%d 而 fin_opp_eval 该族行=0——信号活而评估零观测盲区（探针只报不修，修归常票/饥饿响应）",
			template, driver, silenceWindowDays, sigN),
		silenceEscClass, template); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}

// kindCounts 全映射 kind 窗内行数逐列（病灶行证据锚——驱动+从件全谱）。
func (s *Sensor) kindCounts(ctx context.Context, kinds []string) (string, error) {
	var b strings.Builder
	b.WriteString("md_signal 7d{")
	for i, k := range kinds {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM md_signal
			WHERE kind=$1 AND ts>now()-make_interval(days=>$2)`,
			k, silenceWindowDays).Scan(&n); err != nil {
			return "", fmt.Errorf("证据锚读败: %w", err)
		}
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%s:%d", k, n)
	}
	b.WriteString("}")
	return b.String(), nil
}
