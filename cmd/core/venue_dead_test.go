package main

// venue_dead_test.go —— 工单#26999 锚：venue 结构死分类。三断言组：
// ①navBy 对 venueDeadMark 腿不数 miss（死腿剔除照写 NAV=其余腿和）；
// ②普通读败 note 行照 miss 照挡不动；③mt5Live 注 ErrVenueDead→卡面
// 读件缺行显形不动（note 载死因标）+nav() 不拦；IsVenueDead 判定表。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"o/internal/gen/dashboardv1"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func TestNavByVenueDeadLeg(t *testing.T) {
	row := func(venue, section, label, note string, amt float64) *dashboardv1.MoneyRow {
		return &dashboardv1.MoneyRow{Venue: venue, Section: section, Label: label,
			Note: note, Ccy: "USD", Amount: amt}
	}
	// ①仅死腿缺（note 载 venueDeadMark）→不拦，NAV=其余腿和。
	o := &dashboardv1.OverviewResponse{Money: []*dashboardv1.MoneyRow{
		row("okx", "交易户", "USDT", "", 10),
		row("mt5", "户", "读件缺", "读件缺·"+venueDeadMark, 0),
	}}
	o.NavTotal = 10
	by, err := navBy(o)
	if err != nil {
		t.Fatalf("死腿不应拦本拍: %v", err)
	}
	if by["USD"] != 10 {
		t.Fatalf("死腿剔除后 NAV 应=其余腿和 10, got %+v", by)
	}
	// ②普通读败（无死因标）→照 miss 照挡。
	o2 := &dashboardv1.OverviewResponse{Money: []*dashboardv1.MoneyRow{
		row("okx", "交易户", "USDT", "", 10),
		row("mt5", "户", "读件缺", "读件缺", 0),
	}}
	if _, err := navBy(o2); err == nil || !strings.Contains(err.Error(), "户/读件缺") {
		t.Fatalf("暂态读败应照拦: %v", err)
	}
	// ③死腿+暂态腿并存→暂态腿仍拦（死腿豁免不连带）。
	o3 := &dashboardv1.OverviewResponse{Money: []*dashboardv1.MoneyRow{
		row("okx", "交易户", "USDT", "", 10),
		row("mt5", "户", "读件缺", "读件缺·"+venueDeadMark, 0),
		row("okx", "资金户", "读件缺", "读件缺", 0),
	}}
	if _, err := navBy(o3); err == nil || !strings.Contains(err.Error(), "资金户") {
		t.Fatalf("暂态腿应仍拦: %v", err)
	}
}

func TestIsVenueDead(t *testing.T) {
	if IsVenueDead(nil) {
		t.Fatal("nil 非死")
	}
	if !IsVenueDead(venueDeadWrap(errors.New("ConnectEx err: OLD_VERSION"))) {
		t.Fatal("哨兵包装应判死")
	}
	if !IsVenueDead(errors.New("阶段=connect: ConnectEx err: OLD_VERSION(1010)")) {
		t.Fatal("裸 OLD_VERSION 串应判死（旧径兜底）")
	}
	if IsVenueDead(errors.New("dial timeout")) {
		t.Fatal("暂态读败非死")
	}
}

// ④装配面：mt5Live 注 ErrVenueDead→Overview 卡面 mt5/户 读件缺行
// 显形不动（note 载死因标）+nav() 闭包不拦本拍。
func TestMt5DeadLegCardAndNav(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(ts,channel,data)
		VALUES(now(),'account','{"details":[{"ccy":"USDT","eqUsd":"10","eq":"10"}]}')`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	okMap := func(v float64) func(context.Context) (map[string]float64, error) {
		return func(context.Context) (map[string]float64, error) {
			return map[string]float64{"USDT": v}, nil
		}
	}
	dcdOK := func(context.Context) ([]okxexec.DcdOrder, error) { return nil, nil }
	mt5Dead := func(context.Context) ([]mt5Acct, error) {
		return nil, venueDeadWrap(errors.New("ConnectEx err: OLD_VERSION"))
	}
	svc, navFn := assemble(pool, cfg, okMap(5), okMap(7), mt5Dead, dcdOK, nil, nil, nil)
	o, err := svc.Overview(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var deadRow *dashboardv1.MoneyRow
	for _, m := range o.Money {
		if m.GetVenue() == "mt5" && m.GetSection() == "户" {
			deadRow = m
		}
	}
	if deadRow == nil || !strings.Contains(deadRow.GetNote(), venueDeadMark) {
		t.Fatalf("mt5 死腿显形行缺/死因标缺: %+v", deadRow)
	}
	// nav 闭包=死腿剔除照写——funding5+earn7+交易10=USDT22（mt5 死腿零计）。
	by, err := navFn(ctx)
	if err != nil {
		t.Fatalf("死腿不应拦 NAV 本拍: %v", err)
	}
	if by["USDT"] != 22 {
		t.Fatalf("NAV 应=其余腿和 22, got %+v", by)
	}
	// 对照：暂态读败腿→nav() 仍拦（闸义不削）。
	mt5Down := func(context.Context) ([]mt5Acct, error) {
		return nil, errors.New("dial timeout")
	}
	_, navFn2 := assemble(pool, cfg, okMap(5), okMap(7), mt5Down, dcdOK, nil, nil, nil)
	if _, err := navFn2(ctx); err == nil || !strings.Contains(err.Error(), "户/读件缺") {
		t.Fatalf("暂态读败应照拦本拍: %v", err)
	}
}
