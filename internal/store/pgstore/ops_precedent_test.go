package pgstore

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
)

// TestOpsPrecedentFace T-0334 判例索引面机检——四干跑场景（同类自动裁/
// 新类照常上行/红线类结构性不裁/出界未命中）+登记臂（裁决落行同拍
// 判例化/翻案替活/「升级」裁词不登记）。
func TestOpsPrecedentFace(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM ops_event"); err != nil {
		t.Fatal(err)
	}
	esc := func(ticket, class, scope, note string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
			VALUES('桩席','升件',$1,$4,$2,$3) RETURNING id`,
			ticket, class, scope, note).Scan(&id); err != nil {
			t.Fatal("升件桩:", err)
		}
		return id
	}
	rule := func(ticket, event, note string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('桩裁',$2,$1,$3) RETURNING id`, ticket, event, note).Scan(&id); err != nil {
			t.Fatal("裁行桩:", err)
		}
		return id
	}
	applied := func(escID int64) int {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
			WHERE event='先例适用' AND note LIKE '升件#'||$1::bigint||' %'`, escID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// 判例桩：class=授权补授 scope=o_fin_ro（源裁行=自造裁行——source_row FK 须实存）。
	src := rule("T-9P01", "业主批", "桩判例源裁")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_precedent(class,scope,ruling,source_row)
		VALUES('授权补授','o_fin_ro','白名单迁移补授',$1)`, src); err != nil {
		t.Fatal("判例桩:", err)
	}

	// 干跑①同类同域升件→自动裁：先例适用行在案。
	id1 := esc("T-9P02", "授权补授", "o_fin_ro", "桩:同类升件")
	if applied(id1) != 1 {
		t.Fatalf("干跑① 同类升件未自动裁——先例适用=%d 应=1", applied(id1))
	}
	// 干跑②新类升件→照常上行不截：零先例适用行。
	id2 := esc("T-9P03", "全新类", "某域", "桩:新类升件")
	if applied(id2) != 0 {
		t.Fatalf("干跑② 新类升件被截——先例适用=%d 应=0", applied(id2))
	}
	// 干跑③红线类升件→结构性不自动裁（即使植入貌似同类判例）。
	if _, err := pool.Exec(ctx, `INSERT INTO ops_precedent(class,scope,ruling,source_row)
		VALUES('资金扩投','某域','永不应适用',$1)`, src); err != nil {
		t.Fatal("红线判例桩:", err)
	}
	id3 := esc("T-9P04", "资金扩投", "某域", "桩:红线升件")
	if applied(id3) != 0 {
		t.Fatalf("干跑③ 红线类被自动裁——先例适用=%d 应=0（类级黑名单失守）", applied(id3))
	}
	// 干跑④出界升件（同类别域）→未命中照常上行。
	id4 := esc("T-9P05", "授权补授", "app_secrets", "桩:出界升件")
	if applied(id4) != 0 {
		t.Fatalf("干跑④ 出界升件误中——先例适用=%d 应=0", applied(id4))
	}
	// 无类升件（esc_class NULL）→永不入判例面。
	var id5 int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('桩席','升件','T-9P06','桩:无类升件') RETURNING id`).Scan(&id5); err != nil {
		t.Fatal(err)
	}
	if applied(id5) != 0 {
		t.Fatalf("无类升件误入判例面——先例适用=%d 应=0", applied(id5))
	}

	// 登记臂①：带类升件被裁（裁决落行）→同拍判例化 class/scope/ruling/source_row。
	esc("T-9P10", "部署窗", "O-Dev", "桩:部署窗升件")
	rid := rule("T-9P10", "裁决", "通过——直派O-Dev")
	var cls, scp, rul string
	var srow, sesc int64
	var act bool
	if err := pool.QueryRow(ctx, `SELECT class,scope,ruling,source_row,COALESCE(source_esc,0),active
		FROM ops_precedent WHERE class='部署窗' AND scope='O-Dev'`,
	).Scan(&cls, &scp, &rul, &srow, &sesc, &act); err != nil {
		t.Fatal("登记臂① 判例未落:", err)
	}
	if srow != rid || !act {
		t.Fatalf("登记臂① source_row=%d(应%d) active=%v", srow, rid, act)
	}
	// 登记臂②翻案：同类同域新裁→旧判例落 active=false+新行 supersedes 链。
	esc("T-9P11", "部署窗", "O-Dev", "桩:部署窗再升")
	rule("T-9P11", "业主批", "翻案——改派COS")
	var nAct, nAll int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE active),count(*)
		FROM ops_precedent WHERE class='部署窗' AND scope='O-Dev'`).Scan(&nAct, &nAll); err != nil {
		t.Fatal(err)
	}
	if nAct != 1 || nAll != 2 {
		t.Fatalf("登记臂② 翻案替活失守——active=%d(应1) total=%d(应2)", nAct, nAll)
	}
	var sup int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(supersedes,0) FROM ops_precedent
		WHERE class='部署窗' AND scope='O-Dev' AND active`).Scan(&sup); err != nil || sup == 0 {
		t.Fatalf("登记臂② supersedes 链缺——supersedes=%d err=%v", sup, err)
	}
	// 登记臂③「升级」裁词不登记（升级=再上行非消化——判例化反吞升件）。
	esc("T-9P12", "选项采纳", "修法候选", "桩:升级升件")
	rule("T-9P12", "裁决", "升级——红线疑似件转业主")
	var nUp int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_precedent
		WHERE class='选项采纳' AND scope='修法候选'`).Scan(&nUp); err != nil {
		t.Fatal(err)
	}
	if nUp != 0 {
		t.Fatalf("登记臂③ 升级裁词被登记——判例行=%d 应=0", nUp)
	}
	// 登记臂④红线类升件被裁亦不登记（双闸——面在亦无判例产出）。
	esc("T-9P13", "席位立换", "某席", "桩:红线被裁")
	rule("T-9P13", "业主批", "桩:红线批")
	var nRed int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_precedent
		WHERE class='席位立换'`).Scan(&nRed); err != nil {
		t.Fatal(err)
	}
	if nRed != 0 {
		t.Fatalf("登记臂④ 红线类被登记——判例行=%d 应=0", nRed)
	}
}
