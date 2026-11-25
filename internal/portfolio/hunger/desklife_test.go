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
