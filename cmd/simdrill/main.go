// simdrill —— 执行径演习 CLI（T-0426——纸面跑 venue 语义全径）。
//
// 对指定策略跑「开→平→对账」全径演习（同型同名义，simvenue 合成 venue 注入
// IO 缝——消费件全产线同码），verdict 落 fin_exec_drill（probe→live 毕业闸
// 判据件——缺/败 fail-closed）。乐器面直取 fixture 册（T-0427 品种主档前同径）。
//
// 用法：DB_DSN=... simdrill -strategy=42
//
// 产线常驻节拍另议不钦定（票面⑤）——本件=离线/手动拍入口：DM 对 probe 候选
// 策略逐个演习，绿凭据落账后毕业闸自解。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"o/internal/platform/config"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/simdrill"
	"o/internal/trading/simvenue"
)

func main() {
	strategy := flag.Int64("strategy", 0, "被演习策略 id（fin_strategy.id，必填）")
	flag.Parse()
	if *strategy <= 0 {
		fmt.Fprintln(os.Stderr, "用法: simdrill -strategy=<fin_strategy.id>")
		os.Exit(2)
	}
	if err := run(*strategy); err != nil {
		log.Fatalf("simdrill FAIL: %v", err)
	}
}

func run(strategyID int64) error {
	ctx := context.Background()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return fmt.Errorf("DB_DSN 未设（唯一 bootstrap env——缺件 fail-closed）")
	}
	pool, err := pgstore.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		return err
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		return fmt.Errorf("config 首载: %w", err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1e9, nil })
	d := &simdrill.Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: fixtureVenue()}
	res, err := d.Run(ctx, strategyID)
	if err != nil {
		return err
	}
	fmt.Printf("演习 #%d %s：verdict=%s\n  trigger=%s\n  detail=%s\n",
		res.StrategyRef, res.InstID, res.Verdict, res.Trigger, res.Detail)
	if res.Verdict != "pass" {
		os.Exit(1)
	}
	return nil
}

// fixtureVenue 演习乐器册（fixture 值——产线径=T-0427 品种主档供后换源）。
func fixtureVenue() *simvenue.SimVenue {
	return simvenue.New().
		WithInstrument(simvenue.Instrument{InstID: "CORE-USDT", InstType: "SPOT",
			LotSz: 0.01, MinSz: 0.01, TickSz: 0.0001, TakerFee: 0.001, Px: 0.0125}).
		WithInstrument(simvenue.Instrument{InstID: "CORE-USDT-SWAP", InstType: "SWAP",
			LotSz: 0.01, MinSz: 0.01, TickSz: 0.0001, CtVal: 100, CtValCcy: "CORE",
			TakerFee: 0.0005, Px: 0.0125}).
		WithBalance("USDT", 1000)
}
