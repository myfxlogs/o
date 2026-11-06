package hunger

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0385 锚①桩测三形：聚类达阈=pending 行落（签名/锚/草案齐）／未达阈=
// 零行／同签名重扫=去重不产；锚③实况窗回放（0380 ff-merge 单件升件 +
// 0382 种键案驳回双写——生产真行形状复刻）；锚④日额闸 >5 截断；
// 高危单件直起稿面（升件/驳回/异常单件即稿、返修/收口偏离走 ≥2 阈）。

func minerFixture(t *testing.T) (*Sensor, *pgxpool.Pool) {
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

// seedOps 事件账行（esc 空串=NULL——生产无类行形状同）。
func seedOps(t *testing.T, pool *pgxpool.Pool, seat, event, ticket, note, cls, scope string) {
	t.Helper()
	ctx := context.Background()
	var c, s *string
	if cls != "" {
		c = &cls
	}
	if scope != "" {
		s = &scope
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,$2,$3,$4,$5,$6)`, seat, event, ticket, note, c, s); err != nil {
		t.Fatal(err)
	}
}

// seedOpsAged 事件账行定龄（窗外行素材）。
func seedOpsAged(t *testing.T, pool *pgxpool.Pool, age time.Duration, seat, event, ticket, note, cls, scope string) {
	t.Helper()
	ctx := context.Background()
	var c, s *string
	if cls != "" {
		c = &cls
	}
	if scope != "" {
		s = &scope
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope,ts)
		VALUES($1,$2,$3,$4,$5,$6,now()-make_interval(secs=>$7))`,
		seat, event, ticket, note, c, s, age.Seconds()); err != nil {
		t.Fatal(err)
	}
}

func pendingCount(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM fin_eng_lesson WHERE status='pending'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestMinerClusterThresholdProducesPending 形①：同窗同签名带类升件×2 达阈
// →pending 行落——签名/证据锚/草案/建议条目名/验证态五件齐。
func TestMinerClusterThresholdProducesPending(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick detect 败: 窗键不可读", "写败", "LIFECYCLE-HUNGER")
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick respond 败: 步序列中断", "写败", "LIFECYCLE-HUNGER")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 || pendingCount(t, pool) != 1 {
		t.Fatalf("达阈未产: n=%d pending=%d", n, pendingCount(t, pool))
	}
	var sig, vs string
	var cn int
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT signature,cluster_n,verify_state,numbers
		FROM fin_eng_lesson`).Scan(&sig, &cn, &vs, &num); err != nil {
		t.Fatal(err)
	}
	if sig != "升件:写败:LIFECYCLE-HUNGER" || cn != 2 || vs != "ai_evaluated" {
		t.Fatalf("行面坏: sig=%s cn=%d vs=%s", sig, cn, vs)
	}
	var p minerPayload
	if err := json.Unmarshal(num, &p); err != nil {
		t.Fatalf("numbers 非机读: %v", err)
	}
	if len(p.EvidenceIDs) != 2 || p.Draft == "" || p.SuggestedEntry == "" || p.Event != "升件" {
		t.Fatalf("载荷五件不齐: %+v", p)
	}
	// 留账行（自主≠无账）。
	var m int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='教训矿工' AND ticket='T-0385' AND note LIKE 'pending#%'`).Scan(&m); err != nil || m != 1 {
		t.Fatalf("留账行缺: m=%d err=%v", m, err)
	}
}

// TestMinerBelowThresholdZero 形②：未达阈零行——单件非高危（返修）不产。
func TestMinerBelowThresholdZero(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "无头返修席", "返修", "T-0293", "返修认领（R1）", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 0 || pendingCount(t, pool) != 0 {
		t.Fatalf("未达阈误产: n=%d pending=%d", n, pendingCount(t, pool))
	}
}

// TestMinerDedupRescan 形③：同签名重扫=去重不产（pending 恒挡）。
func TestMinerDedupRescan(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick detect 败: a", "写败", "LIFECYCLE-HUNGER")
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick detect 败: b", "写败", "LIFECYCLE-HUNGER")
	if n, err := s.MineLessons(ctx); err != nil || n != 1 {
		t.Fatalf("首挖: n=%d err=%v", n, err)
	}
	// 窗内同签名再添行重扫——pending 在去重挡。
	seedOps(t, pool, "决策席", "升件", "T-0376", "hungertick detect 败: c", "写败", "LIFECYCLE-HUNGER")
	if n, err := s.MineLessons(ctx); err != nil || n != 0 {
		t.Fatalf("去重失效: n=%d err=%v", n, err)
	}
	if pendingCount(t, pool) != 1 {
		t.Fatalf("去重失效 pending=%d", pendingCount(t, pool))
	}
}

// TestMinerHighRiskSingleton 高危单件直起稿：升件/驳回/异常各单件即产。
func TestMinerHighRiskSingleton(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "COS", "升件", "T-0380", "合并走ff径违§9——闸拒推", "", "")
	seedOps(t, pool, "决策席", "驳回", "T-0382", "驳回——收口支全谱过审", "", "")
	seedOps(t, pool, "Dev", "异常", "T-0217", "同票错派副本让位", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 3 || pendingCount(t, pool) != 3 {
		t.Fatalf("高危单件未全产: n=%d pending=%d", n, pendingCount(t, pool))
	}
}

// TestMinerDailyCapTruncates 锚④：日额闸——>5 簇只产 5。
func TestMinerDailyCapTruncates(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	for _, scope := range []string{"s1", "s2", "s3", "s4", "s5", "s6"} {
		seedOps(t, pool, "决策席", "升件", "T-0376", "败情"+scope+"a", "写败", scope)
		seedOps(t, pool, "决策席", "升件", "T-0376", "败情"+scope+"b", "写败", scope)
	}
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != minerDailyCap || pendingCount(t, pool) != minerDailyCap {
		t.Fatalf("日额闸失效: n=%d pending=%d", n, pendingCount(t, pool))
	}
}

// TestMinerOutOfWindowIgnored 窗外行不入聚类（返修非高危——双陈行零产）。
func TestMinerOutOfWindowIgnored(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOpsAged(t, pool, 13*time.Hour, "无头返修席", "返修", "T-0293", "返修认领（R1）", "", "")
	seedOpsAged(t, pool, 14*time.Hour, "无头返修席", "返修", "T-0293", "返修认领（R2）", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 0 {
		t.Fatalf("窗外行误入: n=%d", n)
	}
}

// TestMinerReplay0380ffMerge 锚③实况窗回放①：0380 ff-merge——生产真行形状
// （COS 无类升件单件「合并走ff径违§9」）→高危单件产 升件:T-0380 pending。
func TestMinerReplay0380ffMerge(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	// ops_event#2350 真行复刻（seat/event/ticket/note 前缀+esc NULL）。
	seedOps(t, pool, "COS", "升件", "T-0380",
		"合并走ff径违§9——施工commit(d5498cf2/a3268b59)落master first-parent,T-0370闸拒推(零豁免);解径:reset+merge-train", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 {
		t.Fatalf("0380 案未产: n=%d", n)
	}
	var sig string
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT signature,numbers FROM fin_eng_lesson
		WHERE signature='升件:T-0380'`).Scan(&sig, &num); err != nil {
		t.Fatalf("0380 pending 缺: %v", err)
	}
	var p minerPayload
	if err := json.Unmarshal(num, &p); err != nil || !strings.Contains(p.Draft, "ff径违") {
		t.Fatalf("0380 载荷锚缺: %v %+v", err, p)
	}
}

// TestMinerReplay0382SeedKey 锚③实况窗回放②：0382 种键案——生产真行形状
// （裁决驳回+驳回双写同票同窗——note 驳回* 归一驳回签名类聚簇 n=2 达阈）
// →驳回:T-0382 pending。
func TestMinerReplay0382SeedKey(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	note := "驳回——收口支t/0382-price-alert-notify@c3c94930全谱过审(报价卡DOM12/12+提醒拍四桩测uncached+check_migrations绿)——但漏种notify.feishu.pri键返修"
	// ops_event#2390 裁决行 + #2391 驳回行真行复刻（同票同窗双写径）。
	seedOps(t, pool, "决策席", "裁决", "T-0382", note, "", "")
	seedOps(t, pool, "决策席", "驳回", "T-0382", note, "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 {
		t.Fatalf("0382 案未产: n=%d", n)
	}
	var sig string
	var cn int
	if err := pool.QueryRow(ctx, `SELECT signature,cluster_n FROM fin_eng_lesson
		WHERE signature='驳回:T-0382'`).Scan(&sig, &cn); err != nil || cn != 2 {
		t.Fatalf("0382 pending 缺/簇数坏: sig=%s cn=%d err=%v", sig, cn, err)
	}
}

// TestMinerCloseDeviation 收口偏离签名：收口行 note 含「偏离」单件按类阈
// ≥2 产（非高危签名类——偏离声明同窗两条才挖）。
func TestMinerCloseDeviation(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "施工席", "收口", "T-0999", "收口abc123:实现A;偏离说明:票面B项未做交回", "", "")
	if n, err := s.MineLessons(ctx); err != nil || n != 0 {
		t.Fatalf("单件偏离误产: n=%d err=%v", n, err)
	}
	seedOps(t, pool, "施工席", "收口", "T-0999", "收口def456:返修;偏离说明:B项补落", "", "")
	if n, err := s.MineLessons(ctx); err != nil || n != 1 {
		t.Fatalf("偏离簇未产: n=%d err=%v", n, err)
	}
}

// TestMinerReproHintSeedKey T-0388 锚③：可机判病灶类→pending 载荷携
// repro_hint 建议文（0382 种键案真行复刻——note「漏种…键」命中种键缺类；
// 建议文非可编译码，工人据以写真桩）。
func TestMinerReproHintSeedKey(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	note := "驳回——收口支t/0382-price-alert-notify@c3c94930全谱过审——但漏种notify.feishu.pri键返修"
	seedOps(t, pool, "决策席", "驳回", "T-0382", note, "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 1 {
		t.Fatalf("种键案未产: n=%d", n)
	}
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT numbers FROM fin_eng_lesson
		WHERE signature='驳回:T-0382'`).Scan(&num); err != nil {
		t.Fatalf("pending 缺: %v", err)
	}
	var p minerPayload
	if err := json.Unmarshal(num, &p); err != nil {
		t.Fatalf("numbers 非机读: %v", err)
	}
	if !strings.Contains(p.ReproHint, "种键缺") {
		t.Fatalf("repro_hint 缺种键缺建议文: %+v", p)
	}
}

// TestMinerReproHintFreshDomain 鲜度闸/域拒两类各自命中——升件 note 词面
// 驱动（鲜度词→鲜度闸建议；域拒词→域拒建议）。
func TestMinerReproHintFreshDomain(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "决策席", "异常", "T-9101", "md_ticks 鲜度闸连拒：陈行超阈未愈", "", "")
	seedOps(t, pool, "决策席", "异常", "T-9102", "dmconfig 写径域拒：owner 类键 DMSet 拒", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 2 {
		t.Fatalf("两案未全产: n=%d", n)
	}
	hints := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT signature,numbers FROM fin_eng_lesson`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var sig string
		var num []byte
		if err := rows.Scan(&sig, &num); err != nil {
			t.Fatal(err)
		}
		var p minerPayload
		if err := json.Unmarshal(num, &p); err != nil {
			t.Fatalf("numbers 非机读: %v", err)
		}
		hints[sig] = p.ReproHint
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hints["异常:T-9101"], "鲜度闸") {
		t.Fatalf("鲜度闸建议缺: %s", hints["异常:T-9101"])
	}
	if !strings.Contains(hints["异常:T-9102"], "域拒") {
		t.Fatalf("域拒建议缺: %s", hints["异常:T-9102"])
	}
}

// TestMinerReproHintAbsent 非机判类零载——无类词行 repro_hint 缺席不硬塞
// （omitempty：载荷面零假阳性域）。
func TestMinerReproHintAbsent(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "COS", "升件", "T-0380", "合并走ff径违§9——闸拒推", "", "")
	if n, err := s.MineLessons(ctx); err != nil || n != 1 {
		t.Fatalf("mine: n=%d err=%v", n, err)
	}
	var num []byte
	if err := pool.QueryRow(ctx, `SELECT numbers FROM fin_eng_lesson`).Scan(&num); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(num), "repro_hint") {
		t.Fatalf("非机判类误携 repro_hint: %s", num)
	}
}

// TestMinerSilentNoSignal benny 沉默律：窗内零签名集行=零产零行。
func TestMinerSilentNoSignal(t *testing.T) {
	s, pool := minerFixture(t)
	ctx := context.Background()
	seedOps(t, pool, "施工席", "领单", "T-0385", "领单开工", "", "")
	seedOps(t, pool, "施工席", "收口", "T-0384", "收口aa:全锚绿", "", "")
	n, err := s.MineLessons(ctx)
	if err != nil {
		t.Fatalf("mine: %v", err)
	}
	if n != 0 || pendingCount(t, pool) != 0 {
		t.Fatalf("无信号误产: n=%d", n)
	}
}
