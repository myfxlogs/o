// notifyflood_test.go —— ev_eval 通知洪泛源治桩锚（T-0162）：实质变才发——
// 同键同值覆写零 NOTIFY；批内多变一 NOTIFY（同事务空 payload 同值去重）；
// INSERT 恒发（新键=新评估）；实质变仍即时醒（lifecycle 唤醒语义不变线）。
// 行为测每测独占 channel（触发器重指独有名——共享测试实例上他包 fin_opp_eval
// 写拍同频道噪声隔离）；生产频道形由结构锚直验触发器定义。
package events

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// upsertEval 生产写拍同形（funding 扫描臂 persist 同构——INSERT..ON CONFLICT
// DO UPDATE 无条件覆写含 ts）。
const upsertEval = `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
	VALUES($1,'funding-hedge','short_perp_long_spot',$2,$3,$4,$5::jsonb,$6::jsonb)
	ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
	gross_apr=EXCLUDED.gross_apr,net_apr=EXCLUDED.net_apr,verdict=EXCLUDED.verdict,
	legs=EXCLUDED.legs,evidence=EXCLUDED.evidence`

func evalArgs(verdict string, netApr float64, evidence string) []any {
	return []any{"T0162-USDT-SWAP", nil, netApr, verdict, "[]", evidence}
}

// listener 独占听会话（借 conn LISTEN——还池前 UNLISTEN 清频，T-0131 律）。
type listener struct{ conn *pgxpool.Conn }

func listenChannel(t *testing.T, pool *pgxpool.Pool, ch string) *listener {
	t.Helper()
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(context.Background(), "LISTEN "+ch); err != nil {
		conn.Release()
		t.Fatalf("LISTEN %s: %v", ch, err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), "UNLISTEN *")
		conn.Release()
	})
	return &listener{conn}
}

// expect 窗内恰收 want 条（收满即返；不足=deadline 爆——多到/不到皆 FAIL）。
func (l *listener) expect(t *testing.T, want int, within time.Duration) {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), within)
	defer stop()
	for i := 0; i < want; i++ {
		if _, err := l.conn.Conn().WaitForNotification(ctx); err != nil {
			t.Fatalf("窗 %v 内应恰收 %d 条通知，收第 %d 条败（%v）", within, want, i+1, err)
		}
	}
	// 收满后再等半窗确认无超额（恰 want 条非 ≥want 条）。
	ctx2, stop2 := context.WithTimeout(context.Background(), within/2)
	defer stop2()
	if _, err := l.conn.Conn().WaitForNotification(ctx2); err == nil {
		t.Fatalf("应恰收 %d 条通知，实收超额", want)
	}
}

// expectQuiet 静默判：整窗零通知（同值覆写/DO NOTHING/批全同值=零发射）。
func (l *listener) expectQuiet(t *testing.T, within time.Duration) {
	t.Helper()
	ctx, stop := context.WithTimeout(context.Background(), within)
	defer stop()
	if _, err := l.conn.Conn().WaitForNotification(ctx); err == nil {
		t.Fatalf("整窗应零通知（同值覆写静默——实质变才发被破）")
	}
}

// TestEvalNotifyFloodShrink T-0162 四桩锚（触发器重指独占频道后按生产写拍形逐锚）。
func TestEvalNotifyFloodShrink(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// 结构锚：生产频道形（Migrate 后 fin_opp_eval 触发器=row-level+实质变函数+'ev_eval'）。
	var def string
	if err := pool.QueryRow(ctx, `SELECT pg_get_triggerdef(oid) FROM pg_trigger
		WHERE tgrelid='fin_opp_eval'::regclass AND tgname='ev_eval_trg' AND NOT tgisinternal`).Scan(&def); err != nil {
		t.Fatalf("ev_eval_trg 读形: %v", err)
	}
	for _, frag := range []string{"FOR EACH ROW", "ev_notify_eval('ev_eval')"} {
		if !strings.Contains(def, frag) {
			t.Fatalf("生产触发器形缺 %q: %s", frag, def)
		}
	}

	// 行为测重指独占频道（共享实例跨包噪声隔离——函数同源只换频道实参）。
	ch := fmt.Sprintf("ev_eval_t0162_%d", time.Now().UnixNano())
	if _, err := pool.Exec(ctx, "DROP TRIGGER ev_eval_trg ON fin_opp_eval"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf(`CREATE TRIGGER ev_eval_trg AFTER INSERT OR UPDATE ON fin_opp_eval
		FOR EACH ROW EXECUTE FUNCTION ev_notify_eval('%s')`, ch)); err != nil {
		t.Fatal(err)
	}
	l := listenChannel(t, pool, ch)

	// 锚③ INSERT 恒发：新键=新评估=实质变，即时醒。
	if _, err := pool.Exec(ctx, upsertEval, evalArgs("proposed", 0.10, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚① 同键同值覆写零 NOTIFY：三连写同值（逐行 autocommit=生产洪泛形）整窗静默。
	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx, upsertEval, evalArgs("proposed", 0.10, `{"n":1}`)...); err != nil {
			t.Fatal(err)
		}
	}
	l.expectQuiet(t, 600*time.Millisecond)

	// 锚④ 实质变仍即时醒（lifecycle 唤醒语义不变线）：verdict/评分/evidence 各换面即醒。
	if _, err := pool.Exec(ctx, upsertEval, evalArgs("rejected", 0.10, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, upsertEval, evalArgs("rejected", 0.20, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, upsertEval, evalArgs("rejected", 0.20, `{"n":2}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚② 批内多变一 NOTIFY：一语句三行（一不变+一实质变+一新键）=同事务
	// 三次 pg_notify 同值去重→恰一醒；同语句重放（全行已同值）=零醒。
	batch := `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('T0162-B1','funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}'),
		('T0162-B2','funding-hedge','short_perp_long_spot',NULL,'rejected','[]','{}'),
		('T0162-USDT-SWAP','funding-hedge','short_perp_long_spot',NULL,'rejected','[]','{"n":2}')
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		verdict=EXCLUDED.verdict,legs=EXCLUDED.legs,evidence=EXCLUDED.evidence`
	if _, err := pool.Exec(ctx, batch); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, batch); err != nil {
		t.Fatal(err)
	}
	l.expectQuiet(t, 600*time.Millisecond)

	// 锚②（拍级事务合流形——funding 扫描臂 T-0162 改后同形）：一事务多条
	// 逐行 upsert（两行实质变+一行同值）→整事务恰一醒；重放全同值=零醒。
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, a := range [][]any{
			evalArgs("rejected", 0.20, `{"n":2}`),                // 同值行（静默）
			{"T0162-B1", nil, nil, "rejected", "[]", `{"x":1}`},  // 实质变
			{"T0162-B2", nil, 0.20, "proposed", "[]", `{"x":1}`}, // 实质变（fin_0094 闸：proposed 行 net_apr 非 NULL）
		} {
			if _, err := tx.Exec(ctx, upsertEval, a...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, a := range [][]any{
			evalArgs("rejected", 0.20, `{"n":2}`),
			{"T0162-B1", nil, nil, "rejected", "[]", `{"x":1}`},
			{"T0162-B2", nil, 0.20, "proposed", "[]", `{"x":1}`},
		} {
			if _, err := tx.Exec(ctx, upsertEval, a...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	l.expectQuiet(t, 600*time.Millisecond)

	// 边缘锚：DO NOTHING 臂零行零 fire=静默。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('T0162-USDT-SWAP','funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}')
		ON CONFLICT (inst_id,template,direction) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	l.expectQuiet(t, 600*time.Millisecond)
}
