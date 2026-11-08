// simdrill —— 执行径演习 CLI（T-0426 单策略径 + T-0461 矩阵批跑径）。
//
// 单策略径：对指定策略跑「开→平→对账」全径演习（同型同名义，simvenue 合成
// venue 注入 IO 缝——消费件全产线同码），verdict 落 fin_exec_drill
// （probe→live 毕业闸判据件——缺/败 fail-closed）。
//
// 矩阵径（-matrix）：excell.Enumerate 机枚举全格×面值档，逐格跑
// 「开→持→强平→对账」全周期（venue 三面全仿真——simvenue/mt5sim/dcdSim，
// 消费件产线同码），凭据落 fin_exec_drill.cell_key——T-0145 部署前置闸
// （armed→open 格凭据闸）消费判据；任一格 fail=批次非零退出+ops_event 升件。
//
// 用法：DB_DSN=... simdrill -strategy=42 | simdrill -matrix
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/simdrill"
	"o/internal/trading/simvenue"
)

func main() {
	strategy := flag.Int64("strategy", 0, "被演习策略 id（fin_strategy.id——单策略径）")
	matrix := flag.Bool("matrix", false, "矩阵批跑径（excell 全格×面值档——T-0461 部署前置闸凭据机）")
	flag.Parse()
	if *matrix {
		if err := runMatrix(); err != nil {
			log.Fatalf("simdrill matrix FAIL: %v", err)
		}
		return
	}
	if *strategy <= 0 {
		fmt.Fprintln(os.Stderr, "用法: simdrill -strategy=<fin_strategy.id> | simdrill -matrix")
		os.Exit(2)
	}
	if err := run(*strategy); err != nil {
		log.Fatalf("simdrill FAIL: %v", err)
	}
}

func poolAndGate() (*pgxpool.Pool, *config.Client, *risk.Gate, error) {
	ctx := context.Background()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, nil, nil, fmt.Errorf("DB_DSN 未设（唯一 bootstrap env——缺件 fail-closed）")
	}
	pool, err := pgstore.Connect(ctx, dsn)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		pool.Close()
		return nil, nil, nil, err
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("config 首载: %w", err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1e9, nil })
	return pool, cfg, gate, nil
}

func run(strategyID int64) error {
	pool, cfg, gate, err := poolAndGate()
	if err != nil {
		return err
	}
	defer pool.Close()
	d := &simdrill.Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: fixtureVenue()}
	res, err := d.Run(context.Background(), strategyID)
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

// runMatrix 矩阵批跑径（T-0461——excell 全格×面值档全周期演习；任一格
// fail=非零退出 fail-closed——凭据已逐格落账 cell_key 供闸消费）。
func runMatrix() error {
	pool, cfg, gate, err := poolAndGate()
	if err != nil {
		return err
	}
	defer pool.Close()
	m := &simdrill.Matrix{Pool: pool, Cfg: cfg, Gate: gate}
	res, err := m.Run(context.Background())
	if err != nil {
		return err
	}
	for _, r := range res.Results {
		fmt.Printf("矩阵格 %s：verdict=%s\n  detail=%s\n", r.CellKey, r.Verdict, r.Detail)
	}
	fmt.Printf("矩阵批跑毕：格档=%d fail=%d\n", len(res.Results), res.Failed)
	if res.Failed > 0 {
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
