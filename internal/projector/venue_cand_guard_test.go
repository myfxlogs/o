// venue_cand_guard_test.go —— O-38820 锚：桥 upsert 终态守卫——
// ①SG 镜像已终态行不被 md 正本非终态刷回（rejected 保持 rejected）；
// ③非终态行照常全字段投影（state 随 EXCLUDED 迁移）；
// ④终态册加新行=守卫自动覆盖（'paper' 入册即受保护——名单非硬编码）。
package projector

import (
	"context"
	"testing"
)

func TestVenueCandTerminalGuard(t *testing.T) {
	r, md, sg := runner(t)
	ctx := context.Background()

	// md 正本：v-gone 推荐态；v-alive measuring 态
	for _, q := range []string{
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('v-gone','cex-perp','recommended',true)`,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('v-alive','cex-perp','measuring',true)`,
	} {
		if _, err := md.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// SG 镜像：v-gone 已被手工翻 rejected（终态）——桥投影不得刷回
	if _, err := sg.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-gone','cex-perp','rejected',true)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-alive','cex-perp','candidate',true)`); err != nil {
		t.Fatal(err)
	}

	if err := r.projectVenueCandidates(ctx); err != nil {
		t.Fatal(err)
	}

	var st string
	if err := sg.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-gone'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "rejected" {
		t.Fatalf("终态行被刷回 state=%s want rejected", st)
	}
	if err := sg.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-alive'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "measuring" {
		t.Fatalf("非终态行未随正本迁移 state=%s want measuring", st)
	}
}

// ④册驱动：新终态值入册即被守卫覆盖（不硬编码名单）。
func TestVenueCandGuardRegistryDriven(t *testing.T) {
	r, md, sg := runner(t)
	ctx := context.Background()
	// 'paper' 原非终态——入册后同守
	if _, err := md.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-paper','cex-perp','paper',true)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		VALUES('v-paper','cex-perp','paper',true)`); err != nil {
		t.Fatal(err)
	}
	// md 侧翻 measuring（模拟正本迁移）——入册前应随迁
	if _, err := md.Exec(ctx,
		`UPDATE fin_venue_candidate SET state='measuring' WHERE venue='v-paper'`); err != nil {
		t.Fatal(err)
	}
	if err := r.projectVenueCandidates(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := sg.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-paper'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "measuring" {
		t.Fatalf("入册前 paper 应随正本迁移 state=%s", st)
	}
	// 终态册补 'paper'（SG 侧——守卫子查询在 SG 库执行）+ SG 行落
	// paper 终态（模拟镜像侧终态已立）——md 再翻 recommended 不得刷回
	if _, err := sg.Exec(ctx,
		`INSERT INTO fin_venue_terminal_state(state) VALUES('paper')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sg.Exec(ctx,
		`UPDATE fin_venue_candidate SET state='paper' WHERE venue='v-paper'`); err != nil {
		t.Fatal(err)
	}
	if _, err := md.Exec(ctx,
		`UPDATE fin_venue_candidate SET state='recommended' WHERE venue='v-paper'`); err != nil {
		t.Fatal(err)
	}
	if err := r.projectVenueCandidates(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sg.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue='v-paper'`).Scan(&st); err != nil {
		t.Fatal(err)
	}
	if st != "paper" {
		t.Fatalf("册入终态后 paper 被刷回 state=%s want paper", st)
	}
}
