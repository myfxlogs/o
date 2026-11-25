package main

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// depSegSha 指纹件判据——只吐 sha 不吐龄（A1 实证:「sha(龄)」拼指纹=
// 分钟变=gaugeDue 6h 闸永火,self_state 4522帧/30d）。表驱钉死:
// 合法 sha 形出 12 位;非标/短 sha/空行/缺标记一律 "?"——
// 指纹件宁标 ? 不含时漂串,闸语义靠「结构变才重报」保。
func TestDepSegSha(t *testing.T) {
	for _, c := range []struct{ note, want string }{
		{"deploy_image o-core:de8ba7650d80 rc=0", "de8ba7650d80"},
		{"o-core:abbbf192d031 healthy", "abbbf192d031"},
		{"x o-core:54c01f3be11aa9f0c1d2e3 rc=0 ok", "54c01f3be11a"},
		{"deploy rc=0 no-sha-line", "?"},
		{"o-core:abc123 rc=0", "?"}, // sha<12 位=截断不安全
		{"", "?"},
		{"o-core:ZZZZZZZZZZZZ", "?"}, // 非 hex 不命中 regex
	} {
		if got := depSegSha(c.note); got != c.want {
			t.Errorf("depSegSha(%q)=%q want %q", c.note, got, c.want)
		}
	}
	// 判据钉死:同 note 重取恒同——时漂件（age）无进入径。
	n := "deploy_image o-core:de8ba7650d80 rc=0"
	if depSegSha(n) != depSegSha(n) {
		t.Error("depSegSha 非纯函数——指纹件含时漂=闸报废")
	}
}

// —— O-39420 升件根聚类段桩测（同桶计数/异桶不聚/topN 截断/窗外剔/
// cfg 缺 fail-closed/读败降档/帧集成——spec 四验收全覆）。

func escClusterCfgSeed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, days, top string) {
	t.Helper()
	for _, kv := range [][2]string{
		{"selfstate.esc_cluster_days", days},
		{"selfstate.esc_cluster_topn", top},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO app_config(key,value) VALUES($1,$2::jsonb)
			 ON CONFLICT(key) DO UPDATE SET value=$2::jsonb`, kv[0], kv[1]); err != nil {
			t.Fatalf("cfg 桩 %s: %v", kv[0], err)
		}
	}
}

func escSeed(t *testing.T, ctx context.Context, pool *pgxpool.Pool, class, scope, event, age string) {
	t.Helper()
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope,ts)
		 VALUES('探测席',$1,'o-39420','桩',$2,$3,now()-$4::interval)`,
		event, class, scope, age); err != nil {
		t.Fatalf("升件桩 %s@%s: %v", class, scope, err)
	}
}

// ①同桶×3→疑似同族标+窗外剔+异桶不标+无类/非升件剔。
func TestEscClusterSeg(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	escClusterCfgSeed(t, ctx, pool, "30", "5")
	for i := 0; i < 3; i++ {
		escSeed(t, ctx, pool, "O39420A", "SCOPE_A", "升件", "1 hour")
	}
	escSeed(t, ctx, pool, "O39420A", "SCOPE_A", "升件", "40 days") // 窗外——剔
	escSeed(t, ctx, pool, "O39420B", "SCOPE_B", "升件", "2 hours")
	escSeed(t, ctx, pool, "", "", "升件", "1 hour")               // 无类无域——剔
	escSeed(t, ctx, pool, "O39420C", "SCOPE_C", "量表", "1 hour") // 非升件——剔
	got := (coreService{pool: pool}).escClusterSeg(ctx)
	want := "30d:O39420A@SCOPE_A×3[疑似同族]|O39420B@SCOPE_B×1"
	if got != want {
		t.Fatalf("escClusterSeg=%q want %q", got, want)
	}
}

// ②异桶不聚——全 n=1 时无疑似同族标（①已覆 n=1 单条，此案钉多桶全单）。
func TestEscClusterSegDistinctBuckets(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	escClusterCfgSeed(t, ctx, pool, "30", "5")
	escSeed(t, ctx, pool, "O39420D1", "S1", "升件", "1 hour")
	escSeed(t, ctx, pool, "O39420D2", "S2", "升件", "1 hour")
	escSeed(t, ctx, pool, "O39420D3", "S3", "升件", "1 hour")
	got := (coreService{pool: pool}).escClusterSeg(ctx)
	if strings.Contains(got, "疑似同族") {
		t.Fatalf("异桶全单不应聚类标: %q", got)
	}
	for _, tok := range []string{"O39420D1@S1×1", "O39420D2@S2×1", "O39420D3@S3×1"} {
		if !strings.Contains(got, tok) {
			t.Fatalf("缺桶 %s: %q", tok, got)
		}
	}
}

// ③topN 截断——topn=3 五桶只出前三（计数降序）。
func TestEscClusterSegTopN(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	escClusterCfgSeed(t, ctx, pool, "30", "3")
	for i, p := range []struct{ cls, n string }{
		{"O39420T1", "5"}, {"O39420T2", "4"}, {"O39420T3", "3"}, {"O39420T4", "2"}, {"O39420T5", "1"},
	} {
		for j := 0; j < atoiT(t, p.n); j++ {
			escSeed(t, ctx, pool, p.cls, "S"+string(rune('1'+i)), "升件", "1 hour")
		}
	}
	got := (coreService{pool: pool}).escClusterSeg(ctx)
	want := "30d:O39420T1@S1×5[疑似同族]|O39420T2@S2×4[疑似同族]|O39420T3@S3×3[疑似同族]"
	if got != want {
		t.Fatalf("topN 截断=%q want %q", got, want)
	}
}

func atoiT(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

// ④cfg 缺 fail-closed——双键缺/半键/坏值一律 ? 不炸帧。
func TestEscClusterSegCfgMissing(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	escSeed(t, ctx, pool, "O39420E", "SE", "升件", "1 hour")
	c := coreService{pool: pool}
	if got := c.escClusterSeg(ctx); got != "?" {
		t.Fatalf("双键缺=%q want ?", got)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('selfstate.esc_cluster_days','30'::jsonb)`); err != nil {
		t.Fatal("半键桩:", err)
	}
	if got := c.escClusterSeg(ctx); got != "?" {
		t.Fatalf("半键=%q want ?", got)
	}
}

// ⑤读败降档——ops_event 改名致聚类读败=「读败」降级行非 panic。
func TestEscClusterSegReadFail(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	escClusterCfgSeed(t, ctx, pool, "30", "5")
	if _, err := pool.Exec(ctx, `ALTER TABLE ops_event RENAME TO ops_event_hidden`); err != nil {
		t.Fatal("改名桩:", err)
	}
	if got := (coreService{pool: pool}).escClusterSeg(ctx); got != "读败" {
		t.Fatalf("读败=%q want 读败", got)
	}
}

// ⑥帧集成——selfStateBeat 落 量表/self_state 行含聚类段。
func TestSelfStateBeatClusterNote(t *testing.T) {
	pool := testutil.PoolIsolated(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal("fin migrate:", err)
	}
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal("md migrate:", err)
	}
	escClusterCfgSeed(t, ctx, pool, "30", "5")
	for i := 0; i < 2; i++ {
		escSeed(t, ctx, pool, "O39420F", "SCOPE_F", "升件", "1 hour")
	}
	(coreService{pool: pool}).selfStateBeat(ctx)
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE event='量表' AND ticket='self_state'
		 ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatal("量表行缺席:", err)
	}
	for _, want := range []string{"升件聚类=30d:", "O39420F@SCOPE_F×2[疑似同族]"} {
		if !strings.Contains(note, want) {
			t.Fatalf("帧缺 %q: %s", want, note)
		}
	}
}
