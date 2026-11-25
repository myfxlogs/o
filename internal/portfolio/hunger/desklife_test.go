package hunger

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// O-39400 锚桩（票桌生命周期探针——交付实据销/前提死亡/陈腐衰减阀）：
// ①契形回执在案未销票→处置销行落（desklife 全域版=CLOSESET 同谓词）；
// ②venue=rejected 引用票→premise-dead 直销行；③note 命中 retired
// template_ref→premise_dead 升件不直杀；④deadline+14d 零活动→
// desk_stale 升件且二次拍不重复（单次）；⑤窗键缺→fail-closed；
// ⑥susp/rulable/claimed 票不碰（禁误杀锚）。

func desklifeFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return New(pool, cfg), pool
}

// seedDeskTicket 落一张 desk 工单票（ops_event event='工单'——视图
// 30d 窗内 id=该行 id；ageDays=票龄回拨 filed_ts/deadline/last_active
// 同步老化——decay 判据全吃 filed_ts 派生面）。
func seedDesklifeTicket(t *testing.T, pool *pgxpool.Pool, note string, ageDays float64) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note,ts)
		 VALUES('自我','工单','', $1, now()-make_interval(days=>$2)) RETURNING id`,
		note, ageDays).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func desklifeCloseRows(t *testing.T, pool *pgxpool.Pool, id int64) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='处置'
		 AND note ~ ('工单#'||$1||'([^0-9]|$)') AND note ~ '销'`, itoa(id)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func desklifeEscCount(t *testing.T, pool *pgxpool.Pool, class, scope string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM ops_event WHERE event='升件' AND esc_class=$1 AND esc_scope=$2`,
		class, scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDesklifeDeliveredSweep(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	id := seedDesklifeTicket(t, pool, "工单/foo-deliver: 某件交付", 1)
	// 契形回执（完工回执词位+票号引用——CLOSESET 谓词同款）
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('施工','完工','', $1)`,
		"完工回执：支已推 O-"+itoa(id)+" @ sha=abcdef123456"); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeCloseRows(t, pool, id); n != 1 {
		t.Fatalf("交付实据销行=%d want 1", n)
	}
}

func TestDesklifePremiseVenueRejected(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('dead-venue','cex-perp','rejected',false)`); err != nil {
		t.Fatal(err)
	}
	id := seedDesklifeTicket(t, pool, "工单[venue=dead-venue·达阈清单] P3 资料整理", 1)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeCloseRows(t, pool, id); n != 1 {
		t.Fatalf("venue判死直销行=%d want 1", n)
	}
}

func TestDesklifePremiseRetiredEscalates(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		 VALUES('TEST-USDT-SWAP','dead_tmpl_x','retired',100)`); err != nil {
		t.Fatal(err)
	}
	id := seedDesklifeTicket(t, pool, "工单/foo-deadref: dead_tmpl_x 族复活评审", 1)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeEscCount(t, pool, deskPremiseEscClass, itoa(id)); n != 1 {
		t.Fatalf("premise_dead 升件=%d want 1", n)
	}
	if n := desklifeCloseRows(t, pool, id); n != 0 {
		t.Fatalf("存疑前提误直销=%d want 0", n)
	}
}

func TestDesklifeDecayOnce(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	id := seedDesklifeTicket(t, pool, "工单/foo-stale: 陈旧票", 20) // deadline≈19d前+14d窗→已超
	scope := itoa(id)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeEscCount(t, pool, deskStaleEscClass, scope); n != 1 {
		t.Fatalf("desk_stale 升件=%d want 1", n)
	}
	// 二次拍=单次不刷屏
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeEscCount(t, pool, deskStaleEscClass, scope); n != 1 {
		t.Fatalf("desk_stale 二拍=%d want 1（单次）", n)
	}
}

func TestDesklifeFailClosed(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`DELETE FROM app_config WHERE key='dm.hunger.desk_decay_days'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	s := New(pool, cfg)
	if err := s.ProbeDeskLifecycle(ctx); err == nil {
		t.Fatal("窗键缺=应 fail-closed 报错不探")
	}
}

// 禁误杀锚：挂起票（susp_until 未到）venue 判死也不直销——前提死
// 不等于可越权杀在办挂起面。
func TestDesklifeSuspendedUntouched(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('dead-venue2','cex-perp','rejected',false)`); err != nil {
		t.Fatal(err)
	}
	id := seedDesklifeTicket(t, pool, "工单[venue=dead-venue2·x] P3", 1)
	// act.id=源 ops_event 行 id——另落一行 处置 作宿主防 pkey 撞
	var actHost int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		 VALUES('自我','处置','工单',$1) RETURNING id`,
		"工单#"+itoa(id)+" 挂起条件=裁定——挂起期限=9999h").Scan(&actHost); err != nil {
		t.Fatal(err)
	}
	// act.id=源 ops_event 行 id 语义位——恢复库序列错位防撞：取大偏移
	// 唯一值（无 FK——id 仅 PK；视图 susp 支只读 rel/act/susp_*）。
	actID := actHost + 100000000
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level,tkind)
		 VALUES($1,now(),'探测席','处置',$2,'suspend','adjudication',9999,NULL,NULL)`, actID, id); err != nil {
		t.Fatal(err)
	}
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeCloseRows(t, pool, id); n != 0 {
		t.Fatalf("挂起票误销=%d want 0", n)
	}
}

// ── O-39410 锚契约锚桩 ─────────────────────────────────────────

// seedAnchoredTicket 显式锚票（sem 锚键先行——derive 见 anchor_type
// 已在=返回空=显式恒赢；anchor_type='manual' 模拟无锚票）。
func seedAnchoredTicket(t *testing.T, pool *pgxpool.Pool, note, atype, aref string, ageDays float64) int64 {
	t.Helper()
	var id int64
	sem := `{"anchor_type":"` + atype + `","anchor_ref":"` + aref + `"}`
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO ops_event(seat,event,ticket,note,sem,ts)
		 VALUES('自我','工单','', $1, $2::jsonb, now()-make_interval(days=>$3)) RETURNING id`,
		note, sem, ageDays).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// 锚触发器：venue= 票出生即得 premise/venue:<v> 锚（sem jsonb 键）。
func TestDeskAnchorTriggerDerives(t *testing.T) {
	_, pool := desklifeFixture(t)
	ctx := context.Background()
	id := seedDesklifeTicket(t, pool, "工单[venue=anchor-v·x] P3", 1)
	var at, ar string
	if err := pool.QueryRow(ctx,
		`SELECT sem->>'anchor_type', sem->>'anchor_ref' FROM ops_event WHERE id=$1`, id).
		Scan(&at, &ar); err != nil {
		t.Fatal(err)
	}
	if at != "premise" || ar != "venue:anchor-v" {
		t.Fatalf("锚推导错 at=%s ar=%s", at, ar)
	}
	// 显式锚恒赢——sem 已带 anchor_type 不覆写
	id2 := seedAnchoredTicket(t, pool, "工单[venue=other-v·x] P3", "design", "D-0001", 1)
	if err := pool.QueryRow(ctx,
		`SELECT sem->>'anchor_type', sem->>'anchor_ref' FROM ops_event WHERE id=$1`, id2).
		Scan(&at, &ar); err != nil {
		t.Fatal(err)
	}
	if at != "design" || ar != "D-0001" {
		t.Fatalf("显式锚被覆写 at=%s ar=%s", at, ar)
	}
	// 无可提票=manual
	id3 := seedDesklifeTicket(t, pool, "工单/plain-ticket: 无锚文本", 1)
	if err := pool.QueryRow(ctx,
		`SELECT sem->>'anchor_type' FROM ops_event WHERE id=$1`, id3).Scan(&at); err != nil {
		t.Fatal(err)
	}
	if at != "manual" {
		t.Fatalf("无锚票锚型=%s want manual", at)
	}
}

// 锚径：premise/venue: 硬引用+rejected→直销（①带锚销）。
func TestDesklifeAnchoredVenueCloses(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('av-dead','cex-perp','rejected',false)`); err != nil {
		t.Fatal(err)
	}
	id := seedAnchoredTicket(t, pool, "工单/anchored: 显式锚票", "premise", "venue:av-dead", 1)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeCloseRows(t, pool, id); n != 1 {
		t.Fatalf("锚径直销行=%d want 1", n)
	}
}

// 锚径：venue 活态→不动作；锚指对象缺席→升件不直杀。
func TestDesklifeAnchoredVenueAliveAndMissing(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('av-alive','cex-perp','recommended',true)`); err != nil {
		t.Fatal(err)
	}
	alive := seedAnchoredTicket(t, pool, "工单/anchored-alive", "premise", "venue:av-alive", 1)
	missing := seedAnchoredTicket(t, pool, "工单/anchored-miss", "premise", "venue:av-ghost", 1)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeCloseRows(t, pool, alive); n != 0 {
		t.Fatalf("venue 活票误销=%d", n)
	}
	if n := desklifeEscCount(t, pool, deskPremiseEscClass, itoa(missing)); n != 1 {
		t.Fatalf("锚指缺席升件=%d want 1", n)
	}
	if n := desklifeCloseRows(t, pool, missing); n != 0 {
		t.Fatalf("锚指缺席误销=%d", n)
	}
}

// 无锚票兜底：manual 锚+note venue=rejected→只升件不直销（②）。
func TestDesklifeUnanchoredVenueEscalatesOnly(t *testing.T) {
	s, pool := desklifeFixture(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_venue_candidate(venue,kind,state,kyc_ok)
		 VALUES('uv-dead','cex-perp','rejected',false)`); err != nil {
		t.Fatal(err)
	}
	// 显式 manual 锚=无锚票（derive 不动）——note 仍引 rejected venue
	id := seedAnchoredTicket(t, pool, "工单/manual-anchor: venue=uv-dead 旧票形", "manual", "", 1)
	if err := s.ProbeDeskLifecycle(ctx); err != nil {
		t.Fatal(err)
	}
	if n := desklifeEscCount(t, pool, deskPremiseEscClass, itoa(id)); n != 1 {
		t.Fatalf("无锚兜底升件=%d want 1", n)
	}
	if n := desklifeCloseRows(t, pool, id); n != 0 {
		t.Fatalf("无锚票误直销=%d want 0", n)
	}
}
