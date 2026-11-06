package notify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚②：建行→价到→fired 翻位+ops_event 行+推送调用断言（Sender 桩）。
// 锚③：fired 行不再触发（一发即锁）。锚④：notify 败=自报行+行不翻位
// （票不丢下拍重试）；败愈后重拍=补推+fired。
func TestPriceAlertFireOnce(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 种键断言（返修1钉——fixture on()=true 桩与生产真形键缺=false 分歧案；
	// Migrate 全链跑 fin_0119 自携种，键缺/值歪=生产 DOA 同病灶再现）。
	var seed string
	if err := pool.QueryRow(ctx, `SELECT value::text FROM app_config
		WHERE key='notify.feishu.price_alert.enabled'`).Scan(&seed); err != nil || seed != "true" {
		t.Fatalf("开关键种应='true': %q err=%v", seed, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask)
		VALUES(now(),'okx','BTC-USDT-SWAP',61000,60999,61001)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_price_alert(symbol,direction,target,created_by)
		VALUES('BTC-USDT-SWAP','>=',60000,'桩')`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	p := NewPriceAlert(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 || !strings.Contains(fs.sent[0], "BTC-USDT-SWAP") ||
		!strings.Contains(fs.sent[0], "61000") || !strings.Contains(fs.sent[0], "60000") {
		t.Fatalf("推送文不符: %+v", fs.sent)
	}
	var state string
	var firedAt *string
	if err := pool.QueryRow(ctx, `SELECT state,fired_at::text FROM fin_price_alert`).
		Scan(&state, &firedAt); err != nil || state != "fired" || firedAt == nil {
		t.Fatalf("翻位不符: state=%s fired_at=%v err=%v", state, firedAt, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='价格提醒' AND note LIKE '%BTC-USDT-SWAP%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("ops_event 行应=1: %d", n)
	}
	// 锚③：重扫零推零行（fired 一发即锁）
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("重扫不应重推: %d", len(fs.sent))
	}
}

// 未命中不推不翻（< 方向反例同扫）。
func TestPriceAlertNoHit(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask)
		VALUES(now(),'okx','TRX-USDT-SWAP',0.28,0.279,0.281)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_price_alert(symbol,direction,target)
		VALUES('TRX-USDT-SWAP','<=',0.20)`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	p := NewPriceAlert(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("未命中不应推: %+v", fs.sent)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_price_alert`).Scan(&st); err != nil || st != "active" {
		t.Fatalf("未命中行应保持 active: %s", st)
	}
}

// 锚④：推送败=行不翻 fired（票不丢）+升件自报行；败愈重拍=补推+fired。
type failSender struct{}

func (f *failSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	return "", errors.New("feishu 桩败")
}

func TestPriceAlertPushFailKeepsActive(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','ETH-USDT-SWAP',3000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_price_alert(symbol,direction,target)
		VALUES('ETH-USDT-SWAP','>=',2500)`); err != nil {
		t.Fatal(err)
	}
	p := NewPriceAlert(pool, &failSender{}, func() string { return "c1" }, func() bool { return true })
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_price_alert`).Scan(&st); err != nil || st != "active" {
		t.Fatalf("推败行应保持 active（票不丢）: %s", st)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='NOTIFY-PRICE-ALERT'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自报升件行应=1: %d", n)
	}
	// 同窗去重：再败不再落升件行
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='NOTIFY-PRICE-ALERT'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("升件同窗应去重仍=1: %d", n)
	}
	// 败愈（换好桩重拍）=补推+fired 翻位
	fs := &fakeSender{}
	p2 := NewPriceAlert(pool, fs, func() string { return "c1" }, func() bool { return true })
	if err := p2.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 1 {
		t.Fatalf("败愈应补推: %d", len(fs.sent))
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_price_alert`).Scan(&st); err != nil || st != "fired" {
		t.Fatalf("败愈重拍应 fired: %s", st)
	}
}

// 关态（on=false）=不发只翻位+ops_event 注「推送关」。
func TestPriceAlertOffStillFires(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
		VALUES(now(),'okx','BTC-USDT',70000)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_price_alert(symbol,direction,target)
		VALUES('BTC-USDT','<=',80000)`); err != nil {
		t.Fatal(err)
	}
	fs := &fakeSender{}
	p := NewPriceAlert(pool, fs, func() string { return "c1" }, func() bool { return false })
	if err := p.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fs.sent) != 0 {
		t.Fatalf("关态不应推: %+v", fs.sent)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_price_alert`).Scan(&st); err != nil || st != "fired" {
		t.Fatalf("关态命中应 fired: %s", st)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='价格提醒' AND note LIKE '%推送关%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("关态落行注「推送关」应=1: %d", n)
	}
}
