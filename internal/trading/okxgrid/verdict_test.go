package okxgrid

import "testing"

func vc() verdictCfg {
	return verdictCfg{
		stopFrac: 0.15, stopPatienceH: 72, stopWarmupH: 24,
		scaleMinAgeH: 168, scaleAprMin: 0.20, scaleFloatMax: 0.05,
		scaleSize: 100, aggCap: 500, maxParallel: 3, harvestMin: 5, enabled: true,
	}
}

// 判官纯判定面——先红桩纪律：判平须热身+窗内快照全穿;判加须龄+APR+浮界三全。
func TestJudgeStop(t *testing.T) {
	c := vc()
	// 热身内不判（新单浮亏天然）
	if k, _ := c.judge(10, snapRow{invest: 100, totalP: -20}, 5, 5); k != vHold {
		t.Fatalf("热身内误判平: %v", k)
	}
	// 窗内非全穿不判（波动误杀防）
	if k, _ := c.judge(30, snapRow{invest: 100, totalP: -20}, 4, 5); k != vHold {
		t.Fatalf("非全穿误判平: %v", k)
	}
	// 样本不足不判
	if k, _ := c.judge(30, snapRow{invest: 100, totalP: -20}, 2, 2); k != vHold {
		t.Fatalf("样本不足误判平: %v", k)
	}
	// 龄足+全穿+亏穿线=判平
	if k, why := c.judge(30, snapRow{invest: 100, totalP: -16}, 6, 6); k != vStop {
		t.Fatalf("应判平: %v %s", k, why)
	}
	// 亏不穿线不判（-14% < -15%线）
	if k, _ := c.judge(30, snapRow{invest: 100, totalP: -14}, 6, 6); k != vHold {
		t.Fatalf("未穿线误判平: %v", k)
	}
}

func TestJudgeScale(t *testing.T) {
	c := vc()
	// 龄不足不判加
	if k, _ := c.judge(100, snapRow{invest: 100, gridP: 10, floatP: 0}, 0, 0); k != vHold {
		t.Fatalf("龄不足误判加: %v", k)
	}
	// APR 不达不判加:0.4/100/200h*24*365=17.5%<20%
	if k, _ := c.judge(200, snapRow{invest: 100, gridP: 0.4, floatP: 0}, 0, 0); k != vHold {
		t.Fatalf("APR 不达误判加: %v", k)
	}
	// 全过判加:gridP=2/100/(200h→0.0228y)=87.7%≥20%,float=0≥-5
	if k, why := c.judge(200, snapRow{invest: 100, gridP: 2, floatP: 0}, 0, 0); k != vScale {
		t.Fatalf("应判加: %v %s", k, why)
	}
	// 浮亏穿界不判加
	if k, _ := c.judge(200, snapRow{invest: 100, gridP: 2, floatP: -6}, 0, 0); k != vHold {
		t.Fatalf("浮亏穿界误判加: %v", k)
	}
}
