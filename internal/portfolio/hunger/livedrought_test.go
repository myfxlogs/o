package hunger

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-35729 锚桩（live 终局断流探针——业主令「钱闲着」与管道断同格响）：
// ①live 在册=0 且断流超龄→升件行落,note携归因卡(漏斗/断点/责任归位/在办)
// ②live 在册>0→静默；③断流未超龄→静默；④窗键缺→fail-closed报错；
// ⑤同窗24h去重。

func liveDroughtFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

func seedStrategy(t *testing.T, pool *pgxpool.Pool, state string, ageDays float64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at,created_at,venue,direction)
		 VALUES('BTC-USDT-SWAP','funding-hedge',$1,100,
		   now()-make_interval(days=>$2),now()-make_interval(days=>$2),'okx','short_perp_long_spot')
		 RETURNING id`, state, ageDays).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func seedLiveTransition(t *testing.T, pool *pgxpool.Pool, sid int64, from, to string, ageDays float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason,ts)
		 VALUES($1,$2,$3,'test',now()-make_interval(days=>$4))`, sid, from, to, ageDays); err != nil {
		t.Fatal(err)
	}
}

func liveEscCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class='live_drought'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLiveDroughtFires(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	// 史无 live 迁移+策略宇宙 9d 前诞生——超 3d 阈即火（P0 闲置成立）
	seedStrategy(t, pool, "paper", 9)
	seedEvalVR(t, pool, "AAA-USDT-SWAP", "rejected", "net_apr")
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := liveEscCount(t, pool); n != 1 {
		t.Fatalf("live断流升件未落 n=%d", n)
	}
	var note string
	if err := pool.QueryRow(context.Background(),
		`SELECT note FROM ops_event WHERE event='升件' AND esc_class='live_drought'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"live在册=0", "持续=", "eval7d=", "拒因top=net_apr",
		"断点推定=", "责任=钱径管理面", "在办="} {
		if !strings.Contains(note, want) {
			t.Fatalf("归因卡缺段 %q —— note=%s", want, note)
		}
	}
	// 同窗去重——二拍零新行
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := liveEscCount(t, pool); n != 1 {
		t.Fatalf("同窗复火 n=%d", n)
	}
}

func TestLiveDroughtSilentWhenLive(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	seedStrategy(t, pool, "live", 9) // live 在册>0——终局段活着
	seedStrategy(t, pool, "retired", 9)
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := liveEscCount(t, pool); n != 0 {
		t.Fatalf("live 在册误报 n=%d", n)
	}
}

func TestLiveDroughtSilentWithinAge(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	// 末次 live 降格迁移 1d 前——未超 3d 阈
	sid := seedStrategy(t, pool, "retired", 5)
	seedLiveTransition(t, pool, sid, "live", "retired", 1)
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := liveEscCount(t, pool); n != 0 {
		t.Fatalf("未超龄误报 n=%d", n)
	}
}

func TestLiveDroughtFiresAfterLiveExit(t *testing.T) {
	s, pool := liveDroughtFixture(t)
	// 曾 live 后 9d 前降格——断流龄锚=末次涉 live 迁移
	sid := seedStrategy(t, pool, "retired", 10)
	seedLiveTransition(t, pool, sid, "probe", "live", 20)
	seedLiveTransition(t, pool, sid, "live", "retired", 9)
	if err := s.ProbeLiveDrought(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := liveEscCount(t, pool); n != 1 {
		t.Fatalf("live 出籍断流升件未落 n=%d", n)
	}
}

func TestLiveDroughtPointOrder(t *testing.T) {
	cases := []struct {
		liveSw                 bool
		prop, rej, paper, prob int
		want                   string
	}{
		{false, 5, 10, 3, 1, "委员会闸(live_switch=off)"},
		{true, 0, 10, 3, 1, "评估层(提案0/全拒)"},
		{true, 0, 0, 0, 0, "评估层(7d零提案)"},
		{true, 5, 3, 0, 0, "培育层空(无paper/probe)"},
		{true, 5, 3, 4, 0, "paper→probe断(判据未越线/闸拦)"},
		{true, 5, 3, 4, 2, "probe→live断(观察窗/演习闸/净回优)"},
	}
	for i, c := range cases {
		if got := liveDroughtPoint(c.liveSw, c.prop, c.rej, c.paper, c.prob); got != c.want {
			t.Fatalf("case%d: got=%q want=%q", i, got, c.want)
		}
	}
}

func TestLiveDroughtKeyMissing(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='dm.hunger.live_drought_days'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	s := New(pool, cfg)
	if err := s.ProbeLiveDrought(ctx); err == nil {
		t.Fatal("窗键缺未 fail-closed")
	}
}
