// terminal_guard_test.go —— O-38820 锚：trans 状态写径终态守卫——
// 现态终态（fin_venue_terminal_state 册内）的 venue 拒被迁移覆写
// （state 保持原终态；evidence 合并同 UPDATE 被守卫连带拦——终态
// 行整体冻结=禁误杀语义）；非终态照常迁移。
package onboard

import (
	"context"
	"testing"
)

func TestTransTerminalStateGuarded(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()

	// 终态 venue：trans 试图迁 'probed'——state 不得被覆写
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-dead','cex-perp','rejected',true)`); err != nil {
		t.Fatal(err)
	}
	if err := r.trans(ctx, "v-dead", "0", "probed", "探测", map[string]any{}, "n", false); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-dead'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "rejected" {
		t.Fatalf("终态 venue 被 trans 覆写 state=%s want rejected", st)
	}

	// 非终态照常：candidate→probed 迁移成功
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-live','cex-perp','candidate',true)`); err != nil {
		t.Fatal(err)
	}
	if err := r.trans(ctx, "v-live", "0", "probed", "探测", map[string]any{}, "n", false); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-live'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "probed" {
		t.Fatalf("非终态 venue 未迁移 state=%s want probed", st)
	}
}
