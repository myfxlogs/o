// evalgate_test.go —— 指纹闸桩测（工单#22904 验收锚）：同指纹零重复行
// （跳 INSERT 复用行 id）+verdict/net_apr 变迁仍留版本史+NULL 族同闸。
package evalgate

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatalf("md migrate: %v", err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("fin migrate: %v", err)
	}
	return ctx, pool
}

func insertEval(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inst, tmpl, dir, verdict string, net any) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES($1,$2,$3,$4,$5,'[]','{}') RETURNING id`,
		inst, tmpl, dir, net, verdict).Scan(&id)
	if err != nil {
		t.Fatalf("insert eval: %v", err)
	}
	return id
}

func TestSameFingerprint(t *testing.T) {
	ctx, pool := setup(t)
	const inst = "TESTGATE-USDT-SWAP"
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE inst_id=$1", inst)
	}()

	// 无历史行→(0,false) 走 INSERT 径。
	id, same, err := SameFingerprint(ctx, pool, inst, "funding-hedge", "short_perp", "rejected", 0.05)
	if err != nil || same || id != 0 {
		t.Fatalf("无历史行: id=%d same=%v err=%v", id, same, err)
	}

	// 首版落账。
	first := insertEval(t, ctx, pool, inst, "funding-hedge", "short_perp", "rejected", 0.05)

	// 同指纹（verdict+net_apr 双同）→(行id,true) 跳 INSERT 复用。
	id, same, err = SameFingerprint(ctx, pool, inst, "funding-hedge", "short_perp", "rejected", 0.05)
	if err != nil || !same || id != first {
		t.Fatalf("同指纹: id=%d same=%v err=%v（want id=%d same=true）", id, same, err, first)
	}

	// net_apr 漂移→指纹异→(0,false) 新版史。
	id, same, err = SameFingerprint(ctx, pool, inst, "funding-hedge", "short_perp", "rejected", 0.051)
	if err != nil || same || id != 0 {
		t.Fatalf("net漂移: id=%d same=%v err=%v", id, same, err)
	}
	second := insertEval(t, ctx, pool, inst, "funding-hedge", "short_perp", "rejected", 0.051)

	// verdict 变迁→指纹异（最新行=second）。
	id, same, err = SameFingerprint(ctx, pool, inst, "funding-hedge", "short_perp", "proposed", 0.051)
	if err != nil || same || id != 0 {
		t.Fatalf("verdict变迁: id=%d same=%v err=%v", id, same, err)
	}
	third := insertEval(t, ctx, pool, inst, "funding-hedge", "short_perp", "proposed", 0.051)
	if third <= second {
		t.Fatalf("版本史 id 序破: %d <= %d", third, second)
	}

	// 指纹回到 proposed+0.051（=最新行）→同指纹复用 third。
	id, same, err = SameFingerprint(ctx, pool, inst, "funding-hedge", "short_perp", "proposed", 0.051)
	if err != nil || !same || id != third {
		t.Fatalf("回指纹: id=%d same=%v err=%v（want id=%d）", id, same, err, third)
	}

	// 键三分量隔离：同 inst 异 template/direction 不互染。
	if _, same, _ = SameFingerprint(ctx, pool, inst, "dir-mom", "short_perp", "rejected", 0.05); same {
		t.Fatal("异 template 误同指纹")
	}
	if _, same, _ = SameFingerprint(ctx, pool, inst, "funding-hedge", "long_perp", "rejected", 0.05); same {
		t.Fatal("异 direction 误同指纹")
	}
}

func TestSameFingerprintNullNet(t *testing.T) {
	ctx, pool := setup(t)
	const inst = "TESTGATE-NULL"
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE inst_id=$1", inst)
	}()

	// NULL net_apr 族（gapscan 面）：NULL↔NULL 同指纹、非 NULL 异指纹。
	first := insertEval(t, ctx, pool, inst, "surface-gap", "gap_hedge", "unevaluable", nil)
	id, same, err := SameFingerprint(ctx, pool, inst, "surface-gap", "gap_hedge", "unevaluable", nil)
	if err != nil || !same || id != first {
		t.Fatalf("NULL同指纹: id=%d same=%v err=%v", id, same, err)
	}
	if _, same, _ = SameFingerprint(ctx, pool, inst, "surface-gap", "gap_hedge", "unevaluable", 0.1); same {
		t.Fatal("NULL↔非NULL 误同指纹")
	}
}
