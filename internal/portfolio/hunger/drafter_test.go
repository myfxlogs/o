package hunger

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0393 锚①②③④桩测：升件行→pending 草案断言 / 24h 去重 / 在飞票
// dedup / 签名不明 fail-closed；生产谱面零误报复刻（类册外升件不碰）。

func drafterFixture(t *testing.T) (*Sensor, context.Context) {
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
	return New(pool, cfg), ctx
}

// seedEsc 探针升件行素材（class/scope 非空——签名面；空值形见 fail-closed 锚）。
func seedEsc(t *testing.T, s *Sensor, ctx context.Context, class, scope, note string) int64 {
	t.Helper()
	var id int64
	if err := s.pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('决策席','升件','T-0384',$1,$2,$3) RETURNING id`,
		note, class, scope).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestDraftTicketsLesionToDraft 锚①：观测盲升件行→pending 草案+起草行
// 证据引齐（ids/查询式/疑似根域——票面钦三件形态）。
func TestDraftTicketsLesionToDraft(t *testing.T) {
	s, ctx := drafterFixture(t)
	eid := seedEsc(t, s, ctx, "观测盲", "xvenue-carry",
		"族沉默 xvenue-carry：源信号 carry_diff 近7d 鲜行=2 而 fin_opp_eval 该族行=0")
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if n != 1 {
		t.Fatalf("起草数非 1: %d", n)
	}
	var status, kind, key, payload string
	if err := s.pool.QueryRow(ctx, `SELECT status,kind,lesion_key,payload::text
		FROM fin_ticket_draft WHERE signature='观测盲:xvenue-carry'`).Scan(&status, &kind, &key, &payload); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || kind != "观测盲" || key != "xvenue-carry" {
		t.Fatalf("草案行形态非期: %s %s %s", status, kind, key)
	}
	for _, frag := range []string{"xvenue-carry", "evidence_ids", "evidence_query",
		"domain_hint", "SELECT id,ts,seat,note FROM ops_event"} {
		if !strings.Contains(payload, frag) {
			t.Fatalf("载荷缺 %q: %s", frag, payload)
		}
	}
	var note string
	if err := s.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='起草' AND ticket='T-0393'`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{"signature=观测盲:xvenue-carry", "evidence=#"} {
		if !strings.Contains(note, frag) {
			t.Fatalf("起草行缺 %q: %s", frag, note)
		}
	}
	_ = eid
}

// TestDraftTicketsDedup24h 锚②前半：同窗再拍/新行再升——同病灶键 24h
// 至多一草案（pending 占位+防灌窗双闸）。
func TestDraftTicketsDedup24h(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedEsc(t, s, ctx, "观测盲", "xvenue-carry", "族沉默首报")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("首拍: n=%d err=%v", n, err)
	}
	seedEsc(t, s, ctx, "观测盲", "xvenue-carry", "族沉默同窗再报")
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("再拍应零产: n=%d err=%v", n, err)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE signature='观测盲:xvenue-carry'`).Scan(&c); err != nil || c != 1 {
		t.Fatalf("草案防灌失效: c=%d err=%v", c, err)
	}
}

// TestDraftTicketsInflightDedup 锚③：同病灶键已立票在飞（materialized
// 票 latest 事件非终态）→不重复起草+dedup 跳过行落账；票翻完成→
// 病灶复发可再起（24h 窗外）。
func TestDraftTicketsInflightDedup(t *testing.T) {
	s, ctx := drafterFixture(t)
	seedEsc(t, s, ctx, "观测盲", "BTC-USDT", "覆盖失明 BTC-USDT")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("首拍: n=%d err=%v", n, err)
	}
	// 物料化落号 T-9001（桩拟——票面在飞事件链：派单/领单非终态）。
	if _, err := s.pool.Exec(ctx, `UPDATE fin_ticket_draft
		SET status='materialized',ticket='T-9001',materialized_at=now(),ts=now()-interval '48 hours'
		WHERE signature='观测盲:BTC-USDT'`); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []string{"已批", "派单"} {
		if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
			VALUES('决策席',$1,'T-9001','')`, ev); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('施工席','领单','T-9001','')`); err != nil {
		t.Fatal(err)
	}
	seedEsc(t, s, ctx, "观测盲", "BTC-USDT", "覆盖失明 BTC-USDT 再升")
	if n, err := s.DraftTickets(ctx); err != nil || n != 0 {
		t.Fatalf("在飞票应挡: n=%d err=%v", n, err)
	}
	var skip string
	if err := s.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='起草' AND note LIKE 'dedup skip signature=观测盲:BTC-USDT%'`).Scan(&skip); err != nil {
		t.Fatalf("dedup 跳过行缺: %v", err)
	}
	if !strings.Contains(skip, "在飞票=T-9001") {
		t.Fatalf("跳过行缺在飞票引: %s", skip)
	}
	// 票翻完成→同窗（>24h 陈草案）复发可再起草。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('决策席','完成','T-9001','')`); err != nil {
		t.Fatal(err)
	}
	seedEsc(t, s, ctx, "观测盲", "BTC-USDT", "覆盖失明 BTC-USDT 复发")
	if n, err := s.DraftTickets(ctx); err != nil || n != 1 {
		t.Fatalf("完成后复发应再起草: n=%d err=%v", n, err)
	}
}

// TestDraftTicketsFailClosed 锚④：签名不明/证据链缺→不起草只升件
// （类册外类/scope 空=零草案零错——benny「not enough evidence=no issue」）。
func TestDraftTicketsFailClosed(t *testing.T) {
	s, ctx := drafterFixture(t)
	// 类册外类（淤积=巡检类、范围外文件=决策席硬触发类——非探针族）。
	seedEsc(t, s, ctx, "淤积", "fin_opp_eval", "巡检闸量表触阈")
	seedEsc(t, s, ctx, "范围外文件", "T-0391", "diff 动票面范围外文件")
	// 签名残缺（scope 空——本拍扫窗即滤）。
	if _, err := s.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('决策席','升件','','观测盲类无scope','观测盲','')`); err != nil {
		t.Fatal(err)
	}
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("fail-closed 应零错: %v", err)
	}
	if n != 0 {
		t.Fatalf("类册外/签名残缺误起草: n=%d", n)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft`).Scan(&c); err != nil || c != 0 {
		t.Fatalf("fail-closed 破: c=%d err=%v", c, err)
	}
}

// TestDraftTicketsProdLandscape 锚③dry 对拍面：生产谱面复刻——探针族
// 三类签名起草，巡检/判例/席位类升件静默放行不误起。
func TestDraftTicketsProdLandscape(t *testing.T) {
	s, ctx := drafterFixture(t)
	// 探针族升件（三签名各一）。
	seedEsc(t, s, ctx, "观测盲", "xvenue-carry", "族沉默")
	seedEsc(t, s, ctx, "观测盲", "BTC-USDT", "覆盖失明")
	seedEsc(t, s, ctx, "写败", "LIFECYCLE-HUNGER", "hungertick detect 败")
	// 非探针族升件（谱面干扰项——类册外一律不碰）。
	seedEsc(t, s, ctx, "淤积", "fin_proposal", "巡检触阈")
	seedEsc(t, s, ctx, "读败", "okx", "venue 读败连败")
	seedEsc(t, s, ctx, "取数失败", "META-GAUGES", "量表取数败")
	n, err := s.DraftTickets(ctx)
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if n != 3 {
		t.Fatalf("起草数非 3（误报或漏报）: %d", n)
	}
	var c int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM fin_ticket_draft
		WHERE kind NOT IN ('观测盲','写败','账链断')`).Scan(&c); err != nil || c != 0 {
		t.Fatalf("类册外草案混入: c=%d err=%v", c, err)
	}
}
