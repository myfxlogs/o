// 工单#22184 测锚——md 正本闸双面判据钉死：
// SG 镜像面（ops_event 在册）card/decide 写径被拒；
// md 正本面（ops_event 缺席）放行且 hypo 写通——
// 反向依赖锚=同一判据两侧各验一次,不再靠单面存在性
// （fin_capability_surface 两侧同存恒过=退回判词#21346实案）。
package main

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestMdMasterGuard(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// fin 面（ops_event 在册）=镜像面拒写。
	if err := mdMasterErr(ctx, pool); err == nil {
		t.Fatal("fin 面（ops_event 在册）未拒——镜像面缝开")
	}
	// md 面建面:ops_event 缺席+hypo 在册=域特征签。
	if _, err := pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS mdface;
		CREATE TABLE IF NOT EXISTS mdface.fin_opp_hypothesis(
			id bigserial PRIMARY KEY, venue text NOT NULL, surface text NOT NULL,
			family text NOT NULL, leg_primitives jsonb NOT NULL DEFAULT '[]',
			requirements jsonb NOT NULL DEFAULT '{}', state text NOT NULL DEFAULT 'proposed',
			evidence jsonb NOT NULL DEFAULT '{}')`); err != nil {
		t.Fatalf("mdface 建面: %v", err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SET search_path=mdface`); err != nil {
		t.Fatalf("search_path: %v", err)
	}
	// md 面放行。
	if err := mdMasterErr(ctx, conn); err != nil {
		t.Fatalf("md 面（ops_event 缺席）被误拒: %v", err)
	}
	// 放行面写径实通（decide 写域同款 INSERT 面）。
	if _, err := conn.Exec(ctx, `INSERT INTO fin_opp_hypothesis(venue,surface,family)
		VALUES('mdface-venue','mdface-surface','basis')`); err != nil {
		t.Fatalf("md 面 hypo 写败: %v", err)
	}
}
