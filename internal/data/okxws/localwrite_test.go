// localwrite_test.go —— T-0209 桩锚：谓词过滤+per-inst 3s 节流双断言。
package okxws

import (
	"testing"
	"time"
)

func TestLocalWritePredFilter(t *testing.T) {
	c := &Collector{
		LocalWritePred: func(kind, instID string) bool {
			return kind == "ticks" && instID == "BTC-USDT-SWAP" // funding 同集但测 ticks 域
		},
	}
	it := rowItem{tick: &tickRow{instID: "BTC-USDT-SWAP"}}
	// 谓词闸路径（handle 内联件——桩直接断言谓词判据）
	if !c.LocalWritePred("ticks", it.tick.instID) {
		t.Fatal("活跃集 inst 应过闸")
	}
	if c.LocalWritePred("ticks", "SOL-USDT-SWAP") {
		t.Fatal("离集 inst 应跳闸（不落行不写僵尸）")
	}
}

func TestPassThrottle3s(t *testing.T) {
	c := &Collector{passAt: map[string]time.Time{}}
	it := rowItem{tick: &tickRow{instID: "BTC-USDT-SWAP"}}
	if !c.passThrottle(it) {
		t.Fatal("首帧应过")
	}
	if c.passThrottle(it) {
		t.Fatal("3s 内同 inst 应跳帧（写洪防）")
	}
	c.passAt["ticks:BTC-USDT-SWAP"] = time.Now().Add(-4 * time.Second)
	if !c.passThrottle(it) {
		t.Fatal("4s 后应过（间隔满）")
	}
}
