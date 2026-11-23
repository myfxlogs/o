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
		!strings.Contains(txt, "停手已解除") {
		t.Fatalf("解除向文案不符: %q", txt)
	}
	txt, actable = HumanText("kill_switch", "kill_switch false→true")
	if !actable {
		t.Fatalf("触发向 actable 应=true（告警级）, got false: %q", txt)
	}
	if !strings.Contains(txt, "系统停手") || !strings.Contains(txt, "请关注") ||
		!strings.Contains(txt, "急停触发") {
		t.Fatalf("触发向文案不符: %q", txt)
	}
}

// ── 工单O-34997 业主面「发生」段人话化验收 ─────────────────────────

// 黑话词表（spec 钦定全列）——业主面任何段不得命中。
var jargonWords = []string{
	"venue读", "账本净", "容差", "已定性豁免", "簿记构成",
	"recon_adjust", "spot_sell", "spot_buy", "earn_in", "earn_out",
	"升件", "谓词", "口径差", "venue=", "esc_scope", "diff=",
	"payload", "fingerprint", "ops_event", "act=",
}

func assertNoJargon(t *testing.T, tag, txt string) {
	t.Helper()
	for _, w := range jargonWords {
		if strings.Contains(txt, w) {
			t.Fatalf("%s 黑话词 %q 命中: %q", tag, w, txt)
		}
	}
}

// actable kind 全配人话摘要（或显式兜底）——漏配=红。
func TestHumanTextActablePlainCoverage(t *testing.T) {
	for kind, spec := range kindSpec {
		if !spec.actable {
			continue
		}
		txt, _ := HumanText(kind, "机读诊断原文 venue读=1.0 账本净=2.0 容差=0.5")
		assertNoJargon(t, kind, txt)
		if !strings.Contains(txt, "发生：") {
			t.Fatalf("%s 缺发生段: %q", kind, txt)
		}
		occurred := strings.SplitN(txt, "发生：", 2)[1]
		occurred = strings.SplitN(occurred, "\n定性", 2)[0]
		if strings.Contains(occurred, "机读诊断原文") {
			t.Fatalf("%s 发生段漏出机读原文: %q", kind, occurred)
		}
	}
}

// 未知 kind 兜底人话——机读 payload 永不直出业主面。
func TestHumanTextUnknownKindFallback(t *testing.T) {
	txt, actable := HumanText("brand_new_kind_xyz",
		"venue读 9480.1008 vs 账本净 9544.6921 recon_adjust -11715.94")
	if actable {
		t.Fatalf("未知 kind 应降级非 actable")
	}
	assertNoJargon(t, "unknown-kind", txt)
	if !strings.Contains(txt, "系统发现一处需要关注的情况") {
		t.Fatalf("兜底文案不符: %q", txt)
	}
}

// ledger_diff_alert 示范口径——生产侧双载荷 plain 直给优先于 kindSpec
// 兜底；机读 payload 不进业主面。
func TestHumanTextLedgerDiffPlain(t *testing.T) {
	diag := "账实差 $64.59（容差 $0.50 +已定性豁免 $0.70；venue读 9480.1008 vs 账本净 9544.6921）→ 24h出入金在账: 入+$1655.94 出-$0.00→含外部移动嫌疑（业主可核）；72h簿记构成: recon_adjust $-11715.94 / withdraw $-10000.00 / spot_sell $9999.08"
	plain := "账上记的钱比交易所实际显示多 $64.59。初步判断：大概率是已知的计量口径不同，机器定性中——钱没丢，不用动。"
	txt, _ := HumanTextPlain("ledger_diff_alert", diag, plain)
	assertNoJargon(t, "ledger_diff", txt)
	if !strings.Contains(txt, plain) {
		t.Fatalf("显式 plain 未上发生段: %q", txt)
	}
	if strings.Contains(txt, "9480.1008") || strings.Contains(txt, "recon_adjust") {
		t.Fatalf("机读诊断漏出业主面: %q", txt)
	}
	// 无显式载荷回落 kindSpec.plain——仍人话不裸机读。
	txt2, _ := HumanText("ledger_diff_alert", diag)
	assertNoJargon(t, "ledger_diff-fallback", txt2)
	if strings.Contains(txt2, "9480.1008") {
		t.Fatalf("兜底形机读漏出: %q", txt2)
	}
}
