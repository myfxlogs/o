package band

import (
	"reflect"
	"testing"
)

func TestFamiliesStayInsideTwoThresholdBands(t *testing.T) {
	wantNear := []string{"quarter_basis_long", "perp_quarter_spread", "dcd_call", "dcd_put"}
	var gotNear []string
	for family := range NearCertain {
		gotNear = append(gotNear, family)
	}
	if len(gotNear) != len(wantNear) {
		t.Fatalf("near-certain whitelist changed: %v", gotNear)
	}
	for _, family := range wantNear {
		if !NearCertain[family] {
			t.Errorf("required near-certain family missing: %s", family)
		}
	}
	if len(FamilyKeys) != 7 {
		t.Fatalf("family threshold registry must enumerate all seven families: %v", FamilyKeys)
	}
	values := map[string]float64{
		"research.funding.min_net_apr": 0.06,
		"research.basis.min_net_apr":   0.04,
		"research.dcd.min_apr":         0.04,
		"research.volsell.min_apr":     0.06,
	}
	get := func(key string) (float64, error) { return values[key], nil }
	if got := Audit(get); len(got) != 0 {
		t.Fatalf("valid two-band config reported drift: %v", got)
	}
	values["research.volsell.min_apr"] = 0.04
	wantBad := []string{"vol_sell_single_call", "vol_sell_strangle"}
	if got := Audit(get); !reflect.DeepEqual(got, wantBad) {
		t.Fatalf("directional vol-sell must not enter middle band: got %v want %v", got, wantBad)
	}
}
