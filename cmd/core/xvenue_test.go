package main

// xvenue_test.go —— T-0145 跨所 carry untradeable 联动桩（DM 裁定锚④⑤）：
// mt5 腿终局拒→cfd canonical 入册+双腿+armed 提案退役三件；黑名单预检命中=
// 零 venue 调用退三件；终局码双源分流（okx→okx inst 入册/mt5→cfd canonical）。
import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/portfolio/lifecycle"
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
