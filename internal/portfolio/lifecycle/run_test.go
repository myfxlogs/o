package lifecycle

import "testing"

// TestFundingGateTemplateDomain T-0202 桩锚：dcd_call 跳 funding 闸
// （模板域错配=全谱 live 零开仓根治）；funding-hedge 系照判。
func TestFundingGateTemplateDomain(t *testing.T) {
	if fundingHedgeFamily["dcd_call"] {
		t.Fatal("dcd_call 不入 funding 闸域")
	}
	if !fundingHedgeFamily["funding-hedge"] || !fundingHedgeFamily["xvenue-carry"] {
		t.Fatal("funding-hedge/xvenue-carry 入闸域")
	}
}
