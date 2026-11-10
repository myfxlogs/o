package ledger

import (
	"strings"
	"testing"
)

// 业主令 2026-11-10：告警须答「钱从哪来」——出入金标识=外部移动 marker。
func TestLedgerDiffBrief(t *testing.T) {
	// 零出入金=簿记域（机器定性倾向,业主无需核）
	b := diffBriefText(0, 0, "spot_sell $50.00 / spot_buy -$49.50")
	if !strings.Contains(b, "无出入金") || !strings.Contains(b, "非外部移动") {
		t.Fatalf("零出入金未定性簿记域: %s", b)
	}
	if !strings.Contains(b, "spot_sell") {
		t.Fatalf("簿记构成缺席: %s", b)
	}
	// 出入金在账=外部移动嫌疑（明呈业主可核）
	b = diffBriefText(1200.5, 0, "deposit $1200.50")
	if !strings.Contains(b, "外部移动嫌疑") || !strings.Contains(b, "1200.50") {
		t.Fatalf("入金嫌疑未明呈: %s", b)
	}
	b = diffBriefText(0, 300, "withdraw -$300.00")
	if !strings.Contains(b, "出-$300.00") {
		t.Fatalf("出金标识缺席: %s", b)
	}
	// 零现金流行=构成兜底不空串
	b = diffBriefText(0, 0, "")
	if !strings.Contains(b, "近72h无现金流行") {
		t.Fatalf("空构成未兜底: %s", b)
	}
}
