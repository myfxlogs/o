package hunger

import "testing"

// TestFaceBaselineWhitelistDashboardUI 工单#16664 补册实证——
// face:service:dashboard-ui（业主前端驾驶舱写席）入基线册后
// faceBaselineExempt 精确键豁免成立，理由键随新面行留账。
func TestFaceBaselineWhitelistDashboardUI(t *testing.T) {
	reason, ok := faceBaselineExempt("face:service:dashboard-ui")
	if !ok {
		t.Fatal("face:service:dashboard-ui 未入 faceBaselineWhitelist——补册失效")
	}
	if reason == "" {
		t.Fatal("face:service:dashboard-ui 册条理由键为空")
	}
}
