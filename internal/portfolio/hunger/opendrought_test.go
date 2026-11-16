package hunger

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// D案锚桩（业主令直修——开仓断流探针自带归因卡）：
// ①断流超阈→升件行落,note携漏斗卡(鲜票率/eval谱/拒因/池计数/断点推定)
// ②断流未超→零行；③同窗24h去重；④窗键缺→fail-closed报错。

func droughtFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return New(pool, cfg), pool
}

func seedPos(t *testing.T, pool *pgxpool.Pool, ageDays float64) {
	t.Helper()
	var eid int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state,created_at)
		 VALUES(1,'BTC-USDT-SWAP',100,'open',now()) RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_position(execution_id,inst_id,notional_usd,opened_at,closed_at)
		 VALUES($1,'BTC-USDT-SWAP',100,now()-make_interval(days=>$2),now())`, eid, ageDays); err != nil {
		t.Fatal(err)
	}
}

func seedEvalVR(t *testing.T, pool *pgxpool.Pool, inst, verdict, reason string) {
	t.Helper()
	ev := `{"reason":"` + reason + `"}`
	var apr any
	if verdict == "proposed" {
		ev = `{}`
		apr = 0.1 // fin_opp_eval_proposed_net_apr: proposed 行 net_apr 非空
	}
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,net_apr,legs,evidence)
		 VALUES($1,'funding-hedge','short_perp_long_spot',$2,$3,'[]',$4::jsonb)`,
		inst, verdict, apr, ev); err != nil {
		t.Fatal(err)
	}
}

func escCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class='产线断流'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestOpenDroughtFires(t *testing.T) {
	s, pool := droughtFixture(t)
	seedPos(t, pool, 9) // 开仓末行 9d 前——超 3d 阈
	seedEvalVR(t, pool, "AAA-USDT-SWAP", "rejected", "insufficient_balance")
	seedEvalVR(t, pool, "BBB-USDT-SWAP", "proposed", "")
	if err := s.ProbeOpenDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool); n != 1 {
		t.Fatalf("断流升件未落 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND esc_class='产线断流'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"开仓零产", "鲜票=", "eval7d=", "拒因top=insufficient_balance", "断点推定="} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因卡缺段 %q —— note=%s", want, note)
		}
	}
	// 同窗去重——二拍零新行
	if err := s.ProbeOpenDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool); n != 1 {
		t.Fatalf("同窗复火 n=%d", n)
	}
}

func TestOpenDroughtNoRejectRows(t *testing.T) {
	s, pool := droughtFixture(t)
	seedPos(t, pool, 9) // 断流超阈但 eval 谱全空——拒因谱须落 (无) 非炸卡
	if err := s.ProbeOpenDrought(context.Background()); err != nil {
		t.Fatalf("零拒件断流卡读败: %v", err)
	}
	if n := escCount(t, pool); n != 1 {
		t.Fatalf("零拒件断流升件未落 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND esc_class='产线断流'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "拒因top=(无)") {
		t.Fatalf("零拒件拒因段非(无) —— note=%s", note)
	}
}

func TestOpenDroughtSilent(t *testing.T) {
	s, pool := droughtFixture(t)
	seedPos(t, pool, 0.5) // 12h 前刚开——未断流
	if err := s.ProbeOpenDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := escCount(t, pool); n != 0 {
		t.Fatalf("未断流误报 n=%d", n)
	}
}

func TestDroughtPointOrder(t *testing.T) {
	cases := []struct {
		fresh, total, prop, rej, openNow int
		reasons                          string
		want                             string
	}{
		{24, 511, 5, 90, 0, "net_apr×90", "上游层(鲜票率<15%)"},
		{400, 500, 5, 40, 0, "insufficient_balance×40", "本金层"},
		{400, 500, 0, 40, 1, "net_apr×40", "评估层(提案0/全拒)"},
		{400, 500, 3, 10, 0, "dist_thin×10", "闸门层"},
		{400, 500, 3, 10, 2, "dist_thin×10", "复合/未明"},
		// 混合拒因序：insufficient_balance 非首因=非本金主导——HasPrefix 语义钉死
		{400, 500, 3, 40, 0, "dist_thin×30,insufficient_balance×10", "闸门层"},
	}
	for i, c := range cases {
		if got := droughtPoint(c.fresh, c.total, c.prop, c.rej, c.reasons, c.openNow); got != c.want {
			t.Fatalf("case%d: got=%q want=%q", i, got, c.want)
		}
	}
}

func TestOpenDroughtKeyMissing(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='dm.hunger.open_drought_days'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	s := New(pool, cfg)
	if err := s.ProbeOpenDrought(ctx); err == nil {
		t.Fatal("窗键缺未 fail-closed")
	}
	_ = time.Now
}
