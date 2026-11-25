package ledger

// rebase_test.go —— O-33206 业主令重锚锚（spec 验收② go test 面）：
//   - 逐户锚行：delta=eq−net 落 recon_adjust(account_id 直发,
//     source_table='rebase_owner_decree'+source_ref=令簿记行)，
//     锚后 net==eq=残差归零（共存核验——recon_adjust 与周期 recon
//     不双计：recon 净额口径本就含 recon_adjust kind）；
//   - 幂等：armed→done 单次性+桶级已有重锚行跳（半截续跑不双补）；
//   - 残差退役：静态 USD 行 expires 归零+reason 追加，resolver 跟踪型
//     与非 USD 行保留；
//   - fail-safe：venue 读败/未 armed=零写零旗变（盲锚禁）。

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// rebaseVen 六账户 venue 实读桩（okx 四子户+mt5 两户全 present——
// main 型账户须 ven 键在场才入对拍集）。
func rebaseVen() map[[2]string]float64 {
	return map[[2]string]float64{
		{"okx:trading", "USDT"}:       100,
		{"okx:funding", "USDT"}:       10,
		{"okx:earn", "USDT"}:          60,
		{"okx:dcd", "USDT"}:           50,
		{"exness:3001157", "USD"}:     1000,
		{"xmglobal:110660304", "USD"}: 1.8,
	}
}

func rebaseArm(t *testing.T, pool *pgxpool.Pool, val string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO app_config(key,value,updated_at,updated_by)
			VALUES('ledger.rebase_armed',$1::jsonb,now(),'test')
			ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, `"`+val+`"`); err != nil {
		t.Fatal(err)
	}
}

func rebaseFlag(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var v string
	if err := pool.QueryRow(context.Background(),
		`SELECT value::text FROM app_config WHERE key='ledger.rebase_armed'`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// rebaseFixture 现金流底仓：trading 存 5000/earn 13/funding 提 20
// （dcd/mt5 零基线——delta=eq 全额锚）。
func rebaseFixture(t *testing.T, ctx context.Context, r *Runner) {
	t.Helper()
	for _, e := range []Entry{
		{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 5000, AccountID: "okx:trading", SourceRef: 1, SourceTable: "t"},
		{Kind: "deposit", Venue: "okx", Ccy: "USDT", Amount: 13, AccountID: "okx:earn", SourceRef: 1, SourceTable: "t"},
		{Kind: "withdraw", Venue: "okx", Ccy: "USDT", Amount: -20, AccountID: "okx:funding", SourceRef: 1, SourceTable: "t"},
	} {
		if _, err := Post(ctx, r.pool, e); err != nil {
			t.Fatal(err)
		}
	}
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, VenueParts, error) {
		return rebaseVen(), nil, nil
	})
}

func rebaseRows(t *testing.T, pool *pgxpool.Pool) []struct {
	acct, ccy string
	amt       float64
} {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT account_id,ccy,amount FROM fin_cashflow
			WHERE source_table='rebase_owner_decree' ORDER BY account_id,ccy`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []struct {
		acct, ccy string
		amt       float64
	}
	for rows.Next() {
		var a struct {
			acct, ccy string
			amt       float64
		}
		if err := rows.Scan(&a.acct, &a.ccy, &a.amt); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	return out
}

// 锚①：六账户逐键 delta=eq−net 落行+net 归零+done 落旗+二次调用零新增。
func TestRebaseDecreeAnchorsAllAccounts(t *testing.T) {
	ctx, r := setup(t)
	rebaseArm(t, r.pool, "armed")
	rebaseFixture(t, ctx, r)
	r.rebaseDecree(ctx)

	got := rebaseRows(t, r.pool)
	if len(got) != 6 {
		t.Fatalf("重锚行应=6账户键 got=%d %+v", len(got), got)
	}
	// 净额=实读核验（共存判据——recon_adjust 入 net 口径,锚后差=0）。
	for k, eq := range rebaseVen() {
		var net float64
		if err := r.pool.QueryRow(ctx,
			`SELECT COALESCE(sum(amount),0) FROM fin_cashflow
				WHERE account_id=$1 AND ccy=$2 AND kind NOT IN ('earn_in','earn_out')`,
			k[0], k[1]).Scan(&net); err != nil {
			t.Fatal(err)
		}
		if d := net - eq; d > 1e-6 || d < -1e-6 {
			t.Fatalf("%s/%s 锚后 net=%.6f 应=eq=%.6f", k[0], k[1], net, eq)
		}
	}
	// 代表锚值核对：trading 5000→eq100=delta−4900;dcd 0→50=+50;
	// exness 0→1000=+1000;xmglobal 0→1.8=+1.8。
	want := map[string]float64{
		"okx:trading|USDT": -4900, "okx:dcd|USDT": 50,
		"exness:3001157|USD": 1000, "xmglobal:110660304|USD": 1.8,
	}
	for _, g := range got {
		k := g.acct + "|" + g.ccy
		if w, ok := want[k]; ok && (g.amt-w > 1e-6 || g.amt-w < -1e-6) {
			t.Fatalf("%s delta=%.4f 应=%.4f", k, g.amt, w)
		}
	}
	if rebaseFlag(t, r.pool) != `"done"` {
		t.Fatalf("armed 应置 done got=%s", rebaseFlag(t, r.pool))
	}
	// 幂等：二次调用零新增（done 闸）+令簿记行=2。
	r.rebaseDecree(ctx)
	if n := len(rebaseRows(t, r.pool)); n != 6 {
		t.Fatalf("二次调用不应新增锚行 got=%d", n)
	}
	var ev int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE ticket='O-33206' AND event='簿记'`).Scan(&ev); err != nil {
		t.Fatal(err)
	}
	if ev != 2 {
		t.Fatalf("令+结果簿记行应=2 got=%d", ev)
	}
}

// 锚②：桶级幂等——已有重锚行的键跳过,其余照锚（半截续跑语义）。
func TestRebaseDecreeSkipsAnchoredBucket(t *testing.T) {
	ctx, r := setup(t)
	rebaseArm(t, r.pool, "armed")
	rebaseFixture(t, ctx, r)
	// 预置 trading 桶重锚行（模拟半截续跑已锚）——amount 任意,存在即跳。
	if _, err := Post(ctx, r.pool, Entry{Kind: "recon_adjust", Venue: "okx",
		Ccy: "USDT", Amount: -1, AccountID: "okx:trading",
		SourceRef: 1, SourceTable: rebaseSourceTable}); err != nil {
		t.Fatal(err)
	}
	// trading 净额现=5000-1=4999 vs eq=100——若误重锚会再写行;
	// 幂等跳后该键差留存=探针域真差（续跑语义=不覆盖已有锚）。
	r.rebaseDecree(ctx)
	var n int
	if err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_cashflow WHERE source_table='rebase_owner_decree'
			AND account_id='okx:trading' AND ccy='USDT'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("trading 桶重锚行应=1(预置) got=%d", n)
	}
	if got := len(rebaseRows(t, r.pool)); got != 6 {
		t.Fatalf("总锚行=预置1+新锚5=6 got=%d", got)
	}
}

// 锚③：静态 USD 残差退役+resolver/非USD 保留。
func TestRebaseDecreeResidualRetire(t *testing.T) {
	ctx, r := setup(t)
	rebaseArm(t, r.pool, "armed")
	rebaseFixture(t, ctx, r)
	seedResid(t, r.pool, "okx", "USDT", 1951.22, "", "okx:trading")       // 静态→退役
	seedResid(t, r.pool, "okx", "USDT", 55, "dcd_mtm_drift:x", "okx:dcd") // resolver→留
	seedResid(t, r.pool, "okx", "BTC", 5, "", "okx:dcd")                  // 非USD→留
	var staticID, resID, btcID int64
	r.pool.QueryRow(ctx, `SELECT id FROM fin_known_residual WHERE account_id='okx:trading' AND amount_usd=1951.22`).Scan(&staticID)
	r.pool.QueryRow(ctx, `SELECT id FROM fin_known_residual WHERE account_id='okx:dcd' AND resolver LIKE 'dcd_mtm%'`).Scan(&resID)
	r.pool.QueryRow(ctx, `SELECT id FROM fin_known_residual WHERE account_id='okx:dcd' AND ccy='BTC'`).Scan(&btcID)

	r.rebaseDecree(ctx)

	var exp bool
	var reason string
	if err := r.pool.QueryRow(ctx,
		`SELECT expires_at<=now(), reason FROM fin_known_residual WHERE id=$1`, staticID).Scan(&exp, &reason); err != nil {
		t.Fatal(err)
	}
	if !exp || len(reason) < 10 {
		t.Fatalf("静态USD残差应退役+reason追加 got exp=%v", exp)
	}
	for _, id := range []int64{resID, btcID} {
		if err := r.pool.QueryRow(ctx,
			`SELECT expires_at>now() FROM fin_known_residual WHERE id=$1`, id).Scan(&exp); err != nil {
			t.Fatal(err)
		}
		if !exp {
			t.Fatalf("残差#%d 应保留(unexpired) got expired", id)
		}
	}
}

// 锚④：fail-safe——venue 读败=零写零旗变（盲锚禁,armed 保留下拍重试）。
func TestRebaseDecreeVenueReadFail(t *testing.T) {
	ctx, r := setup(t)
	rebaseArm(t, r.pool, "armed")
	rebaseFixture(t, ctx, r)
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, VenueParts, error) {
		return nil, nil, fmt.Errorf("venue 读败桩")
	})
	r.rebaseDecree(ctx)
	if got := len(rebaseRows(t, r.pool)); got != 0 {
		t.Fatalf("读败不应落锚行 got=%d", got)
	}
	if rebaseFlag(t, r.pool) != `"armed"` {
		t.Fatalf("读败 armed 应保留 got=%s", rebaseFlag(t, r.pool))
	}
}

// 锚⑤：未 armed/done=零成本直返（对账拍不背重锚税）。
func TestRebaseDecreeNotArmed(t *testing.T) {
	ctx, r := setup(t)
	rebaseArm(t, r.pool, "done")
	rebaseFixture(t, ctx, r)
	called := false
	r.WithBalances(func(ctx context.Context) (map[[2]string]float64, VenueParts, error) {
		called = true
		return rebaseVen(), nil, nil
	})
	r.rebaseDecree(ctx)
	if called {
		t.Fatal("未 armed 不应触 venue 现读")
	}
	if got := len(rebaseRows(t, r.pool)); got != 0 {
		t.Fatalf("done 态不应落锚行 got=%d", got)
	}
}
