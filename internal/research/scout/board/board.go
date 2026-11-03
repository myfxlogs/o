// Package board —— scout 榜单行共享型（T-0312 返修①丙案：notify 读面直引
// 同 module 共享——metrics jsonb 字段漂移编译期炸非运行时静默降级；
// 单类型叶子件零依赖解 notify→scout 环）。
package board

// Board 榜单行（venue×族×top-N 中位净 APR×30d 持续占比——metrics jsonb
// 写回 schema；scout.metrics boards[] 逐字段对拍）。
type Board struct {
	Family    string  `json:"family"`
	NetAprMid float64 `json:"net_apr_mid"` // 30d 逐日净 APR 序列中位
	Persist   float64 `json:"persist"`     // 逐日净 APR>baseline 天数占比（30d）
	Baseline  float64 `json:"baseline"`
	BaseSrc   string  `json:"baseline_source"`
	TopN      int     `json:"top_n"`
}
