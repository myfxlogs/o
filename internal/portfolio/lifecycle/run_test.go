package lifecycle

import (
	"testing"

	"o/internal/platform/execfam"
)

// TestFundingGateTemplateDomain T-0202 桩锚：dcd_call 跳 funding 闸
// （模板域错配=全谱 live 零开仓根治）；funding-hedge 系照判。
// T-0392：族属判据=execfam.FundingHedge 同源（本地册已删）。
func TestFundingGateTemplateDomain(t *testing.T) {
	if execfam.FundingHedge("dcd_call") {
		t.Fatal("dcd_call 不入 funding 闸域")
	}
	if !execfam.FundingHedge("funding-hedge") || !execfam.FundingHedge("xvenue-carry") {
		t.Fatal("funding-hedge/xvenue-carry 入闸域")
	}
}
