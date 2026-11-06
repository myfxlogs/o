// depenum.go —— T-0394 依赖对枚举面（depblind.go 对账拍的供对侧）：
// 枚举器实扫注册表/归一链产依赖对（零硬编码清单——票面红线），
// 谱面清单落账供人工基线。枚举源：
//
//	①信号对=kind→eval template 两链派生（0384 同款假想卡 data_needs
//	  注册表链+template 命名规约链）+两链皆不属的 orphan kind；
//	②腿-行情对=legset.ActiveSetUnion 同源表达式∩订阅宇宙；
//	③表读写对=读面注册谱（depTablePairs 钉定对——T-0405 启用；
//	  产表窗内有写而消表窗内零行=表读写失明）；
//	④配置对=app_config 键域谱面枚举（depConfigReaders 读面册——
//	  域在册而读面册外=产而活零读死键，orphan 形升件）。
package hunger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"o/internal/platform/legset"
)

// enumerateDeps 依赖对枚举（实扫注册表/归一链——零硬编码清单）：
// ①信号对=kind→eval template 两链派生+消费册外 orphan kind；②腿对=
// legset.ActiveSetUnion∩订阅宇宙；③表对=depTablePairs 钉定谱（读面注册——
// 票面钦三对首谱）；④配置对=app_config 键域×depConfigReaders 读面册。
func (s *Sensor) enumerateDeps(ctx context.Context) ([]depPair, error) {
	var pairs []depPair
	kindSet, err := s.signalKindSet(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对 kind 谱读败: %w", err)
	}
	needs, err := s.hypothesisNeeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对假想卡需求读败: %w", err)
	}
	templates, err := s.evalTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("信号对模板全集读败: %w", err)
	}
	claimed := map[string]bool{}
	for _, t := range templates {
		for _, k := range deriveKinds(t, kindSet, needs) {
			claimed[k] = true
			pairs = append(pairs, depPair{
				sig: "signal:" + k + "→" + t, family: "signal",
				producer: k, consumer: t})
		}
	}
	var orphans []string
	for k := range kindSet {
		if !claimed[k] {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	for _, k := range orphans {
		pairs = append(pairs, depPair{
			sig: "signal:" + k + "→∅", family: "signal",
			producer: k, consumerAbsent: true})
	}
	legs, err := s.depLegKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("腿对键集读败: %w", err)
	}
	for _, inst := range legs {
		pairs = append(pairs, depPair{
			sig: "leg:" + inst, family: "leg",
			producer: inst, consumer: inst})
	}
	// ③表读写对：钉定谱逐对入列（产者窗内写>0 而消费者窗内零行=失明——
	// 「paper 策略活而 fin_paper_fill 零行」正落本族罩位）。
	for _, spec := range depTablePairs {
		pairs = append(pairs, depPair{
			sig: spec.sig, family: "table",
			producer: spec.prod, consumer: spec.cons,
			prodSQL: spec.prodSQL, consSQL: spec.consSQL})
	}
	// ④配置对：app_config 键域谱面枚举——域在册（键存在=产者已写）而
	// 读面册外=产而活零读死键（orphan 形）；册内域=常对（键域有册载
	// 读者——枚举面留账供谱面审计，恒不盲）。
	domains, err := s.configDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("配置对键域谱读败: %w", err)
	}
	for _, d := range domains {
		if reader, ok := depConfigReaders[d]; ok {
			pairs = append(pairs, depPair{
				sig: "config:" + d + "→" + reader, family: "config",
				producer: d, consumer: reader})
		} else {
			pairs = append(pairs, depPair{
				sig: "config:" + d + "→∅", family: "config",
				producer: d, consumerAbsent: true})
		}
	}
	return pairs, nil
}

// depUniverse 订阅宇宙（okx.symbols 名单——腿对所需键的合法性边界；
// '*' 放形 fail-closed 让位：宇宙扩件=venue REST 实读闭包，本探针不虚构）。
func (s *Sensor) depUniverse() (map[string]bool, error) {
	raw, err := s.cfg.Get("okx.symbols")
	if err != nil {
		return nil, err
	}
	var all []string
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("okx.symbols JSON 解析: %w", err)
	}
	out := make(map[string]bool, len(all))
	for _, sym := range all {
		if sym == "*" {
			return nil, errors.New("okx.symbols=* 宇宙扩件探针未挂——fail-closed 不虚构标的集")
		}
		out[sym] = true
	}
	return out, nil
}

// depLegKeys 腿-行情对所需键集（归一链=legset.ActiveSetUnion 同源表达式
// ——订阅集/镜像谓词同串禁第二套；∩订阅宇宙+IsUSDTInst 闸=派生合法性滤，
// 虚构键宇宙外不入所需集——同 syncActiveSet 加订闸判据语义）。
func (s *Sensor) depLegKeys(ctx context.Context) ([]string, error) {
	universe, err := s.depUniverse()
	if err != nil {
		return nil, fmt.Errorf("宇宙读败: %w", err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT inst_id FROM (`+legset.ActiveSetUnion+`) t`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		if legset.IsUSDTInst(k) && universe[k] {
			out = append(out, k)
		}
	}
	return out, rows.Err()
}

// depCensus 枚举器产出对清单落账（首跑人工基线材料——「依赖对清单」行
// 携全谱对签名；谱面不变不重复落行——最新行 note 全等比对，零膨胀）。
func (s *Sensor) depCensus(ctx context.Context, pairs []depPair) error {
	sigs := make([]string, 0, len(pairs))
	for _, p := range pairs {
		sigs = append(sigs, p.sig)
	}
	sort.Strings(sigs)
	note := fmt.Sprintf("pairs=%d sigs=%s", len(pairs), strings.Join(sigs, ","))
	var last string
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT note FROM ops_event
		WHERE event='依赖对清单' ORDER BY id DESC LIMIT 1),'')`).Scan(&last); err != nil {
		return err
	}
	if last == note {
		return nil // 谱面未变——不重复落行
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'依赖对清单',$2,$3)`,
		hungerSeat, depblindTicket, note)
	return err
}

// depTableSpec 表读写对钉定谱元（T-0405 ③族——读面注册：对签名+产/消
// 窗内行数 SQL 同列；$1=窗天数。SQL 皆本文件内常量非外部输入）。
type depTableSpec struct {
	sig     string // 对签名（dep_blind key/esc_scope）
	prod    string // 产者键（note/dedup 自然键）
	cons    string // 消费者键
	prodSQL string // 产者窗内行数（$1=窗天数）
	consSQL string // 消费者窗内行数（$1=窗天数）
}

// depTablePairs ③族首谱（票面钦三对钉定——白名单豁面照旧 depBlindWhitelist）：
//
//	paper→fill：paper 策略在册逾窗（产者活——应产 paper 记账）而
//	  fin_paper_fill 窗内零行=失明（0394 deferred ③正案——「paper 策略活
//	  而 fin_paper_fill 零行」罩位；entered_at<窗 免新生 paper 误报）；
//	ops_event→链：写者窗内有行而链写覆写（hash 非空）窗内零行=消费全死
//	  （部分断归 ops_chain_verify 裸行检出——本对罩「链消费全灭」形）；
//	eval→proposal：评估窗内有产而提案窗内零行=评估→提案链失明
//	  （与饥饿 Detect zero_proposal 同病异面——本件=产消对粒度，对签名
//	  去重不与 Detect 互吸收）。
var depTablePairs = []depTableSpec{
	{"table:fin_strategy[paper]→fin_paper_fill",
		"fin_strategy[paper]", "fin_paper_fill",
		`SELECT count(*) FROM fin_strategy WHERE state='paper'
			AND entered_at<now()-make_interval(days=>$1)`,
		`SELECT count(*) FROM fin_paper_fill WHERE ts>now()-make_interval(days=>$1)`},
	{"table:ops_event→ops_event[chain]",
		"ops_event", "ops_event[chain]",
		`SELECT count(*) FROM ops_event WHERE ts>now()-make_interval(days=>$1)`,
		`SELECT count(*) FROM ops_event WHERE ts>now()-make_interval(days=>$1) AND hash IS NOT NULL`},
	{"table:fin_opp_eval→fin_proposal",
		"fin_opp_eval", "fin_proposal",
		`SELECT count(*) FROM fin_opp_eval WHERE ts>now()-make_interval(days=>$1)`,
		`SELECT count(*) FROM fin_proposal WHERE created_at>now()-make_interval(days=>$1)`},
}

// depConfigReaders ④族读面册（键域→册载读者标签——读面注册同 depTablePairs
// 钉定例）：域源=scripts/config-write-domain.tsv 写域册全谱+迁移种键/运行期
// DMSet 域。册外域现身 app_config=产而活零读死键→orphan 升件（合法死键走
// depBlindWhitelist 注册豁免——新域初报一次后人工过基线，非暗吞）。
var depConfigReaders = map[string]string{
	// —— 写域册 dm 类（「如何去赚」机器自主域——读者=对应执行/评估环）——
	"research":  "研究评估环",
	"guard":     "守仓执行",
	"invest":    "lifecycle 放行线",
	"scout":     "venue 测榜",
	"lifecycle": "生命周期 TTL",
	"strategy":  "降级 TTL",
	"gate":      "退役闸族",
	"exec":      "执行参数",
	"funding":   "funding 扫描",
	"gapscan":   "gap 扫描",
	"dm":        "DM 环自参数",
	"earn":      "Earn 资金运用",
	"trx":       "Tron Earn 族",
	// —— 写域册 owner 类（业主面键——读者=对应基建/钱径面）——
	"risk":     "资金上限/停机总闸",
	"ledger":   "对账容差",
	"feishu":   "业主通知通道",
	"notify":   "通知投递面",
	"bridge":   "采集基建面",
	"calendar": "宏观日历面",
	"cmc":      "CMC 采集面",
	"net":      "带宽预算",
	"mt5":      "mt5 网关面",
	"okx":      "okx 采集宇宙",
	"readcred": "读凭据权限面",
	// —— 册外运行期种键域（迁移种键在案——读者=对应清道夫/采拍面）——
	"data": "数据保留/陈阈面",
}

// configDomains app_config 键域谱（首段域枚举——④族枚举源；
// 'key.with.dots' 首段=域界）。
func (s *Sensor) configDomains(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT split_part(key,'.',1) FROM app_config ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
