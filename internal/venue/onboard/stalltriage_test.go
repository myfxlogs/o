// stalltriage_test.go —— O-33223 桩测三锚：①recommended 超龄无凭据→
// fin_condition_gap credential 行生+重跑不重行（幂等）；②<venue>.foo
// 非 .mt5. 方言键在册→ops_esc venue_dialect_gap 升件（note 含 venue
// 名+键形）；③recommended 超龄+.mt5. 族凭据已识→零动作。阈内新鲜
// recommended 反例同挂①。
package onboard

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedRecommended recommended 态候选落账（last_request_at 拨 48h 前——
// 恒超 stallTriageAge 阈；fresh 变体拨 now() 阈内反例）。
func seedRecommended(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string, stale bool) {
	t.Helper()
	age := "now()"
	if stale {
		age = "now() - interval '48 hours'"
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_candidate(venue,kind,state,last_request_at)
		VALUES($1,'cex-perp','recommended',`+age+`)`, venue); err != nil {
		t.Fatalf("seed recommended %s: %v", venue, err)
	}
}

func famsOf(t *testing.T, ctx context.Context, r *Runner) map[string]credFamily {
	t.Helper()
	fams, err := r.discover(ctx)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	return fams
}

// 锚① 超龄 recommended+无任何 <venue>.% 键→credential gap pending 行生；
// 重跑幂等不重行；阈内新鲜 recommended 零动作。
func TestStallTriageNoCredGap(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	seedRecommended(t, ctx, pool, "stallno1", true)
	seedRecommended(t, ctx, pool, "stallfresh", false)
	fams := famsOf(t, ctx, r)
	for i := 0; i < 2; i++ { // 重跑幂等断言——两拍零重行
		if err := r.stallTriage(ctx, fams); err != nil {
			t.Fatalf("stallTriage 拍%d: %v", i+1, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE kind='credential' AND venue='stallno1' AND surface='venue-credential'
		  AND state='pending'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("credential gap pending=%d err=%v（应 1 且幂等不重行）", n, err)
	}
	var ask string
	if err := pool.QueryRow(ctx, `SELECT payload->>'ask' FROM fin_condition_gap
		WHERE venue='stallno1' AND surface='venue-credential'`).Scan(&ask); err != nil ||
		ask != "请注册/供凭据 stallno1——venue_req 达阈候凭据推进接入链" {
		t.Fatalf("gap ask=%q err=%v（应钉死文案）", ask, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE venue='stallfresh'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("阈内新鲜 recommended gap=%d（应 0——龄界生效）", n)
	}
}

// 锚② <venue>.foo 键在册但非 .mt5. 方言→ops_esc 升件 venue_dialect_gap
// （note 形含 venue 名+首键形）；函件 24h 前缀窗去重重跑不重行。
func TestStallTriageDialectGap(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	seedRecommended(t, ctx, pool, "dialx", true)
	// 方言不识凭据键——ciphertext 任意（分诊只读键名不解密）。
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_secrets(key,ciphertext) VALUES('dialx.account','\x00')`); err != nil {
		t.Fatal(err)
	}
	fams := famsOf(t, ctx, r)
	if _, ok := fams["dialx"]; ok {
		t.Fatal("dialx 不应被 discover 产族（.mt5. 方言外）")
	}
	for i := 0; i < 2; i++ {
		if err := r.stallTriage(ctx, fams); err != nil {
			t.Fatalf("stallTriage 拍%d: %v", i+1, err)
		}
	}
	var n int
	var note string
	if err := pool.QueryRow(ctx, `SELECT count(*), COALESCE(max(note),'') FROM ops_event
		WHERE event='升件' AND esc_scope='venue_dialect_gap'`).Scan(&n, &note); err != nil || n != 1 {
		t.Fatalf("方言缺口升件 n=%d err=%v（应 1——函件窗去重不重行）", n, err)
	}
	for _, want := range []string{"venue=dialx", "dialx.account"} {
		if !strings.Contains(note, want) {
			t.Fatalf("升件 note=%q 缺 %q（应含 venue 名+键形）", note, want)
		}
	}
	// credential gap 不应落——键在册走方言支不走凭据支。
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE venue='dialx'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("方言不识不应落 credential gap n=%d", n)
	}
}

// 锚③ 超龄 recommended+.mt5. 族凭据已识→零动作（接入链自走）。
func TestStallTriageRecognizedCred(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	seedRecommended(t, ctx, pool, "mt5ok", true)
	seedCreds(t, ctx, pool, "mt5ok", map[string]string{
		"login": "3001157", "password": "pw", "server": "Fake-MT5"})
	fams := famsOf(t, ctx, r)
	if _, ok := fams["mt5ok"]; !ok {
		t.Fatal("mt5ok 应被 discover 产族")
	}
	if err := r.stallTriage(ctx, fams); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE venue='mt5ok'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("已识凭据不应落 gap n=%d", n)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='venue_dialect_gap' AND note LIKE '%mt5ok%'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("已识凭据不应落方言升件 n=%d", n)
	}
}
