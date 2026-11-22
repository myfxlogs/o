package pgstore

import (
	"context"
	"fmt"
	"testing"

	"o/internal/platform/testutil"
)

// TestOpsEscPendingClusterFold 工单O-27914 同根簇折叠机检（fin_0230）——
// 簇签=note 首个 '#<digits>' ∨ ticket 列数字票根(>=4位)；处置行引根票
// =簇销；同簇折最新行 valence=簇内 max；无签单簇不折。
func TestOpsEscPendingClusterFold(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM ops_event"); err != nil {
		t.Fatal(err)
	}
	esc := func(class, scope, ticket, note, age string) int64 {
		var id int64
		if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope,ts)
			VALUES('桩席','升件',$1,$4,$2,$3, now()-$5::interval) RETURNING id`,
			ticket, class, scope, note, age).Scan(&id); err != nil {
			t.Fatal("升件桩:", err)
		}
		return id
	}
	disp := func(ticket, note, age string) {
		if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
			VALUES('处置席','处置',$1,$2, now()-$3::interval)`, ticket, note, age); err != nil {
			t.Fatal("处置桩:", err)
		}
	}
	pendingIDs := func() map[int64]bool {
		rows, err := pool.Query(ctx, `SELECT id FROM ops_esc_pending`)
		if err != nil {
			t.Fatal("视图读:", err)
		}
		defer rows.Close()
		m := map[int64]bool{}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			m[id] = true
		}
		return m
	}
	valenceOf := func(id int64) int {
		var v int
		if err := pool.QueryRow(ctx, `SELECT valence FROM ops_esc_pending WHERE id=$1`, id).Scan(&v); err != nil {
			t.Fatalf("valence 读 id=%d: %v", id, err)
		}
		return v
	}

	// ── 簇A：note 根票签 3 scope 同簇 → 折最新行，valence=簇内 max ──
	// 权重种子：断流=75 读败=60 孤儿账=45——幸存行(孤儿账)须带 75。
	a1 := esc("断流", "面甲", "", "桩面断流——根因#999001", "3 hours")
	a2 := esc("读败", "面乙", "", "桩读败续火——根因#999001", "2 hours")
	a3 := esc("孤儿账", "面丙", "", "桩孤儿续火——根因#999001", "1 hours")
	pend := pendingIDs()
	if pend[a1] || pend[a2] {
		t.Fatalf("簇A折叠失守——旧行仍在未决面 a1=%v a2=%v", pend[a1], pend[a2])
	}
	if !pend[a3] {
		t.Fatal("簇A幸存行缺席——最新行应在未决面")
	}
	if v := valenceOf(a3); v != 75 {
		t.Fatalf("簇A valence=%d 应=75（簇内max=断流权）——惧值账丢", v)
	}

	// ── 簇B：ticket 数字票根签 2 行 → 折最新 ──
	b1 := esc("孤儿账", "面丁", "O-999002", "桩票龄升件无hash引用", "2 hours")
	b2 := esc("断流", "面戊", "工单#999002", "桩票龄续火", "1 hours")
	pend = pendingIDs()
	if pend[b1] || !pend[b2] {
		t.Fatalf("簇B ticket根折叠失守——b1=%v(应false) b2=%v(应true)", pend[b1], pend[b2])
	}

	// ── 无签单簇：note 无 #num 且 ticket 空 → 不折不并 ──
	c1 := esc("断流", "面己", "", "桩断流无根票引用", "2 hours")
	c2 := esc("断流", "面庚", "", "桩断流另一scope无根票", "1 hours")
	pend = pendingIDs()
	if !pend[c1] || !pend[c2] {
		t.Fatalf("无签行被误并——c1=%v c2=%v 应双在", pend[c1], pend[c2])
	}

	// ── 簇销①：处置行 note 引根票 → 簇全销 ──
	d1 := esc("断流", "面辛", "", "桩断流——根因#999003", "3 hours")
	d2 := esc("读败", "面壬", "", "桩读败——根因#999003", "2 hours")
	disp("", "销——根因#999003 已处置", "1 hours")
	pend = pendingIDs()
	if pend[d1] || pend[d2] {
		t.Fatalf("簇销①失守——处置行引根票后簇行仍在 d1=%v d2=%v", pend[d1], pend[d2])
	}

	// ── 簇销②：处置行 ticket 数字根同 sig（note 无引用）→ 簇全销 ──
	e1 := esc("断流", "面癸", "", "桩断流——根因#999004", "3 hours")
	e2 := esc("读败", "面子", "", "桩读败——根因#999004", "2 hours")
	disp("O-999004", "处置行票根同簇", "1 hours")
	pend = pendingIDs()
	if pend[e1] || pend[e2] {
		t.Fatalf("簇销②失守——ticket根同值处置后簇行仍在 e1=%v e2=%v", pend[e1], pend[e2])
	}

	// ── 反例：非处置行引根票不销簇（event='记事'不算处置行）──
	f1 := esc("断流", "面丑", "", "桩断流——根因#999005", "3 hours")
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,ts)
		VALUES('唤知席','记事','','提及根因#999005 的记事行', now()-interval '1 hours')`); err != nil {
		t.Fatal(err)
	}
	pend = pendingIDs()
	if !pend[f1] {
		t.Fatal("非处置行误销簇——记事行引根票不应关簇")
	}

	// ── 时序边界：处置早于升件不销（x.ts>b.ts 严格后随）──
	g1 := esc("断流", "面寅", "", "桩断流——根因#999006", "1 hours")
	disp("", "销——根因#999006 早先处置", "3 hours")
	pend = pendingIDs()
	if !pend[g1] {
		t.Fatal("时序失守——处置早于升件不应销簇")
	}

	// ── 既有升件#id 口径不互斥：单行被引仍单行销，余行照常折 ──
	h1 := esc("断流", "面卯", "", "桩断流——根因#999007", "3 hours")
	h2 := esc("读败", "面辰", "", "桩读败——根因#999007", "2 hours")
	disp("", fmt.Sprintf("销——升件#%d 单行处置", h1), "1 hours")
	pend = pendingIDs()
	if pend[h1] {
		t.Fatal("升件#id 单行销失守——h1 仍在未决面")
	}
	if !pend[h2] {
		t.Fatal("簇内余行被误并灭——h2 应存活为簇折后唯一行")
	}
}
