package ledger

// diff_human_test.go —— 账差升件白话账段锚（工单O-36964）：
// ①实案形（$67 族）：App实存分量相加+豁免逐行白话名+豁后残差+需动手判词；
// ②白话段零机器词（resolver|baseline|容差|锚|现算|parked|nativeBase 全文检索）；
// ③豁免零条→「无豁免条目」行；④分量缺→降级显总额不虚构；
// ⑤静态行白话名=reason 首段剥票据锚，含禁词→「已定性残差#id」兜底；
// ⑥升件 note 头部=白话段、机读底稿段原样保留。

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// seedResidReason 静态/跟踪豁免行桩（reason 自定——白话名料）。
func seedResidReason(t *testing.T, pool *pgxpool.Pool, venue, ccy string,
	amt float64, reason, resolver string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_known_residual(venue,ccy,amount_usd,reason,ticket_ref,expires_at,resolver)
		 VALUES($1,$2,$3,$4,'t',now()+interval '30 days',$5)`,
		venue, ccy, amt, reason, resolver); err != nil {
		t.Fatal(err)
	}
}

// seedInterest 息流行桩（earn_accrual_gap resolver 现算源）。
func seedInterest(t *testing.T, pool *pgxpool.Pool, amt float64) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		 VALUES('interest','okx','USDT',$1,880001,'md_account_events',now())`, amt); err != nil {
		t.Fatal(err)
	}
}

// incidentParts 实案分量形：Earn理财+交易户+DCD期权=App实读合计。
func incidentParts() []BalPart {
	return []BalPart{
		{Name: "earn", Amount: 9385.66},
		{Name: "trading", Amount: 41.80},
		{Name: "dcd", Amount: 50.00},
	}
}

// incidentExempts 实案豁免形：dcd跟踪回退+earn跟踪+静态尘。
func incidentExempts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	seedResidReason(t, pool, "okx", "USDT", 53,
		"构性残差-DCD金源市值漂移口径差", "dcd_mtm_drift:11448262")
	seedResidReason(t, pool, "okx", "USDT", 10,
		"构性残差-Earn应计时基差", "earn_accrual_gap")
	seedResidReason(t, pool, "okx", "USDT", 4.28,
		"11-22手动调账链残差尘（earn赎回/申购churn+resolver口径见#36422）", "")
	seedInterest(t, pool, 13.64)
}

func TestHumanSegmentIncidentShape(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	incidentExempts(t, pool)
	// 实案数：eq=9477.46 net=9544.69 diff=67.23 exempt=70.92→残差<0→无需动手
	seg := r.ledgerDiffHuman(ctx, "okx", "USDT",
		9477.46, 9544.69, 67.23, 0.50, 70.92, incidentParts())
	for _, want := range []string{
		"OKX USDT 账实差 $67.23",
		"App实存", "交易户 41.80", "Earn理财 9385.66", "DCD期权 50.00", "合计9477.46",
		"vs 账本记 9544.69",
		"DCD期权市价漂移", "记账基准$53.00",
		"Earn应计息未结算", "实时值$13.64",
		"已定性残差·11-22手动调账链残差尘",
		"豁后残差", "无需动手",
	} {
		if !strings.Contains(seg, want) {
			t.Fatalf("白话段缺 %q —— seg=%s", want, seg)
		}
	}
	for _, bad := range humanBanned {
		if strings.Contains(seg, bad) {
			t.Fatalf("白话段含禁词 %q —— seg=%s", bad, seg)
		}
	}
}

func TestHumanSegmentNoExempt(t *testing.T) {
	pool := trkFixture(t)
	r := trkRunner(pool)
	seg := r.ledgerDiffHuman(context.Background(), "okx", "USDT",
		100.0, 110.0, 10.0, 0.5, 0, incidentParts())
	if !strings.Contains(seg, "无豁免条目") {
		t.Fatalf("零豁免未显「无豁免条目」: %s", seg)
	}
	if !strings.Contains(seg, "待定性") {
		t.Fatalf("残差>允许差应判待定性: %s", seg)
	}
}

func TestHumanSegmentPartsMissing(t *testing.T) {
	pool := trkFixture(t)
	r := trkRunner(pool)
	seg := r.ledgerDiffHuman(context.Background(), "mt5", "USD",
		1.80, 1.80, 0.0, 0.5, 0, nil)
	if !strings.Contains(seg, "分量明细缺") {
		t.Fatalf("分量缺未降级显名: %s", seg)
	}
	if strings.Contains(seg, "交易户") || strings.Contains(seg, "Earn理财") {
		t.Fatalf("分量缺不应虚构分量: %s", seg)
	}
}

func TestHumanSegmentPendingVerdict(t *testing.T) {
	pool := trkFixture(t)
	r := trkRunner(pool)
	// 残差>允许差→待定性取证底稿见下段
	seg := r.ledgerDiffHuman(context.Background(), "okx", "USDT",
		100.0, 200.0, 100.0, 0.5, 4.28, incidentParts())
	if !strings.Contains(seg, "豁后残差 $95.72") || !strings.Contains(seg, "待定性，取证底稿见下段") {
		t.Fatalf("待定性判词缺: %s", seg)
	}
}

func TestStaticPlainName(t *testing.T) {
	cases := []struct {
		reason string
		want   string
	}{
		{"11-22手动调账链残差尘（venue实读差豁后实测）", "已定性残差·11-22手动调账链残差尘"},
		{"升件#34532定性：DCD-11448262(BTC-261225-95000-C)漂移", "已定性残差·DCD-11448262"},
		{"工单/accrete-residual-audit2 续档残差", "已定性残差·续档残差"},
		{"含resolver口径见#36422的尘", "已定性残差#99"}, // 禁词→兜底（id=99 桩值）
		{"", "已定性残差#7"},
	}
	for i, c := range cases {
		id := int64(i + 1)
		if i == 3 {
			id = 99
		}
		if i == 4 {
			id = 7
		}
		if got := staticPlainName(id, c.reason); got != c.want {
			t.Fatalf("case%d: got=%q want=%q", i, got, c.want)
		}
	}
}

func TestEscalationNoteHeadIsHuman(t *testing.T) {
	pool := trkFixture(t)
	ctx := context.Background()
	r := trkRunner(pool)
	incidentExempts(t, pool)
	r.alertLedgerDiff(ctx, "okx", "USDT", 9477.46, 9544.69, 0.50, 70.92, 67.23, true, incidentParts())
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE event='升件' AND esc_scope='ledger_diff_okx_USDT'
		 ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	// 头部=白话段：scope 标签后首段即「OKX USDT 账实差」人话起手
	if !strings.HasPrefix(note, "对账/ledger_diff_okx_USDT: OKX USDT 账实差 $67.23——App实存") {
		t.Fatalf("升件 note 头非白话段: %s", note[:min(120, len(note))])
	}
	// 机读底稿段原样保留（容差/venue读/→brief 全在）
	if !strings.Contains(note, "——账实差 $67.23（容差 $0.50") ||
		!strings.Contains(note, "venue 现读 9477.4600 vs 账本净 9544.6900") {
		t.Fatalf("机读底稿段缺: %s", note)
	}
	// 白话段禁词零命中（截取「——账实差」前段检索）
	human := note
	if i := strings.Index(note, "——账实差"); i > 0 {
		human = note[:i]
	}
	for _, bad := range humanBanned {
		if strings.Contains(human, bad) {
			t.Fatalf("白话段含禁词 %q: %s", bad, human)
		}
	}
}
