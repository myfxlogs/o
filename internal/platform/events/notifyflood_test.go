// notifyflood_test.go —— ev_eval 通知洪泛源治桩锚（T-0162）：实质变才发——
// INSERT 恒发（新键/新版本=新评估，版史形下每版皆新信息）；UPDATE 仅
// verdict/评分/legs/evidence IS DISTINCT 才发（ts 元数据不计——SG uEval
// 镜像刷新同值静默仍守）；批内多变一 NOTIFY（同事务空 payload 同值去重）。
// 行为测每测独占 channel（触发器重指独有名——共享测试实例上他包 fin_opp_eval
// 写拍同频道噪声隔离）；生产频道形由结构锚直验触发器定义。
// 工单#19778 版史迁移：md 写径 upsert→纯 INSERT（重述版=新版本仍发——
// 实测写率~1-2/min 远低洪泛阈，版通知=新信念唤醒正当）；UPDATE 径语义不变。
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

// insertEval 生产写拍同形（工单#19778 版史形——md 写径纯 INSERT 无
// ON CONFLICT；每评估一版 id 递增）。
const insertEval = `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
	VALUES($1,'funding-hedge','short_perp_long_spot',$2,$3,$4,$5::jsonb,$6::jsonb)`

// updateEval UPDATE 径同形（SG uEval 镜像刷新臂——同键 ts 新鲜赢覆写
// 实质字段；洪泛闸 IS DISTINCT 判面在此径仍生效）。
const updateEval = `UPDATE fin_opp_eval SET ts=now(),gross_apr=$2,net_apr=$3,
	verdict=$4,legs=$5::jsonb,evidence=$6::jsonb
	WHERE inst_id=$1 AND template='funding-hedge' AND direction='short_perp_long_spot'`

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

	// 锚③ INSERT 恒发：新键/新版本=新评估=实质变，即时醒（版史形）。
	if _, err := pool.Exec(ctx, insertEval, evalArgs("proposed", 0.10, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚①' 同值 UPDATE 零 NOTIFY：三连覆写同值（ts 元数据不计——SG uEval
	// 镜像刷新形）整窗静默；洪泛闸 IS DISTINCT 判面在 UPDATE 径仍守。
	for i := 0; i < 3; i++ {
		if _, err := pool.Exec(ctx, updateEval, evalArgs("proposed", 0.10, `{"n":1}`)...); err != nil {
			t.Fatal(err)
		}
	}
	l.expectQuiet(t, 600*time.Millisecond)

	// 版史语义锚：同值 INSERT 新版本仍发——每版=新信念事件（非覆写噪声，
	// 实测写率~1-2/min 远低洪泛阈）。
	if _, err := pool.Exec(ctx, insertEval, evalArgs("proposed", 0.10, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚④ 实质变 UPDATE 仍即时醒（lifecycle 唤醒语义不变线）：verdict/评分/evidence 各换面即醒。
	if _, err := pool.Exec(ctx, updateEval, evalArgs("rejected", 0.10, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, updateEval, evalArgs("rejected", 0.20, `{"n":1}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, updateEval, evalArgs("rejected", 0.20, `{"n":2}`)...); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚② 批内多变一 NOTIFY：一语句多行版史 INSERT=同事务 pg_notify 同值
	// 去重→恰一醒；重放=新版本再发（版史语义——重述版亦新信息，非
	// T-0162 同值覆写静默形）。
	batch := `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('T0162-B1','funding-hedge','short_perp_long_spot',0.08,'proposed','[]','{}'),
		('T0162-B2','funding-hedge','short_perp_long_spot',NULL,'rejected','[]','{}'),
		('T0162-USDT-SWAP','funding-hedge','short_perp_long_spot',NULL,'rejected','[]','{"n":2}')`
	if _, err := pool.Exec(ctx, batch); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)
	if _, err := pool.Exec(ctx, batch); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 锚②（拍级事务合流形）：一事务多条逐行 INSERT（两新键+一既有键新版）
	// →整事务恰一醒；重放 tx=新版本再发。
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, a := range [][]any{
			evalArgs("rejected", 0.20, `{"n":2}`),
			{"T0162-B1", nil, nil, "rejected", "[]", `{"x":1}`},
			{"T0162-B2", nil, 0.20, "proposed", "[]", `{"x":1}`},
		} {
			if _, err := tx.Exec(ctx, insertEval, a...); err != nil {
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
			if _, err := tx.Exec(ctx, insertEval, a...); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	l.expect(t, 1, 2*time.Second)

	// 边缘锚：UPDATE 零行命中=零 fire=静默。
	if _, err := pool.Exec(ctx, `UPDATE fin_opp_eval SET ts=now()
		WHERE inst_id='T0162-NOPE'`); err != nil {
		t.Fatal(err)
	}
	l.expectQuiet(t, 600*time.Millisecond)
}
