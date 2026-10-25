// eval.go —— 双币赢评估+阈闸（T-0102 钦定初版，判据全 config 热启）：
// dist→term→notional→单 uly 方向不叠→apr 顺闸，拒因=首触（evidence.reason）；
// 锁仓币=C 基础币/P 计价币（quoteCcy 族，USDG/USDC/RLUSD 计价产品各归各币）。
// 判据缺位=unevaluable fail-closed（禁虚构判据）。
package dcd

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"strings"
	"time"
)

// Params 阈闸判据（票面初版数值种 0043——热启可调）。
type Params struct {
	MinDistPct  float64 // strike 距现价 ≥N%
	MaxTermDays float64 // 期限 ≤N 天
	MinAPR      float64 // 年化 ≥N
	NotionalPct float64 // 名义 ≤可用币种余量×N%
}

// Eval 单品评估结论。
type Eval struct {
	Product   Product
	Template  string // dcd_call / dcd_put
	Spot      float64
	TermDays  float64
	DistPct   float64
	APR       float64
	Avail     float64 // 锁仓侧币种可动余量（C=基础币、P=计价币）
	Verdict   string  // proposed / rejected / unevaluable
	Reason    string  // 拒因/不可评估因（evidence.reason）
	lockedCcy string  // 锁仓侧币种
}

// lockedCcyOf 锁仓侧币种（C=锁基础币卖 call；P=锁计价币卖 put——quoteCcy
// 族 USDT/USDG/USDC/RLUSD，缺省回落 USDT 兼容旧夹具）。
func lockedCcyOf(p Product) string {
	if p.Direction == "C" {
		return firstSeg(p.Uly)
	}
	if p.QuoteCcy != "" {
		return p.QuoteCcy
	}
	return "USDT"
}

func firstSeg(uly string) string {
	for i := 0; i < len(uly); i++ {
		if uly[i] == '-' {
			return uly[:i]
		}
	}
	return uly
}

// Evaluate 单品评估（spot=现价 md_ticks 同源；avail=锁仓侧可动余量，
// <0=判据缺位——unevaluable fail-closed）。
func Evaluate(p Product, spot, avail float64, params Params) Eval {
	e := Eval{Product: p, Spot: spot, APR: p.AnnualizedYield,
		Avail: avail, lockedCcy: lockedCcyOf(p)}
	e.Template = "dcd_put" // venue 方言 C/P→模板名 call/put
	if p.Direction == "C" {
		e.Template = "dcd_call"
	}
	if p.Expiry.IsZero() {
		e.Verdict, e.Reason = "unevaluable", "expiry_missing"
		return e
	}
	e.TermDays = math.Ceil(time.Until(p.Expiry).Hours() / 24)
	if spot <= 0 {
		e.Verdict, e.Reason = "unevaluable", "spot_missing"
		return e
	}
	if p.Strike <= 0 {
		e.Verdict, e.Reason = "unevaluable", "strike_missing"
		return e
	}
	e.DistPct = math.Abs(p.Strike-spot) / spot * 100
	// 闸序=票面钦定顺序，拒因取首触。
	if e.DistPct < params.MinDistPct {
		e.Verdict, e.Reason = "rejected", "dist_thin"
		return e
	}
	if e.TermDays > params.MaxTermDays {
		e.Verdict, e.Reason = "rejected", "term_over"
		return e
	}
	if avail < 0 {
		e.Verdict, e.Reason = "unevaluable", "coins_absent"
		return e
	}
	if avail*params.NotionalPct/100 < p.MinSize {
		e.Verdict, e.Reason = "rejected", "insufficient_balance"
		return e
	}
	if e.APR < params.MinAPR {
		e.Verdict, e.Reason = "rejected", "apr_thin"
		return e
	}
	e.Verdict = "proposed"
	return e
}

// Evidence 证据体（评估卡 jsonb——溯源锚：productId/strike/term/dist_pct/spot）。
func (e Eval) Evidence() map[string]any {
	return map[string]any{
		"productId": e.Product.ProductID, "uly": e.Product.Uly,
		"strike": e.Product.Strike, "term": e.TermDays,
		"dist_pct": e.DistPct, "spot": e.Spot, "apr": e.APR,
		"locked_ccy": e.lockedCcy, "avail": e.Avail,
		"min_size": e.Product.MinSize, "reason": e.Reason,
	}
}

// Legs 模板语义标注（legs jsonb——非腿对冲族：锁仓卖权 carry）。
func (e Eval) Legs() map[string]any {
	if e.Product.Direction == "C" {
		return map[string]any{"locked": e.lockedCcy, "sell": "call", "settle": "expiry"}
	}
	return map[string]any{"locked": e.lockedCcy, "sell": "put", "settle": "expiry"}
}

// spot uly 现价（md_ticks 最新——与卡面/对拍同源估值面）。
// spot uly 现价（T-0117 返修③/④——uly 方言归一：OKX DCD uly=标的指数名，
// USD 指数族含 BTC-USD/BTC-USDG（USDG=USDG 稳定币计价族，标的仍是 BTC），
// 行情只采 *-USDT/*-USDT-SWAP——okx-api-spec 符号方言同族坑。归一序
// uly→uly+T→uly+T-SWAP→uly-SWAP，USDG 尾再补 USDG→USDT 变体，首命中即回）。
func (r *Runner) spot(ctx context.Context, uly string) float64 {
	cands := []string{uly, uly + "T", uly + "T-SWAP", uly + "-SWAP"}
	if alt := strings.Replace(uly, "USDG", "USDT", 1); alt != uly {
		cands = append(cands, alt, alt+"-SWAP")
	}
	var last *float64
	for _, inst := range cands {
		_ = r.pool.QueryRow(ctx, `SELECT last FROM md_ticks WHERE inst_id=$1
			ORDER BY ts DESC LIMIT 1`, inst).Scan(&last)
		if last != nil {
			return *last
		}
	}
	return 0
}

// availOf 锁仓侧可动余量（C=基础币、P=计价币 quoteCcy）；读件缺=ok=false
// （unevaluable）。
func (r *Runner) availOf(ctx context.Context, p Product) (float64, bool) {
	if r.coins == nil {
		return 0, false
	}
	coins, err := r.coins(ctx)
	if err != nil {
		r.rej.Inc("dcd_coins_absent")
		return 0, false
	}
	if p.Direction == "P" {
		return coins[lockedCcyOf(p)], true
	}
	return coins[firstSeg(p.Uly)], true
}

// directionStacked 单 uly 方向不叠闸（在途提案/策略任一在案=叠）。
func (r *Runner) directionStacked(ctx context.Context, uly, tmpl string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM fin_proposal
		WHERE direction=$2 AND (inst_id=$1 OR inst_id LIKE $1||'-%')
		AND state IN ('fin_proposed','fin_armed'))
		+ (SELECT count(*) FROM fin_strategy
		WHERE direction=$2 AND (inst_id=$1 OR inst_id LIKE $1||'-%')
		AND state<>'retired')`, uly, tmpl).Scan(&n)
	return n > 0, err
}

// persistProduct 在架产品状态行（T-0123 状态表化——产品键 upsert 末帧覆盖，
// 在架集=键域全量；评估可溯源照旧）。
func (r *Runner) persistProduct(ctx context.Context, s Surface, p Product, spot float64) error {
	var raw json.RawMessage
	if len(p.Raw) > 0 {
		raw = p.Raw
	} else {
		raw = json.RawMessage("{}")
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,
		direction,strike,expiry,annualized_yield,min_size,max_size,spot,raw)
		VALUES(now(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (venue,product_id) DO UPDATE SET ts=EXCLUDED.ts,surface=EXCLUDED.surface,
		uly=EXCLUDED.uly,direction=EXCLUDED.direction,strike=EXCLUDED.strike,
		expiry=EXCLUDED.expiry,annualized_yield=EXCLUDED.annualized_yield,
		min_size=EXCLUDED.min_size,max_size=EXCLUDED.max_size,spot=EXCLUDED.spot,raw=EXCLUDED.raw`,
		s.Venue, s.Surface, p.Uly, p.ProductID, p.Direction,
		nullF(p.Strike), nullT(p.Expiry), nullF(p.AnnualizedYield),
		nullF(p.MinSize), nullF(p.MaxSize), nullF(spot), raw)
	return err
}

// persistEval 评估卡落表；proposed 且无 open 同品提案→fin_proposal 行（闸后唯一提案径）。
func (r *Runner) persistEval(ctx context.Context, e Eval) error {
	legs, _ := json.Marshal(e.Legs())
	ev, _ := json.Marshal(e.Evidence())
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,
		gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		e.Product.ProductID, e.Template, e.Template, nullF(e.APR), nullF(e.APR),
		e.Verdict, legs, ev).Scan(&evalID); err != nil {
		return err
	}
	if e.Verdict != "proposed" {
		if e.Reason != "" {
			log.Printf("[research/dcd] %s 闸拒 %s（%s）", e.Product.ProductID, e.Verdict, e.Reason)
		}
		return nil
	}
	var open int
	if err := r.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id=$1 AND state='fin_proposed'",
		e.Product.ProductID).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil // 已挂提案不重复提案
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, e.Product.ProductID, e.Template, nullF(e.APR))
	if err == nil {
		log.Printf("[research/dcd] 提案落卡 %s %s apr=%.4f（验证官闸过）",
			e.Product.ProductID, e.Template, e.APR)
	}
	return err
}
