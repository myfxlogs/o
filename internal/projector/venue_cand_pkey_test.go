// venue_cand_pkey_test.go —— O-41373②锚：uVenueCand pick CTE 防撞——
// SG 预埋 (12,v-ex) 占 id=12，md 正本批含 (12,v-mt)+(1099,v-ex)：
// 全批提交零 23505；v-mt 另发号落（id≠12）、v-ex venue 冲突更新原行
// （id 保 12 不换——venue 自然键赢同 uEval 判例）。
package projector

import (
	"context"
	"testing"
)

func TestProjectVenueCandidates_IdCollision(t *testing.T) {
	r, md, sg := runner(t)
	ctx := context.Background()

	// SG 预埋撞行：id=12 被异 venue 占（2026-11-25 生产病灶同构——
	// exness@12 残留撞 md mt5@12）。
	if _, err := sg.Exec(ctx, `INSERT INTO fin_venue_candidate(id,venue,kind,state,kyc_ok)
		VALUES(12,'v-ex','cex-perp','candidate',true)`); err != nil {
		t.Fatal(err)
	}
	// md 正本：v-mt 携撞键 id=12；v-ex 携 md 发散号 1099。
	for _, q := range []string{
		`INSERT INTO fin_venue_candidate(id,venue,kind,state,kyc_ok)
		 VALUES(12,'v-mt','cex-perp','measuring',true)`,
		`INSERT INTO fin_venue_candidate(id,venue,kind,state,kyc_ok)
		 VALUES(1099,'v-ex','cex-perp','measuring',true)`,
	} {
		if _, err := md.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	if err := r.projectVenueCandidates(ctx); err != nil {
		t.Fatalf("撞键批投影败（23505 复现=pick CTE 未生效）: %v", err)
	}

	// v-mt：另发号落——id≠12 且 venue 自然键在册。
	var mtID int64
	var mtState string
	if err := sg.QueryRow(ctx,
		`SELECT id,state FROM fin_venue_candidate WHERE venue='v-mt'`).Scan(&mtID, &mtState); err != nil {
		t.Fatalf("v-mt 未落 SG: %v", err)
	}
	if mtID == 12 {
		t.Fatalf("v-mt 仍携撞键 id=12——pick CTE 未另发号")
	}
	if mtState != "measuring" {
		t.Fatalf("v-mt state=%s want measuring（md 正本值随行）", mtState)
	}

	// v-ex：venue 冲突更新原行——id 保 12（SG 侧行不换 id），state 随 md。
	var exID int64
	var exState string
	if err := sg.QueryRow(ctx,
		`SELECT id,state FROM fin_venue_candidate WHERE venue='v-ex'`).Scan(&exID, &exState); err != nil {
		t.Fatalf("v-ex 查询败: %v", err)
	}
	if exID != 12 {
		t.Fatalf("v-ex id=%d want 12（venue 冲突更新不换 id）", exID)
	}
	if exState != "measuring" {
		t.Fatalf("v-ex state=%s want measuring（venue 冲突 DO UPDATE 生效）", exState)
	}

	// 账面齐：v-mt/v-ex 双 venue 在册，id 域无 23505 残留（全批已提交）。
	var n int
	if err := sg.QueryRow(ctx,
		`SELECT count(*) FROM fin_venue_candidate WHERE venue IN ('v-mt','v-ex')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("venue 在册数=%d want 2", n)
	}
}
