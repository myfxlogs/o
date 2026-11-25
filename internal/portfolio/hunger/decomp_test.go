package hunger

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-39480 细胞A 桩测（decompWatcher 已部署分解眼）：
// ①bot 面「grid 单调正增∧float 负吞噬」→decomp_pos_fragment 升件；
// ②吞噬比不足/float 正→不升；③同 scope 冷却窗抑复；
// ④paper 面租金正∧价差负（及反向）→strategy:<id> 升件；
// ⑤阈件缺→fail-closed 报错不升件。

func decompFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

func seedBot(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var id int64
	algo := fmt.Sprintf("algo-%d", time.Now().UnixNano())
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO fin_bot_order(venue,inst_id,algo_id,params)
		 VALUES('okx','BTC-USDT-SWAP',$1,'{}') RETURNING id`, algo).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func seedSnap(t *testing.T, pool *pgxpool.Pool, bot int64, ts time.Time, grid, flt float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_bot_equity_snap(bot_id,algo_id,inst_id,ts,investment,
		 grid_profit,float_profit,total_pnl,details)
		 VALUES($1,'algo-t1','BTC-USDT-SWAP',$2,100,$3,$4,$5,'{}')`,
		bot, ts, grid, flt, grid+flt); err != nil {
		t.Fatal(err)
	}
}

func escRows(t *testing.T, pool *pgxpool.Pool, class string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT esc_scope FROM ops_event WHERE event='升件' AND esc_class=$1 ORDER BY id`, class)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestDecompBotFires(t *testing.T) {
	s, pool := decompFixture(t)
	ctx := context.Background()
	bot := seedBot(t, pool)
	now := time.Now()
	// 全期窗：grid 0.1→0.8 单调正增（增量 0.7≥0.5）；float 末值 -2.0<0
	// 且 |−2.0|>0.8×1.5=1.2 → 两窗判据全过。
	seedSnap(t, pool, bot, now.Add(-72*time.Hour), 0.1, -1.5)
	seedSnap(t, pool, bot, now.Add(-48*time.Hour), 0.4, -1.8)
	seedSnap(t, pool, bot, now.Add(-24*time.Hour), 0.6, -1.9)
	seedSnap(t, pool, bot, now.Add(-1*time.Hour), 0.8, -2.0)
	if err := s.ProbeDecompWatch(ctx); err != nil {
		t.Fatal(err)
	}
	got := escRows(t, pool, decompEscClass)
	if len(got) != 1 || got[0] != "bot:"+strconv.FormatInt(bot, 10) {
		t.Fatalf("升件=%v 期 [bot:%d]", got, bot)
	}
	// 冷却窗内重拍=抑复不重复升。
	if err := s.ProbeDecompWatch(ctx); err != nil {
		t.Fatal(err)
	}
	if got := escRows(t, pool, decompEscClass); len(got) != 1 {
		t.Fatalf("冷却抑复败——升件=%v 期 1", got)
	}
}

func TestDecompBotNoFire(t *testing.T) {
	s, pool := decompFixture(t)
	ctx := context.Background()
	bot := seedBot(t, pool)
	now := time.Now()
	// float 负但吞噬比不足（|−0.5|<0.8×1.5）→不升。
	seedSnap(t, pool, bot, now.Add(-48*time.Hour), 0.1, -0.4)
	seedSnap(t, pool, bot, now.Add(-1*time.Hour), 0.8, -0.5)
	// grid 非单调（中途回落）→不升。
	bot2 := seedBot(t, pool)
	seedSnap(t, pool, bot2, now.Add(-48*time.Hour), 0.5, -2.0)
	seedSnap(t, pool, bot2, now.Add(-24*time.Hour), 0.3, -2.0)
	seedSnap(t, pool, bot2, now.Add(-1*time.Hour), 0.8, -2.0)
	if err := s.ProbeDecompWatch(ctx); err != nil {
		t.Fatal(err)
	}
	if got := escRows(t, pool, decompEscClass); len(got) != 0 {
		t.Fatalf("不应升件 got=%v", got)
	}
}

func TestDecompPaperFires(t *testing.T) {
	s, pool := decompFixture(t)
	ctx := context.Background()
	var sid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(template_ref,state,inst_id,notional_usd,entered_at)
		 VALUES('funding-hedge','paper','BTC-USDT-SWAP',1000,now()) RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	// 租金件 funding 正 +1.0 ∧ 价差件 pnl 负 −2.0（|−2|>1×1.5∧1≥0.5）→升。
	for _, kv := range [][2]any{
		{"funding", 1.0}, {"pnl", -1.2}, {"pnl", -0.8},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			 VALUES($1,$2,'BTC-USDT-SWAP',$3,'{}',now()-interval '1 hour')`,
			sid, kv[0], kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ProbeDecompWatch(ctx); err != nil {
		t.Fatal(err)
	}
	got := escRows(t, pool, decompEscClass)
	if len(got) != 1 || got[0] != "strategy:"+strconv.FormatInt(sid, 10) {
		t.Fatalf("paper 升件=%v 期 [strategy:%d]", got, sid)
	}
}

func TestDecompCfgFailClosed(t *testing.T) {
	s, pool := decompFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='explore.decomp_min_gain_usd'`); err != nil {
		t.Fatal(err)
	}
	if err := s.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDecompWatch(ctx); err == nil {
		t.Fatal("阈键缺应 fail-closed 报错")
	}
}
