package main

// xvenue_test.go —— T-0145 跨所 carry untradeable 联动桩（DM 裁定锚④⑤）：
// mt5 腿终局拒→cfd canonical 入册+双腿+armed 提案退役三件；黑名单预检命中=
// 零 venue 调用退三件；终局码双源分流（okx→okx inst 入册/mt5→cfd canonical）。
import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/findom"
	"o/internal/portfolio/lifecycle"
	"o/internal/risk"
	"o/internal/trading/execution"
	"o/internal/trading/mtexec"
	"o/internal/trading/okxexec"
)

// mkXvenuePair xvenue 双行提案夹具——主行 venue='okx'+副腿行 venue='mt5'
// 同 proposal_ref（提案双行建序钦定——xvenuepair.go）。返主/副策略 id。
func mkXvenuePair(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	okxInst, cfdInst string) (mainID, mateID int64) {
	t.Helper()
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES($1,'xvenue-carry','long_perp_short_cfd',0.08,'proposed','[]','{}') RETURNING id`, okxInst).Scan(&eid); err != nil {
		t.Fatal("评估夹具:", err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,$2,'long_perp_short_cfd','fin_armed',0) RETURNING id`, eid, okxInst).Scan(&pid); err != nil {
		t.Fatal("提案夹具:", err)
	}
	for _, row := range []struct{ inst, venue string }{{okxInst, "okx"}, {cfdInst, "mt5"}} {
		var sid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
			VALUES($1,'xvenue-carry',$2,'long_perp_short_cfd',50,'live',$3,'{}') RETURNING id`,
			row.inst, pid, row.venue).Scan(&sid); err != nil {
			t.Fatal("策略夹具:", err)
		}
		if row.venue == "okx" {
			mainID = sid
		} else {
			mateID = sid
		}
	}
	return mainID, mateID
}

// xvSpec 主行开仓件（xvenue-carry 主行 venue='okx'——副腿行永不独立开仓）。
func xvSpec(mainID int64, okxInst string) lifecycle.OpenSpec {
	return lifecycle.OpenSpec{StrategyRef: mainID, Venue: "okx", InstID: okxInst,
		Direction: "long_perp_short_cfd", NotionalUSD: 50}
}

func xvAssertRetired(t *testing.T, ctx context.Context, pool *pgxpool.Pool, mainID, mateID int64, okxInst string) {
	t.Helper()
	for _, sid := range []int64{mainID, mateID} {
		var st string
		if err := pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "retired" {
			t.Fatalf("策略 #%d 应退 st=%q err=%v", sid, st, err)
		}
	}
	var ps string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal
		WHERE inst_id=$1 AND state='fin_armed'`, okxInst).Scan(&ps); err == nil {
		t.Fatalf("armed 提案应退（仍 fin_armed）")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id=$1 AND state='fin_retired'`, okxInst).Scan(&n); err != nil || n == 0 {
		t.Fatalf("提案应 fin_retired n=%d err=%v", n, err)
	}
}

// 锚④：mt5 腿终局拒（SYMBOL_NOT_FOUND）→cfd canonical 入册 fin_venue_
// untradeable+双腿退役+armed 提案退役三件（trigger 兜同 inst 余量）。
func TestXvenueMt5TerminalUntradeable(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	mainID, mateID := mkXvenuePair(t, ctx, pool, "BTC-USDT-SWAP", "BTCUSDm")
	xerr := &mtexec.APIError{Code: "SYMBOL_NOT_FOUND", Msg: "no such symbol", TerminalInst: true}
	if !mtexec.TerminalInstErr(xerr) {
		t.Fatal("TerminalInst 位判失")
	}
	if err := h.xvenueUntradeable(ctx, xvSpec(mainID, "BTC-USDT-SWAP"), mateID, "BTCUSDm", xerr); err != nil {
		t.Fatal("双腿处置败:", err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM fin_venue_untradeable
		WHERE inst_id='BTCUSDm'`).Scan(&code); err != nil || code != "SYMBOL_NOT_FOUND" {
		t.Fatalf("cfd canonical 应入册 code=%q err=%v", code, err)
	}
	xvAssertRetired(t, ctx, pool, mainID, mateID, "BTC-USDT-SWAP")
}

// 锚⑤a：okx 腿终局拒（51001）→okx inst 入册+双腿退役（终局码双源分流——
// TerminalInstErr 各自域判，入册 inst 按腿 venue 对位非单键抄送）。
func TestXvenueOkxTerminalUntradeable(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	mainID, mateID := mkXvenuePair(t, ctx, pool, "BTC-USDT-SWAP", "BTCUSDm")
	xerr := &okxexec.APIError{Code: "51001", Msg: "Instrument ID doesn't exist"}
	if !okxexec.TerminalInstErr(xerr) {
		t.Fatal("okx 51001 终局判失")
	}
	if err := h.xvenueUntradeable(ctx, xvSpec(mainID, "BTC-USDT-SWAP"), mateID, "BTC-USDT-SWAP", xerr); err != nil {
		t.Fatal("双腿处置败:", err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM fin_venue_untradeable
		WHERE inst_id='BTC-USDT-SWAP'`).Scan(&code); err != nil || code != "51001" {
		t.Fatalf("okx inst 应入册 code=%q err=%v", code, err)
	}
	xvAssertRetired(t, ctx, pool, mainID, mateID, "BTC-USDT-SWAP")
}

// 锚⑤b：终局位分流——mtexec.APIError 非终局（MARKET_CLOSED 市闭瞬态）/
// 非 APIError 传输错=不入册不误退（瞬态误杀防线——venue 级终局位唯一锚）。
func TestXvenueNonTerminalNoRetire(t *testing.T) {
	for _, err := range []error{
		&mtexec.APIError{Code: "MARKET_CLOSED", Msg: "closed"},
		errors.New("transport down"),
		nil,
	} {
		if mtexec.TerminalInstErr(err) {
			t.Fatalf("%v 误终局——瞬态将入册误杀", err)
		}
	}
}

// xvSessFake xvenueSess 桩（返修②窄面注假——acct 定值+MT5 可编程，免真
// OpenSession；dead 形态 mt5Cli=nil 供 nil-conn 守断言）。
type xvSessFake struct {
	mt5Cli mt5pb.MT5Client
	acct   string
}

func (f *xvSessFake) account(context.Context) (string, error)    { return f.acct, nil }
func (f *xvSessFake) Authed(ctx context.Context) context.Context { return ctx }
func (f *xvSessFake) Conn() *grpc.ClientConn                     { return nil }
func (f *xvSessFake) MT5() mt5pb.MT5Client                       { return f.mt5Cli }
func (f *xvSessFake) Suspect()                                   {}

// xvMT5Fake MT5 活读桩（SymbolParams 回报 FullAccess——其余面嵌入零值）。
type xvMT5Fake struct{ mt5pb.MT5Client }

func (xvMT5Fake) SymbolParams(context.Context, *mt5pb.SymbolParamsRequest,
	...grpc.CallOption) (*mt5pb.SymbolParamsReply, error) {
	return &mt5pb.SymbolParamsReply{Result: &mt5pb.SymbolParams{
		SymbolGroup: &mt5pb.SymGroup{TradeMode: mt5pb.TradeMode_TradeMode_FullAccess}}}, nil
}

// xvDialFake 方言译件桩（BTCUSDm 恒译——升件面空守）。
type xvDialFake struct{}

func (xvDialFake) VenueSymbol(context.Context, string) (string, error) { return "BTCUSDm", nil }
func (xvDialFake) Escalate(context.Context, string, string)            {}

// xvExecFake execution.Executor 桩（调用序记录+可编程拒——双腿序/回滚携票断言）。
type xvExecFake struct {
	seq    *[]string
	name   string
	reject error
	ticket int64
}

func (f *xvExecFake) Place(_ context.Context, leg execution.Leg, clOrdID string) (execution.OrderResult, error) {
	tag := f.name + ":place"
	if leg.Close {
		tag = fmt.Sprintf("%s:close:t%d", f.name, leg.Ticket)
	}
	*f.seq = append(*f.seq, tag)
	if f.reject != nil {
		return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, f.reject
	}
	return execution.OrderResult{ClOrdID: clOrdID, State: "filled",
		FillPx: 100, FillSz: 1, FillTicket: f.ticket}, nil
}
func (f *xvExecFake) Resolve(_ context.Context, _ execution.Leg, clOrdID string) (execution.OrderResult, error) {
	return execution.OrderResult{ClOrdID: clOrdID, State: "absent"}, nil
}
func (f *xvExecFake) Cancel(context.Context, execution.Leg, string) error { return nil }

// 返修①端到端锚（桩测必修）：xvenueOpen→Execute 全真径——fake okx 执事 leg2
// 回报终局 APIError（51001）→err 链 %w 透传→xvenueErr 终局分类→
// fin_venue_untradeable(okx inst) 落行+双腿退役+提案退役三件全落。旧
// TestXvenueOkxTerminalUntradeable 只直打 xvenueUntradeable handler 未走
// Execute——wiring 断而全绿，本锚堵「handler 过线径断」测试缺口。
func TestXvenueOpenE2EOkxTerminal(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	for k, v := range map[string]string{
		"exec.xvenue_open":             "true",
		"exec.probe_notional_usd":      "50",
		"exec.reopen_cooldown_sec":     "0",
		"exec.hedge_band_bps":          "20",
		"invest.lifecycle.live_switch": "true",
		"risk.max_notional_usd":        "5000",
		"risk.max_venue_pct":           "0.9",
		"risk.max_exposure_usd":        "100000",
		"risk.token_ttl_seconds":       "60",
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value)
			VALUES($1,to_jsonb($2::text))
			ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, k, v); err != nil {
			t.Fatal("闸键夹具:", err)
		}
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	mainID, mateID := mkXvenuePair(t, ctx, pool, "BTC-USDT-SWAP", "BTCUSDm")
	if _, err := pool.Exec(ctx, `INSERT INTO fin_symbol_dialect
		(venue,account,canonical,venue_symbol,ct_val,tick_sz,min_sz,lot_unit,lot_step,digits,trade_mode,spec_hash)
		VALUES('mt5','123','BTCUSDm','BTCUSDm',1,0.5,0.01,'lot',0.01,2,4,'x')`); err != nil {
		t.Fatal("方言行夹具:", err)
	}
	for _, inst := range []string{"BTC-USDT", "BTC-USDT-SWAP"} {
		if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(venue,inst_id,last,ts)
			VALUES('okx',$1,100,now())
			ON CONFLICT (venue,inst_id) DO UPDATE SET last=EXCLUDED.last,ts=EXCLUDED.ts`, inst); err != nil {
			t.Fatal("价夹具:", err)
		}
	}
	var seq []string
	r := execution.NewRunner(pool, cfg, execution.RouterExecutor{
		"mt5": &xvExecFake{seq: &seq, name: "mt5", ticket: 777},
		"okx": &xvExecFake{seq: &seq, name: "okx",
			reject: &okxexec.APIError{Code: "51001", Msg: "Instrument ID doesn't exist"}},
	}, findom.Live)
	gate := risk.NewGate(pool, cfg).WithEquity(func(context.Context, string) (float64, error) {
		return 1e6, nil
	})
	h := hedgeOpener{pool: pool, cfg: cfg, gate: gate, r: r,
		xcli:  &xvSessFake{mt5Cli: xvMT5Fake{}, acct: "123"},
		xdial: xvDialFake{}, xds: mt5.NewDialectStore(pool),
		xmeta: func(context.Context, string) (execution.InstMeta, error) {
			return execution.InstMeta{CtVal: 0.01, CtValCcy: "BTC", LotSz: 1, MinSz: 0.1}, nil
		}}
	_, err := h.xvenueOpen(ctx, xvSpec(mainID, "BTC-USDT-SWAP"))
	if err == nil || !strings.Contains(err.Error(), "venue_untradeable") {
		t.Fatalf("leg2 终局拒应 venue_untradeable 收场: %v", err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM fin_venue_untradeable
		WHERE inst_id='BTC-USDT-SWAP'`).Scan(&code); err != nil || code != "51001" {
		t.Fatalf("okx inst 应入册 code=%q err=%v", code, err)
	}
	xvAssertRetired(t, ctx, pool, mainID, mateID, "BTC-USDT-SWAP")
	// LEGGED 回滚在案——mt5 腿 OrderClose(ticket=777) 执行序留痕。
	var rbSeen bool
	for _, s := range seq {
		if s == "mt5:close:t777" {
			rbSeen = true
		}
	}
	if !rbSeen {
		t.Fatalf("回滚腿未携票 OrderClose: %v", seq)
	}
}

// 返修②锚：guard 现读面 nil-conn 守——Mt5Sizer.ContractPos 会话未建（MT5=nil）
// 返 err 非 panic（OpenedOrders 实读径=guard 拍面，炸进程=崩循环违例）。
func TestMt5SizerDeadSessErrNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ContractPos nil-conn panic（应 err 上浮）: %v", r)
		}
	}()
	s := Mt5Sizer{C: &xvSessFake{}, X: xvDialFake{}}
	if _, err := s.ContractPos(context.Background(), "BTCUSDm"); err == nil {
		t.Fatal("会话未建应 err 非 panic")
	}
}

// 锚⑤c：xvenuePair 提案双行对位——主行 spec→cfd 官名+副腿 id 同 proposal_ref
// 归并；pair 行缺=bridge.xvenue.map 兜底径（映射双无=fail-closed）。
func TestXvenuePairLookup(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	mainID, mateID := mkXvenuePair(t, ctx, pool, "BTC-USDT-SWAP", "BTCUSDm")
	cfd, mate, err := h.xvenuePair(ctx, xvSpec(mainID, "BTC-USDT-SWAP"))
	if err != nil {
		t.Fatal("pair 对位:", err)
	}
	if cfd != "BTCUSDm" || mate != mateID {
		t.Fatalf("pair 归并错: cfd=%s mate=%d（应 %d）", cfd, mate, mateID)
	}
	// 孤行（无副腿提案）+映射缺=fail-closed 不构单腿
	var solo int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('ETH-USDT-SWAP','xvenue-carry',0,'long_perp_short_cfd',50,'live','okx','{}') RETURNING id`).Scan(&solo); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.xvenuePair(ctx, xvSpec(solo, "ETH-USDT-SWAP")); err == nil {
		t.Fatal("双腿映射双无未拒——单腿险敞开")
	}
}
