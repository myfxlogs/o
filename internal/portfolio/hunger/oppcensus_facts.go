package hunger

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type futInst struct {
	inst     string
	expiry   time.Time
	daysLeft float64
}

type censusRow struct {
	inst      string
	mechanism string
	evalInst  string // eval 臂判据键 inst（机制实际交易腿——SWAP 行与现货行同机会共享 perp 键，防双升件）
	legs      map[string]any
	netBps    *float64
	verdict   string
	evidence  map[string]any
}

// instParts instId 三段解（OKX 方言：BASE-QUOTE[-SWAP|-YYMMDD]——
// 三段尾段 'SWAP'=永续、六数字=到期 FUT、两段=现货）。
func instParts(inst string) (base, quote string, kind string) {
	segs := strings.Split(inst, "-")
	if len(segs) >= 2 {
		base, quote = segs[0], segs[1]
	}
	switch {
	case len(segs) == 3 && segs[2] == "SWAP":
		kind = "swap"
	case futSuffixRe.MatchString(inst):
		kind = "fut"
	case len(segs) == 2:
		kind = "spot"
	default:
		kind = "other"
	}
	return base, quote, kind
}

// futExpiry FUT 尾段 YYMMDD→到期日（2000+YY 归一）。
func futExpiry(inst string) (time.Time, bool) {
	segs := strings.Split(inst, "-")
	if len(segs) != 3 || !futSuffixRe.MatchString(inst) {
		return time.Time{}, false
	}
	d := segs[2]
	if len(d) != 6 {
		return time.Time{}, false
	}
	yy, _ := strconv.Atoi(d[0:2])
	mm, _ := strconv.Atoi(d[2:4])
	dd, _ := strconv.Atoi(d[4:6])
	t := time.Date(2000+yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC)
	return t, t.Year() > 2000
}

// loadFacts 五源装载（腿矩阵原料——主源读败整体败上抛；格级事实缺
// 如 spread 样本不足/价缺走逐格「不可判」不虚构）。
func (c *censusCtx) loadFacts(ctx context.Context, pool *pgxpool.Pool, now time.Time) error {
	// 宇宙=md_funding(okx)∪md_ticks(okx)——spec 钉死两源并集。
	rows, err := pool.Query(ctx,
		`SELECT inst_id FROM md_funding WHERE venue='okx'
		 UNION SELECT inst_id FROM md_ticks WHERE venue='okx' ORDER BY 1`)
	if err != nil {
		return fmt.Errorf("普查宇宙读败: %w", err)
	}
	c.instSet = map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		c.universe = append(c.universe, id)
		c.instSet[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// 费率事实=md_funding 每 inst 末行（ts 降序首行；间隔=next_funding_ts-ts，
	// 缺下一档戳回落 8h OKX 标准结算档——evidence 记口径）。
	frows, err := pool.Query(ctx,
		`SELECT DISTINCT ON (inst_id) inst_id,funding_rate,next_funding_ts,ts
		 FROM md_funding WHERE venue='okx' ORDER BY inst_id,ts DESC`)
	if err != nil {
		return fmt.Errorf("普查费率面读败: %w", err)
	}
	c.funding = map[string]fundFact{}
	for frows.Next() {
		var id string
		var rate float64
		var next, ts *time.Time
		if err := frows.Scan(&id, &rate, &next, &ts); err != nil {
			frows.Close()
			return err
		}
		ivl := 8.0
		if next != nil && ts != nil {
			if h := next.Sub(*ts).Hours(); h > 0 && h <= 24 {
				ivl = h
			}
		}
		c.funding[id] = fundFact{rate: rate, intervalH: ivl,
			apyBps: rate * 1e4 * (8760 / ivl)}
	}
	frows.Close()
	if err := frows.Err(); err != nil {
		return err
	}
	// md_ticks 当拍价/spread（bid/ask 双在才计 spread——单边缺=has 价在 spread 不可判）。
	trows, err := pool.Query(ctx,
		`SELECT inst_id,last,bid,ask FROM md_ticks WHERE venue='okx'`)
	if err != nil {
		return fmt.Errorf("普查 ticks 面读败: %w", err)
	}
	c.ticks = map[string]tickFact{}
	for trows.Next() {
		var id string
		var last, bid, ask *float64
		if err := trows.Scan(&id, &last, &bid, &ask); err != nil {
			trows.Close()
			return err
		}
		tf := tickFact{has: last != nil}
		if last != nil {
			tf.last = *last
		}
		if bid != nil && ask != nil {
			tf.bid, tf.ask = *bid, *ask
		}
		c.ticks[id] = tf
	}
	trows.Close()
	if err := trows.Err(); err != nil {
		return err
	}
	// CFD venue=roster proto='cfd'（平台子属性 platform 不入判据——新经纪商
	// 接入=插 roster 行零码改动）；CFD 在=md_ticks 该 venue 有 base+'USD%'
	// 前缀符号实证（exness 'BTCUSDm'/xmglobal 'BTCUSD#' 方言同前缀判据）。
	cfdRows, err := pool.Query(ctx,
		`SELECT DISTINCT venue FROM fin_venue_roster WHERE proto='cfd'`)
	if err != nil {
		return fmt.Errorf("普查 CFD roster 读败: %w", err)
	}
	for cfdRows.Next() {
		var v string
		if err := cfdRows.Scan(&v); err != nil {
			cfdRows.Close()
			return err
		}
		c.cfdVenues = append(c.cfdVenues, v)
	}
	cfdRows.Close()
	if err := cfdRows.Err(); err != nil {
		return err
	}
	sort.Strings(c.cfdVenues)
	c.cfdMatch = map[string][]string{}
	for _, base := range c.instBases() {
		var hit []string
		for _, v := range c.cfdVenues {
			var ok bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM md_ticks
				 WHERE venue=$1 AND inst_id LIKE $2)`,
				v, base+"USD%").Scan(&ok); err != nil {
				return fmt.Errorf("普查 CFD 符号面读败 venue=%s base=%s: %w", v, base, err)
			}
			if ok {
				hit = append(hit, v)
			}
		}
		if len(hit) > 0 {
			c.cfdMatch[base] = hit
		}
	}
	// FUT 索引：宇宙内尾段六数字未到期者按 base 归组（basis 腿解析）。
	c.futByBase = map[string][]futInst{}
	c.futSet = map[string]bool{}
	for _, id := range c.universe {
		base, _, kind := instParts(id)
		if kind != "fut" {
			continue
		}
		exp, ok := futExpiry(id)
		if !ok {
			continue
		}
		days := exp.Sub(now).Hours() / 24
		if days <= 0 {
			continue // 已到期不入候选（结构死由判定面如实报）
		}
		c.futSet[id] = true
		c.futByBase[base] = append(c.futByBase[base],
			futInst{inst: id, expiry: exp, daysLeft: days})
	}
	for b := range c.futByBase {
		sort.Slice(c.futByBase[b], func(i, j int) bool {
			return c.futByBase[b][i].daysLeft < c.futByBase[b][j].daysLeft
		})
	}
	// spread 样本续存：opp_map evidence->spread_samples 每 inst 一份
	// （四机制行同值冗余存——读取取任一行即可；写回各行自 inst 样本）。
	c.spreadSmp = map[string][]float64{}
	return c.loadSpreadSamples(ctx, pool)
}

// instBases 宇宙 base 去重集（CFD 符号面预取键）。
func (c *censusCtx) instBases() []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range c.universe {
		b, _, _ := instParts(id)
		if b != "" && !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	sort.Strings(out)
	return out
}

// loadSpreadSamples opp_map evidence 续存样本装载（每 inst 取机制序首行
// 样本组——四行冗余同值；新 inst 无行=空样本本拍起计）。
func (c *censusCtx) loadSpreadSamples(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT ON (inst_id) inst_id, evidence->'spread_samples'
		 FROM fin_opp_map ORDER BY inst_id, mechanism`)
	if err != nil {
		return fmt.Errorf("普查 spread 样本续存读败: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return err
		}
		var smp []float64
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &smp); err != nil {
				return fmt.Errorf("spread 样本解败 inst=%s: %w", id, err)
			}
		}
		c.spreadSmp[id] = smp
	}
	return rows.Err()
}
