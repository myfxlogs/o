// drillshape.go —— 腿形注册拆件（模板→腿构法映射登记表面自 simdrill
// matrix.go 迁立，升件#15291 消费域收窄同 commit）：册语义不变——
// 新增执行族模板必先登记否则演习面 fail-closed 拒跑不静默漏。
// 迁立理由：闸凭据消费域收窄谓词 Drillable 归 lifecycle 消费，
// 而腿形册=演习面产能登记真源——册居 platform 格域件（excell 消费面
// 双侧同源律同形），lifecycle/simdrill 双边既有白名单边零新方向。
package excell

import "strings"

// LegShape 腿形（模板→腿构法——票面钦定映射登记表；新增执行族模板
// 必先登记此表否则矩阵 fail-closed 拒跑不静默漏）。
type LegShape string

const (
	ShapeOKXTwoLeg LegShape = "okx_two_leg" // 现货多+合约空（funding/basis 族）
	ShapeXvenue    LegShape = "xvenue"      // mt5 CFD 腿先+okx perp 腿后
	ShapeDCD       LegShape = "dcd"         // DCD 产品三步径（quote→trade→status）
)

// LegShapeOf 模板→腿形映射（注册表——execfam 册内模板全覆盖断言锚）。
func LegShapeOf(tmpl string) (LegShape, bool) {
	switch {
	case tmpl == "funding-hedge":
		return ShapeOKXTwoLeg, true
	case tmpl == "quarter_basis_long" || tmpl == "perp_quarter_spread" ||
		tmpl == "quarter_basis_short":
		return ShapeOKXTwoLeg, true
	case tmpl == "xvenue-carry":
		return ShapeXvenue, true
	case tmpl == "simdrill-gate":
		// 演习 harness 自体测试模板（单钻径通用双腿闸演——非产线模板，
		// Run 派路经 LegShapeOf 单源后须显式登记否则拒载）。
		return ShapeOKXTwoLeg, true
	case strings.HasPrefix(tmpl, "dcd_") || strings.HasPrefix(tmpl, "vol_sell"):
		// vol_sell=DCD 产品径（strangle 复合键 runDCD 内拆腿）——原误映
		// okx_two_leg 会拿产品键当现货/合约键跑错腿（T-0522 假想卡骨架批）。
		return ShapeDCD, true
	}
	return "", false
}

// Drillable 策略级演习凭据产能谓词（毕业闸消费域收窄锚——升件#15291，
// canon simdrill-gate-domain-coverage-mismatch）：LegShapeOf 映到
// Drill.Run 可产凭据腿形=真。ShapeXvenue 产能只在格级键域（matrix
// cell_key——Run 显式拒载），execDrillPass 的 strategy_ref 键域对之
// 无产能；消费侧先以本谓词拦否则凭据闸成 probe→armed_ttl 黑洞。
// 与 Run 派路同出 LegShapeOf 单源——谓词/派路禁两处自判防再裂。
func Drillable(tmpl string) bool {
	s, ok := LegShapeOf(tmpl)
	return ok && s != ShapeXvenue
}
