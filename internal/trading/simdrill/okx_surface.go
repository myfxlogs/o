package simdrill

import (
	_ "embed"
	"strings"
)

// okxSpotSurfaceTxt OKX 实有面现货在册快照（T-0482 夹具保真数据源——
// docs/能力账/okx-instruments-survey-2026-11-06.json SPOT state=live
// 1210 对 distilled 全量 sorted；快照冻结非活读，venue 面漂移归
// survey 重产另票）。
//
//go:embed okx_spot_surface.txt
var okxSpotSurfaceTxt string

var okxSpotSurface = func() map[string]struct{} {
	m := make(map[string]struct{}, 1300)
	for _, ln := range strings.Split(okxSpotSurfaceTxt, "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			m[ln] = struct{}{}
		}
	}
	return m
}()

// okxSpotLive OKX 实有面现货在册判——产线 loadMeta instruments 空 rows
// fail-closed 的夹具侧镜像（实有面无此对=derived spot 不登记，sim 腿构
// fail 机理对齐真径先拦不虚构乐器放腿到 venue 拒槽）。
func okxSpotLive(spotID string) bool {
	_, ok := okxSpotSurface[spotID]
	return ok
}
