package hunger

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// unexploited_opp 升件类（O-39480 细胞B——机会面普查眼：
// 「桌上有菜没点」——可执行∧净估超阈∧eval臂未评格才升，只述事实）。
const censusEscClass = "unexploited_opp"

// censusEventMarker 普查拍成功标记行（ops_event append-only——拍距门
// 读面=本事件末行 ts；写败不标门=下拍重试全量幂等 upsert 兜底）。
const censusEventMarker = "opp_census"

// CensusFacts 普查公共事实源（hunger 侧中性形——import 方向闸禁
// trading/okxexec；cmd/core 装配层适配 okxexec.CensusFacts→本形）。
type CensusFacts struct {
	SpotLive   map[string]bool    // OKX live 现货 instId 集（/public/instruments SPOT）
	BorrowRate map[string]float64 // ccy → 小时借币率 decimal（/public/interest-rate-loan-quota）
}

// censusFeedFunc 供给件缝（AttachUniverse 同构——nil=未挂 fail-closed）。
type censusFeedFunc func(ctx context.Context) (*CensusFacts, error)

// censusSpreadCap 样本窗上限（6h 拍×48=12d——中位数稳态足够，防 jsonb 膨胀）。
const censusSpreadCap = 48

// censusMechTemplates 机制族→eval template 归一名册（eval 臂「从未评过」
// 判据键面——replace(_→-) 双向归一同 breedstall 先例；族外模板不抑升件）。
var censusMechTemplates = map[string][]string{
	"funding":  {"funding-hedge", "funding-extreme-fade", "xvenue-carry", "lending-spread", "earn-term-spread"},
	"grid":     {"grid-spot"},
	"basis":    {"quarter-basis-long", "quarter-basis-short", "perp-quarter-spread"},
	"momentum": {"dir-perp-mom", "mt5-dir-mom", "perp-trend-daily", "xsec-perp-mom-basket"},
}

var futSuffixRe = regexp.MustCompile(`-\d{6}$`)

// censusCfg 普查阈件包（全 cfg 读 fail-closed——fin_0263 种键）。
type censusCfg struct {
	ivlH    int64
	minNet  float64
	escMax  int64
	spreadN int64
	dedupH  int64
}

// ProbeOppCensus 机会面普查眼一拍（hungerTick 同槽——ProbeDecompWatch
// 后 Detect 前；败情隔离位同款）：拍距门 census_ivl_h——窗内已普查
// 静默返回（Quiet 语义=合理零产出拍不误报）；供给件未挂/读败/DB 败
// =上抛归 hungerTick 自报。枚举=md_funding(okx)∪md_ticks(okx) inst
// ×四机制族，逐格判定 upsert fin_opp_map（幂等键 inst+mechanism，
// 陈值 stale 非删）；升件只挑 executable∧净估>阈∧eval 未评格，
// 单拍≤census_esc_max。
func (s *Sensor) ProbeOppCensus(ctx context.Context) error {
	cfg, err := censusLoadCfg(s)
	if err != nil {
		return err
	}
	var last *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(ts) FROM ops_event WHERE event=$1`, censusEventMarker).Scan(&last); err != nil {
		return fmt.Errorf("普查拍距门读败: %w", err)
	}
	if last != nil && s.now().Sub(*last) < time.Duration(cfg.ivlH)*time.Hour {
		return nil // 窗内已普查——Quiet 语义
	}
	if s.censusFeed == nil {
		return fmt.Errorf("普查供给件未挂（census feed nil——fail-closed）")
	}
	facts, err := s.censusFeed(ctx)
	if err != nil {
		return fmt.Errorf("普查事实源读败: %w", err)
	}
	if facts == nil {
		return fmt.Errorf("普查事实源返 nil（供给件契约违例——fail-closed）")
	}
	c := &censusCtx{facts: facts, cfg: cfg}
	if err := c.loadFacts(ctx, s.pool, s.now()); err != nil {
		return err
	}
	c.evalAll()
	if err := c.persist(ctx, s.pool); err != nil {
		return err
	}
	var execN, deadN, negN, unjN int
	for _, r := range c.rows {
		switch r.verdict {
		case "executable":
			execN++
		case "structurally-dead":
			deadN++
		case "net-negative":
			negN++
		default:
			unjN++
		}
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,$2,$3,$4)`,
		hungerSeat, censusEventMarker, hungerTicket,
		fmt.Sprintf("oppCensus 拍毕 inst=%d cells=%d 判定 exec=%d dead=%d neg=%d unj=%d",
			len(c.universe), len(c.rows), execN, deadN, negN, unjN)); err != nil {
		return fmt.Errorf("普查拍标记落败: %w", err)
	}
	return c.escalate(ctx, s, cfg)
}

// censusLoadCfg 阈件全读（缺/坏/非正=fail-closed 上抛——探针不可带默认阈滑行）。
func censusLoadCfg(s *Sensor) (censusCfg, error) {
	var c censusCfg
	var err error
	if c.ivlH, err = s.cfg.GetInt("explore.census_ivl_h"); err != nil {
		return c, fmt.Errorf("普查拍距键不可读: %w", err)
	}
	if c.minNet, err = s.cfg.GetFloat("explore.census_min_net_bps"); err != nil {
		return c, fmt.Errorf("普查净估阈键不可读: %w", err)
	}
	if c.escMax, err = s.cfg.GetInt("explore.census_esc_max"); err != nil {
		return c, fmt.Errorf("普查升件上限键不可读: %w", err)
	}
	if c.spreadN, err = s.cfg.GetInt("explore.census_spread_min_n"); err != nil {
		return c, fmt.Errorf("普查 spread 样本阈键不可读: %w", err)
	}
	if c.dedupH, err = s.cfg.GetInt("explore.census_esc_dedup_h"); err != nil {
		return c, fmt.Errorf("普查升件抑复窗键不可读: %w", err)
	}
	if c.ivlH <= 0 || c.minNet <= 0 || c.escMax <= 0 || c.spreadN <= 0 || c.dedupH <= 0 {
		return c, fmt.Errorf("普查阈件非正（ivl=%d minNet=%.1f escMax=%d spreadN=%d dedup=%d——fail-closed）",
			c.ivlH, c.minNet, c.escMax, c.spreadN, c.dedupH)
	}
	return c, nil
}

// censusCtx 单拍上下文（事实装载→逐格判定→落册→升件）。
type censusCtx struct {
	facts     *CensusFacts
	cfg       censusCfg
	universe  []string
	instSet   map[string]bool
	funding   map[string]fundFact  // SWAP inst → 费率事实（md_funding 末行）
	ticks     map[string]tickFact  // okx inst → 价/spread 当拍值
	cfdVenues []string             // roster proto='cfd' venue
	cfdMatch  map[string][]string  // base → 命中 cfd venue 列（md_ticks 符号前缀实证）
	futByBase map[string][]futInst // base → 未到期 FUT 候选
	futSet    map[string]bool      // FUT inst 集合
	spreadSmp map[string][]float64 // inst → 累计 spread bps 样本（opp_map evidence 续存）
	rows      []censusRow
}

type fundFact struct {
	rate      float64 // 周期费率 decimal
	intervalH float64 // 结算间隔小时
	apyBps    float64 // 年化 bps 带符号（正=空头收租方向）
}

type tickFact struct {
	last, bid, ask float64
	has            bool
}
