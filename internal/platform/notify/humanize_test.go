package notify

import (
	"strings"
	"testing"
)

// 首性审计整改验收：kill_switch 同 kind 双向——解除向=通报(无需操作),
// 触发向=告警(请关注)。锚 2026-11-21 业主实证:复位通知被「请关注」
// 语气读成待办急件。
func TestHumanTextKillSwitchDirection(t *testing.T) {
	txt, actable := HumanText("kill_switch", "kill_switch true→false")
	if actable {
		t.Fatalf("解除向 actable 应=false（通报级）, got true: %q", txt)
	}
	if !strings.Contains(txt, "急停解除") || !strings.Contains(txt, "无需操作") ||
		!strings.Contains(txt, "系统由停手转为放行") {
		t.Fatalf("解除向文案不符: %q", txt)
	}
	txt, actable = HumanText("kill_switch", "kill_switch false→true")
	if !actable {
		t.Fatalf("触发向 actable 应=true（告警级）, got false: %q", txt)
	}
	if !strings.Contains(txt, "系统停手") || !strings.Contains(txt, "请关注") ||
		!strings.Contains(txt, "系统由放行转为停手") {
		t.Fatalf("触发向文案不符: %q", txt)
	}
}
