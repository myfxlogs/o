// noveltyEscCheck 新奇度探：24h 内新假想卡(proposed)→升件(一卡一
// scope,fp=卡id);已裁卡(adopted/dismissed)/老卡(>24h)/销后同卡
// 静默;卡判撤后探针域无死态可复绿(事件类不挂死沿)。
package main

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestNoveltyEscCheck(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, q := range []string{
		`DELETE FROM ops_event WHERE esc_class='novelty'`,
		`DELETE FROM fin_opp_hypothesis`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	svc := coreService{pool: pool}
	deadSince := map[string]time.Time{}
	fireN := func() int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE esc_class='novelty' AND event='升件'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	seed := func(venue, surface, family, state string, ageH int) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_hypothesis
			(venue,surface,family,state,ts) VALUES($1,$2,$3,$4,
			now()-make_interval(hours=>$5)) RETURNING id`,
			venue, surface, family, state, ageH).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	svc.noveltyEscCheck(ctx, deadSince) // 零卡静默
	if n := fireN(); n != 0 {
		t.Fatalf("零卡误唤: n=%d", n)
	}

	seed("okx", "perp:vol-sell", "vol-sell", "adopted", 2) // 已裁卡不唤
	svc.noveltyEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("已裁卡误唤: n=%d", n)
	}

	seed("okx", "spot:grid-old", "grid", "proposed", 30) // 老卡(>24h)不唤
	svc.noveltyEscCheck(ctx, deadSince)
	if n := fireN(); n != 0 {
		t.Fatalf("老卡误唤: n=%d", n)
	}

	idA := seed("okx", "perp:basis-ann", "basis", "proposed", 2) // 新卡唤
	svc.noveltyEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("新卡未唤: n=%d", n)
	}

	// 销后同卡不重火(fp=卡id 销案行同纹抑)——处置行销案再拍
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		SELECT '处置席','处置','升件#'||id||' 卡已裁adopt' FROM ops_event
		WHERE esc_class='novelty' AND event='升件'`); err != nil {
		t.Fatal(err)
	}
	svc.noveltyEscCheck(ctx, deadSince)
	if n := fireN(); n != 1 {
		t.Fatalf("销后同卡重火: n=%d", n)
	}

	idB := seed("mt5", "perp:xau-carry", "swap-carry", "proposed", 1) // 另一新卡再火
	_ = idB
	svc.noveltyEscCheck(ctx, deadSince)
	if n := fireN(); n != 2 {
		t.Fatalf("第二新卡未唤: n=%d", n)
	}
	_ = idA
}
