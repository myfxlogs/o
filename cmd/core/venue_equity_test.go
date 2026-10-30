package main

import (
	"context"
	"fmt"
	"testing"
)

// T-0264 验收锚①——dd 硬线权益基准=riskGate 同源三分量（okxVenueEq 单口径，
// 桩测非 skip）：停泊 Earn>0 且交易户≈0 时 dd 基准=总量非零（不误
// peak<=0→dd_no_base 硬线空转）。
func TestVenueEqParkedEarnNonZero(t *testing.T) {
	ctx := context.Background()
	eq, err := okxVenueEq(ctx,
		func(context.Context) (float64, error) { return 0, nil }, // 交易户≈0：闲资全停泊
		func(_ context.Context, m map[[2]string]float64) error {
			m[[2]string{"okx", "USDT"}] += 5200 // funding+Earn 停泊分量
			return nil
		})
	if err != nil {
		t.Fatalf("三分量读败: %v", err)
	}
	if eq <= 0 {
		t.Fatalf("停泊态 dd 基准=%v（peak<=0 硬线空转病灶）", eq)
	}
}

// 户间划转（trading↔funding）dd 面净值恒稳——不假回撤/假增长误触全平+落闸。
func TestVenueEqTransferStable(t *testing.T) {
	ctx := context.Background()
	snap := func(trading, fund float64) float64 {
		eq, err := okxVenueEq(ctx,
			func(context.Context) (float64, error) { return trading, nil },
			func(_ context.Context, m map[[2]string]float64) error {
				m[[2]string{"okx", "USDT"}] += fund
				return nil
			})
		if err != nil {
			t.Fatalf("读败: %v", err)
		}
		return eq
	}
	before := snap(6000, 400) // 划转前：交易户 6000 + funding/Earn 400
	after := snap(5100, 1300) // 900 自 trading 划入 funding 后
	if before != after {
		t.Fatalf("户间划转 dd 净值漂移 %v→%v（假回撤病灶）", before, after)
	}
}

// T-0264 钦定界②——读败语义对齐 T-0058：任一分量读败=err 上抛 fail-closed。
func TestVenueEqFailClosed(t *testing.T) {
	ctx := context.Background()
	boom := fmt.Errorf("venue 读败")
	if _, err := okxVenueEq(ctx,
		func(context.Context) (float64, error) { return 0, boom },
		func(context.Context, map[[2]string]float64) error { return nil }); err == nil {
		t.Fatal("交易户读败未上抛")
	}
	if _, err := okxVenueEq(ctx,
		func(context.Context) (float64, error) { return 100, nil },
		func(context.Context, map[[2]string]float64) error { return boom }); err == nil {
		t.Fatal("funding/Earn 读败未上抛")
	}
}
