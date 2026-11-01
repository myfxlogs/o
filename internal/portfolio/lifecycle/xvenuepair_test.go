package lifecycle

// xvenuepair_test.go —— 跨所双腿 paper 桩测（T-0144 验收锚）：proposal→主从
// 双腿策略行（venue 各正/proposal_ref 同引/CFD 反查）+swap 快照锁存；
// armed→open 对 xvenue-carry 显式禁。
import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func xsetup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return context.Background(), NewRunner(pool, cfg)
}

// rejCount 拒因计数读（Door 内存快照——显式禁断言面）。
func rejCount(t *testing.T, r *Runner, key string) int {
	t.Helper()
	return int(r.rej.Snapshot()["lifecycle/"+key])
}

// 锚①：双腿开户——主行 okx perp+副行 mt5 CFD（bridge.xvenue.map 反查），
// proposal_ref 同引+armed 翻面+swap 快照 fill 锁存。
func TestXvenuePairAdmit(t *testing.T) {
	ctx, r := xsetup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('bridge.xvenue.map',
		'{"ETH-USDT-SWAP":"ETHUSDm"}') ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	// mt5_swap 信号快照源。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('mt5_swap','ETHUSDm',-817,'{"swap_short":-1786.5,"swap_type":1,"points":0.01}',now())`); err != nil {
		t.Fatal(err)
	}
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('ETH-USDT-SWAP','xvenue-carry','long_perp_short_cfd',0.08,'proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal("评估夹具（0060 direction 枚举扩应放行）:", err)
	}
	var pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'ETH-USDT-SWAP','long_perp_short_cfd','fin_proposed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal("提案夹具:", err)
	}
	if err := r.admitXvenuePair(ctx, pid, "ETH-USDT-SWAP", "long_perp_short_cfd", 100, 24, 0, 0); err != nil {
		t.Fatal("双腿开户败:", err)
	}
	var mainID, cfdID int64
	var mainVenue, cfdVenue, cfdDir string
	if err := r.pool.QueryRow(ctx, `SELECT id, venue FROM fin_strategy
		WHERE proposal_ref=$1 AND venue='okx'`, pid).Scan(&mainID, &mainVenue); err != nil {
		t.Fatal("主行应落:", err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT id, venue, direction FROM fin_strategy
		WHERE proposal_ref=$1 AND venue='mt5'`, pid).Scan(&cfdID, &cfdVenue, &cfdDir); err != nil {
		t.Fatal("副行应落:", err)
	}
	if cfdDir != "short_cfd" {
		t.Fatalf("副腿向应取反 short_cfd got %q", cfdDir)
	}
	var ps string
	r.pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, pid).Scan(&ps)
	if ps != "fin_armed" {
		t.Fatalf("提案应 armed got %q", ps)
	}
	var snapKind string
	if err := r.pool.QueryRow(ctx, `SELECT kind FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='swap_snapshot'`, cfdID).Scan(&snapKind); err != nil {
		t.Fatal("swap 快照应锁存:", err)
	}
	// 幂等重入：不叠仓。
	if err := r.admitXvenuePair(ctx, pid, "ETH-USDT-SWAP", "long_perp_short_cfd", 100, 24, 0, 0); err != nil {
		t.Fatal(err)
	}
	var n int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE proposal_ref=$1`, pid).Scan(&n)
	if n != 2 {
		t.Fatalf("双腿应恒 2 行 got %d", n)
	}
}

// 锚②：armed→open 对 xvenue-carry 显式拒（openLive 零 Opener 消费——钉死日志径）。
func TestXvenueOpenForbidden(t *testing.T) {
	ctx, r := xsetup(t)
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria,direction,venue)
		VALUES('ETH-USDT-SWAP','xvenue-carry',0,'live',100,'{}','long_perp_short_cfd','okx') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	// stub Opener（openLive 有 op 才进消费循环——禁断言=零调用+拒计数）。
	calls := 0
	r.AttachOpener(stubOpener{fn: func() { calls++ }})
	before := rejCount(t, r, "xvenue_open_forbidden")
	r.openLive(ctx)
	after := rejCount(t, r, "xvenue_open_forbidden")
	if after != before+1 {
		t.Fatalf("xvenue-carry live 行应被显式拒（计数 %d→%d）", before, after)
	}
	if calls != 0 {
		t.Fatalf("禁下应零 Opener 调用 got %d", calls)
	}
	// 无 open 持仓行产出（零真执行）。
	var opens int
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_position WHERE strategy_ref=$1`, sid).Scan(&opens)
	if opens != 0 {
		t.Fatalf("禁下应零开仓 got %d", opens)
	}
}

// stubOpener 开仓缝桩（调用计数——禁断言=零调）。
type stubOpener struct{ fn func() }

func (s stubOpener) Open(ctx context.Context, spec OpenSpec) (int64, error) {
	s.fn()
	return 0, nil
}
