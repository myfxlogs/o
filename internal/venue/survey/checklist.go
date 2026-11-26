// checklist.go —— 上岗普查五步检查表执行面（O-41375——T-0099 swapsurvey
// RPC 序持续化：SymbolList→逐符 SymbolParams→抽样 GetQuote→逐符
// RequiredMargin；全读面零写单）。Session=窄缝注假（生产=mt5.OpenSession
// 共享会话面——venue 参数化 roster 寻址零会话码）。
package survey

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
)

// Session 普查会话窄缝（mt5Session 生产适配满足；桩测注假——禁桩真
// venue 同律）。authed 会话面——OpenSession 产物免 Connect。
type Session interface {
	SymbolList(ctx context.Context) ([]string, error)
	AccountFacts(ctx context.Context) (login, leverage int64, ccy string, err error)
	AccountSummary(ctx context.Context) (equity, freeMargin, balance float64, err error)
	SymbolParams(ctx context.Context, symbol string) (map[string]any, error)
	Quote(ctx context.Context, symbol string) (bid, ask float64, err error)
	Margin1Lot(ctx context.Context, symbol string) (float64, error)
	Close()
}

// mt5Session 生产适配（*mt5.Client 满足——OpenSession 共享会话面复用，
// Disconnect/Close 幂等空转不伤共享会话=session.go 既有语义）。
type mt5Session struct{ c *mt5.Client }

func (s mt5Session) SymbolList(ctx context.Context) ([]string, error) {
	return s.c.SymbolList(ctx)
}
func (s mt5Session) AccountFacts(ctx context.Context) (int64, int64, string, error) {
	return s.c.AccountFacts(ctx)
}
func (s mt5Session) AccountSummary(ctx context.Context) (float64, float64, float64, error) {
	return s.c.AccountSummary(ctx)
}
func (s mt5Session) SymbolParams(ctx context.Context, sym string) (map[string]any, error) {
	return s.c.SymbolSurveyFacts(ctx, sym)
}
func (s mt5Session) Quote(ctx context.Context, sym string) (float64, float64, error) {
	q, err := s.c.GetQuoteOne(ctx, sym)
	return q.Bid, q.Ask, err
}
func (s mt5Session) Margin1Lot(ctx context.Context, sym string) (float64, error) {
	return s.c.Margin1LotBuy(ctx, sym)
}
func (s mt5Session) Close() {
	s.c.Disconnect(context.Background())
	s.c.Close()
}

// openSessionDial 生产拨号缝（mt5.OpenSession——roster cred_ns 寻址+
// 共享会话行复挂；venue 无 roster 行=err 显形由上层归类 skip）。
func openSessionDial(fin *pgxpool.Pool, cfg *config.Client) func(context.Context, string) (Session, error) {
	return func(ctx context.Context, venue string) (Session, error) {
		cl, err := mt5.OpenSession(ctx, fin, cfg, venue)
		if err != nil {
			return nil, err
		}
		return mt5Session{c: cl}, nil
	}
}

// surveyPace 逐符 RPC 步进限速（swapsurvey perSym=150ms 同款——var=桩测
// 覆写锚）；surveyBudget 单户拍死线（venue IO 帽——超时收已采行记 partial）。
var (
	surveyPace   = 150 * time.Millisecond
	surveyBudget = 10 * time.Minute
	quoteEvery   = 4 // 抽样步长——每 N 符取 1 报价（~55/218 符样本量）
)

// surveyData 检查表产物（evidence.survey 实载面——全字段机写）。
type surveyData struct {
	Account       map[string]any
	Symbols       []string
	Rows          map[string]map[string]any
	Partial       bool
	Spread        map[string]any
	Swap          map[string]any
	MarginTiers   map[string]any
	QuoteHealth   map[string]any
	FamilyVerdict map[string]any
	VerdictCount  map[string]int
	MapDraft      map[string]any
}

// collect 五步检查表执行（①SymbolList 全量→②逐符 SymbolParams→③抽样
// GetQuote→④逐符 RequiredMargin——会话已 authed 直进）。任一步结构败
// 上浮；逐符 RPC 败=行内 err 如实录不虚构不阻断。
func (r *Runner) collect(ctx context.Context, s Session) (*surveyData, error) {
	syms, err := s.SymbolList(ctx)
	if err != nil {
		return nil, fmt.Errorf("SymbolList: %w", err)
	}
	if len(syms) == 0 {
		return nil, fmt.Errorf("符号表空（SymbolList 零行——venue 侧缺陷 fail-closed）")
	}
	login, lev, ccy, err := s.AccountFacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("AccountFacts: %w", err)
	}
	eq, fm, bal, err := s.AccountSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("AccountSummary: %w", err)
	}
	d := &surveyData{
		Account: map[string]any{"login": login, "leverage": lev, "ccy": ccy,
			"equity": eq, "free_margin": fm, "balance": bal},
		Symbols: syms,
		Rows:    map[string]map[string]any{},
	}
	start := r.now()
	for i, sym := range syms {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(surveyPace): // sched-exempt-ok：逐符限频步进（venue 礼貌面——swapsurvey perSym 同款）
			}
		}
		row := map[string]any{"symbol": sym}
		if p, err := s.SymbolParams(ctx, sym); err == nil {
			for k, v := range p {
				row[k] = v
			}
		} else {
			row["params_err"] = err.Error()
		}
		if i%quoteEvery == 0 { // 抽样 GetQuote 活性（spec③——非全量防 venue 压）
			if bid, ask, err := s.Quote(ctx, sym); err == nil {
				row["bid"], row["ask"] = bid, ask
				row["quote_ok"] = bid > 0 && ask > 0
			} else {
				row["quote_err"] = err.Error()
			}
		}
		if m, err := s.Margin1Lot(ctx, sym); err == nil {
			row["margin_1lot"] = m
		} else {
			row["margin_err"] = err.Error()
		}
		d.Rows[sym] = row
		if r.now().Sub(start) > surveyBudget {
			d.Partial = true
			break // 预算截——已采行照实落 partial 标
		}
	}
	d.Spread = spreadAgg(d.Rows)
	d.Swap = swapAgg(d.Rows)
	d.MarginTiers = marginTiers(d.Rows, eq)
	d.QuoteHealth = quoteHealth(d.Rows)
	return d, nil
}

// spreadAgg spread_pts 谱（min/p50/p90/max/零点差计数——grid 判据原料）。
func spreadAgg(rows map[string]map[string]any) map[string]any {
	var vals []float64
	zero := 0
	for _, r := range rows {
		v, ok := num(r["spread_pts"])
		if !ok {
			continue
		}
		vals = append(vals, v)
		if v == 0 {
			zero++
		}
	}
	if len(vals) == 0 {
		return map[string]any{"n": 0}
	}
	sort.Float64s(vals)
	pick := func(q float64) float64 { return vals[int(q*float64(len(vals)-1))] }
	return map[string]any{"n": len(vals), "min": vals[0], "p50": pick(0.5),
		"p90": pick(0.9), "max": vals[len(vals)-1], "zero": zero}
}

// swapAgg swap 三值分布（swap_type 计数+long/short 非零计数+三日换仓日分布——
// carry 判据原料）。
func swapAgg(rows map[string]map[string]any) map[string]any {
	types := map[string]int{}
	triple := map[string]int{}
	longNZ, shortNZ := 0, 0
	for _, r := range rows {
		if v, ok := r["swap_type"].(string); ok && v != "" {
			types[v]++
		}
		if v, ok := r["triple_swap_day"].(string); ok && v != "" {
			triple[v]++
		}
		if f, ok := num(r["swap_long"]); ok && f != 0 {
			longNZ++
		}
		if f, ok := num(r["swap_short"]); ok && f != 0 {
			shortNZ++
		}
	}
	return map[string]any{"type": types, "triple_day": triple,
		"long_nonzero": longNZ, "short_nonzero": shortNZ}
}

// marginTiers margin_1lot 对户头权益分层（T-0099 同款≤$100/≤$500/≤权益50%/>——
// 判据原料非决策）。
func marginTiers(rows map[string]map[string]any, equity float64) map[string]any {
	le100, le500, leEq, gt, noData := 0, 0, 0, 0, 0
	for _, r := range rows {
		m, ok := num(r["margin_1lot"])
		if !ok || m <= 0 {
			noData++
			continue
		}
		switch {
		case m <= 100:
			le100++
		case m <= 500:
			le500++
		case equity > 0 && m <= equity*0.5:
			leEq++
		default:
			gt++
		}
	}
	return map[string]any{"le_100": le100, "le_500": le500, "le_eq50": leEq,
		"gt": gt, "no_data": noData, "equity": equity}
}

// quoteHealth 抽样报价活性（>20% 死价=品控降等 grade=degraded——spec③阈值
// 钦定面内自裁值，判词原料非闸）。
func quoteHealth(rows map[string]map[string]any) map[string]any {
	sampled, alive := 0, 0
	for _, r := range rows {
		ok, has := r["quote_ok"].(bool)
		if !has {
			continue
		}
		sampled++
		if ok {
			alive++
		}
	}
	if sampled == 0 {
		return map[string]any{"sampled": 0}
	}
	deadPct := float64(sampled-alive) / float64(sampled)
	grade := "ok"
	if deadPct > 0.2 {
		grade = "degraded"
	}
	return map[string]any{"sampled": sampled, "alive": alive,
		"dead_pct": deadPct, "grade": grade}
}

// num jsonb 数值归一（int64/float64/json.Number→float64,ok）。
func num(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	case int:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}
