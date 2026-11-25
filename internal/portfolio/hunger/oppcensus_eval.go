package hunger

import (
	"math"
	"sort"
)

// instLegs 单 inst 四腿解析（腿=标的层事实投影到宇宙键——SWAP 行与其
// 现货行腿集合同属一个机会实体；判定按机制所需腿查）。
type instLegs struct {
	base, quote, kind string
	spotInst          string // base-USDT 候选现货键
	spotBuyable       bool
	borrowable        bool
	perpInst          string // 解析出的 SWAP 键（自身份或 base-USDT-SWAP）
	perpExists        bool
	cfdListed         []string
	fut               *futInst // basis 腿：本行 FUT 自身或同 base 最近未到期 FUT
}

// legsFor 腿解析（供给件 facts 读毕后纯函数——桩测可直调）。
func (c *censusCtx) legsFor(inst string) instLegs {
	base, quote, kind := instParts(inst)
	l := instLegs{base: base, quote: quote, kind: kind,
		spotInst: base + "-USDT"}
	l.spotBuyable = c.facts.SpotLive[l.spotInst]
	_, l.borrowable = c.facts.BorrowRate[base]
	l.cfdListed = c.cfdMatch[base]
	// perp 解析：SWAP 行=自身；余者=base-USDT-SWAP 候选（funding 集实证）。
	if kind == "swap" {
		l.perpInst = inst
	} else {
		cand := base + "-USDT-SWAP"
		if c.instSet[cand] {
			l.perpInst = cand
		}
	}
	if l.perpInst != "" {
		_, l.perpExists = c.funding[l.perpInst]
	}
	// basis 腿：FUT 行=自身（未到期才入 futSet）；余者=同 base 最近未到期。
	if kind == "fut" {
		for i := range c.futByBase[base] {
			if c.futByBase[base][i].inst == inst {
				f := c.futByBase[base][i]
				l.fut = &f
				break
			}
		}
	} else if fs := c.futByBase[base]; len(fs) > 0 {
		f := fs[0] // 已按 daysLeft 升序
		l.fut = &f
	}
	return l
}

// spreadMedian inst spread 中位 bps（样本≥spreadN 才返值——不足
// ok=false=「不可判」不虚构；样本=opp_map evidence 续存+当拍值）。
func (c *censusCtx) spreadMedian(inst string) (float64, bool) {
	smp := c.spreadSmp[inst]
	if int64(len(smp)) < c.cfg.spreadN {
		return 0, false
	}
	cp := append([]float64(nil), smp...)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2], true
	}
	return (cp[n/2-1] + cp[n/2]) / 2, true
}

// spreadBpsNow 当拍 (ask-bid)/bid×1e4（双边价在才计）。
func spreadBpsNow(t tickFact) (float64, bool) {
	if !t.has || t.bid <= 0 || t.ask <= 0 {
		return 0, false
	}
	return (t.ask - t.bid) / t.bid * 1e4, true
}

// evalAll 逐 inst×四机制判定（宇宙排序定序——重拍幂等可重现）。
func (c *censusCtx) evalAll() {
	for _, inst := range c.universe {
		l := c.legsFor(inst)
		legsJSON := map[string]any{
			"spot_inst": l.spotInst, "spot_buyable": l.spotBuyable,
			"borrowable": l.borrowable, "perp_inst": l.perpInst,
			"perp_exists": l.perpExists, "cfd_listed": l.cfdListed,
		}
		c.rows = append(c.rows, c.evalFunding(inst, l, legsJSON))
		c.rows = append(c.rows, c.evalGrid(inst, l, legsJSON))
		c.rows = append(c.rows, c.evalBasis(inst, l, legsJSON))
		c.rows = append(c.rows, c.evalMomentum(inst, l, legsJSON))
	}
}

// evalFunding funding收租：perp 存在∧方向腿齐（rate>0→现货可买收空头租；
// rate<0→可借收多头租）→净估=|费率年化|−借率年化(若用)−双腿 spread。
func (c *censusCtx) evalFunding(inst string, l instLegs, legs map[string]any) censusRow {
	r := censusRow{inst: inst, mechanism: "funding", evalInst: l.perpInst, legs: legs,
		evidence: map[string]any{"kind": l.kind}}
	if !l.perpExists {
		r.verdict = "structurally-dead"
		r.evidence["why"] = "perp缺——md_funding 无此标的永续"
		return r
	}
	ff, ok := c.funding[l.perpInst]
	if !ok {
		r.verdict = "unjudgeable"
		r.evidence["why"] = "费率事实缺"
		return r
	}
	r.evidence["funding_rate"] = ff.rate
	r.evidence["funding_interval_h"] = ff.intervalH
	r.evidence["funding_apy_bps"] = ff.apyBps
	if ff.apyBps == 0 {
		r.verdict = "net-negative"
		r.evidence["why"] = "费率零——无收租面"
		return r
	}
	// 方向：apyBps>0=空头收（rate>0→空 perp+多现货）；<0=多头收（多 perp+空现货借币）。
	direction := "short_perp_long_spot"
	var borrowApy float64
	if ff.apyBps < 0 {
		direction = "long_perp_short_spot"
		if !l.borrowable {
			r.verdict = "structurally-dead"
			r.evidence["why"] = "负费率收租需空现货——base 不可借"
			return r
		}
		if hr, ok := c.facts.BorrowRate[l.base]; ok {
			borrowApy = hr * 8760 * 1e4
			r.evidence["borrow_rate_hr"] = hr
			r.evidence["borrow_apy_bps"] = borrowApy
		} else {
			r.verdict = "unjudgeable"
			r.evidence["why"] = "借率事实缺"
			return r
		}
	} else if !l.spotBuyable {
		r.verdict = "structurally-dead"
		r.evidence["why"] = "正费率收租需多现货——现货不可买"
		return r
	}
	r.evidence["direction"] = direction
	// 执行成本=perp 腿 spread+现货腿 spread（借币腿无盘口 spread）。
	fric, ok := c.frictionBps(l.perpInst, l.spotInst, direction == "short_perp_long_spot")
	if !ok {
		r.verdict = "unjudgeable"
		r.evidence["why"] = "spread 样本不足"
		return r
	}
	r.evidence["friction_bps"] = fric
	net := math.Abs(ff.apyBps) - borrowApy - fric
	r.netBps = &net
	r.evidence["net_formula"] = "|funding_apy|-borrow_apy-friction"
	if net > 0 {
		r.verdict = "executable"
	} else {
		r.verdict = "net-negative"
	}
	return r
}

// evalGrid 网格：可交易腿在（现货可买∨perp 存在）→判定面无静态净估
// 事实源（网格利=波动率捕获——波动率/振幅事实未接入本细胞）→
// unjudgeable 如实；腿缺=structurally-dead。spread 维只入 evidence
// 不参判（无净估可扣——禁阈外虚构）。
func (c *censusCtx) evalGrid(inst string, l instLegs, legs map[string]any) censusRow {
	r := censusRow{inst: inst, mechanism: "grid", legs: legs,
		evidence: map[string]any{"kind": l.kind}}
	if l.spotBuyable {
		r.evalInst = l.spotInst
	} else {
		r.evalInst = l.perpInst
	}
	if !l.spotBuyable && !l.perpExists {
		r.verdict = "structurally-dead"
		r.evidence["why"] = "无可交易腿（现货不可买∧perp 缺）"
		return r
	}
	r.verdict = "unjudgeable"
	r.evidence["why"] = "网格利=波动率捕获——静态净估事实源未接入（禁虚构）"
	return r
}

// evalBasis basis：FUT 腿（自身或同 base 最近未到期）∧参照腿
// （perp 优先否则现货）→净估=|基差|/ref×365/daysLeft 年化−双腿 spread。
func (c *censusCtx) evalBasis(inst string, l instLegs, legs map[string]any) censusRow {
	r := censusRow{inst: inst, mechanism: "basis", legs: legs,
		evidence: map[string]any{"kind": l.kind}}
	if l.fut != nil {
		r.evalInst = l.fut.inst
	}
	if l.fut == nil {
		r.verdict = "structurally-dead"
		r.evidence["why"] = "FUT 腿缺——同 base 无未到期dated合约"
		return r
	}
	// 参照腿：perp 在→perp 价；否则现货可买→现货价；皆缺=死。
	var refInst string
	switch {
	case l.perpExists:
		refInst = l.perpInst
	case l.spotBuyable:
		refInst = l.spotInst
	default:
		r.verdict = "structurally-dead"
		r.evidence["why"] = "参照腿缺——perp 缺∧现货不可买"
		return r
	}
	futPx, ok1 := c.tickPx(l.fut.inst)
	refPx, ok2 := c.tickPx(refInst)
	if !ok1 || !ok2 || refPx <= 0 {
		r.verdict = "unjudgeable"
		r.evidence["why"] = "FUT/参照价缺"
		return r
	}
	r.evidence["fut_inst"] = l.fut.inst
	r.evidence["ref_inst"] = refInst
	r.evidence["days_left"] = l.fut.daysLeft
	basisBps := math.Abs(futPx-refPx) / refPx * 1e4
	annBps := basisBps * 365 / l.fut.daysLeft
	r.evidence["basis_bps"] = basisBps
	r.evidence["basis_apy_bps"] = annBps
	fric, ok := c.frictionBps(l.fut.inst, refInst, true)
	if !ok {
		r.verdict = "unjudgeable"
		r.evidence["why"] = "spread 样本不足"
		return r
	}
	r.evidence["friction_bps"] = fric
	net := annBps - fric
	r.netBps = &net
	r.evidence["net_formula"] = "|basis|*365/days_left-friction"
	if net > 0 {
		r.verdict = "executable"
	} else {
		r.verdict = "net-negative"
	}
	return r
}

// evalMomentum 动量：perp 存在∨CFD 在任一可执行场→unjudgeable
// （动量利=趋势捕获——无静态净估事实源）；两场皆缺=structurally-dead。
func (c *censusCtx) evalMomentum(inst string, l instLegs, legs map[string]any) censusRow {
	r := censusRow{inst: inst, mechanism: "momentum", legs: legs,
		evidence: map[string]any{"kind": l.kind}}
	if l.perpExists {
		r.evalInst = l.perpInst
	} else {
		r.evalInst = inst
	}
	if !l.perpExists && len(l.cfdListed) == 0 {
		r.verdict = "structurally-dead"
		r.evidence["why"] = "无方向场（perp 缺∧CFD 未上市）"
		return r
	}
	r.verdict = "unjudgeable"
	r.evidence["why"] = "动量利=趋势捕获——静态净估事实源未接入（禁虚构）"
	return r
}

// tickPx inst 最新价（has=false=价缺）。
func (c *censusCtx) tickPx(inst string) (float64, bool) {
	t, ok := c.ticks[inst]
	if !ok || !t.has {
		return 0, false
	}
	return t.last, true
}

// frictionBps 双腿 spread 成本合计 bps（现货腿可省——short_spot 借币
// 径无现货盘口；两腿各自样本≥spreadN 才计，不足 false=不可判）。
func (c *censusCtx) frictionBps(legA, legB string, useSpot bool) (float64, bool) {
	a, okA := c.spreadMedian(legA)
	if !okA {
		return 0, false
	}
	if !useSpot {
		return a, true
	}
	b, okB := c.spreadMedian(legB)
	if !okB {
		return 0, false
	}
	return a + b, true
}
