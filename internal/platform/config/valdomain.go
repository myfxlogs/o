// valdomain.go —— 高危配置键值域闸（工单#25801/破口 B4：gate.kill/风控阈类
// 高危键值域零校验——错形/越界值落册即静默改写钱径行为）。值域=读器语义
// 的写侧镜像：typed.go 读器本就收原生 JSON 与字符串编码双形（GetBool 收
// "true"/"false"、GetFloat 收数字串——生产在册键实证：
// notify.*.enabled=string "false"、invest.lifecycle.live_switch=string
// "false"），故值域双形皆收、界值在解串后核。未列键=值域开放（validScalar
// 标量守照旧）；域规则命中的键错形/越界即拒——fail-closed。
package config

import (
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// 值域类（与键域 DomainDM/DomainOwner 正交——键域管「谁可写」，
// 值域管「写什么形/什么界」）。
const (
	valBool = "bool" // 布尔域：bool ∨ "true"/"false" 串形（读器双形镜像）
	valNum  = "num"  // 数值域：float64 ∨ 数字串形 + [lo,hi] 闭界
	valFrac = "frac" // 分数域：num 特化 [0,1]（*_pct 名实不符在案——
	// risk.dir.*_pct/lifecycle.dir.*_frac 在册值皆 0~1 分数非百分）
	valPos    = "pos"    // 正数域：num 特化 (0,∞)
	valNonNeg = "nonneg" // 非负数域：num 特化 [0,∞)（0=回缺省约定在案——
	// guard.funding_max_age_min v>0 方生效，0 非病灶）
)

// valRule 值域规则行（path.Match 通配，序敏感——先中先判）。
type valRule struct {
	glob string
	kind string
}

// valueDomainRules 高危键值域册。覆盖面=三族：①停机/闸类布尔键
// （错形=闸静默失效或误停）②风控阈/资金界数值键（越界=超限裸奔或饿死）
// ③生命周期分数键（>1=判据失真）。业主面键同受闸——值域闸在写径不分
// 席位（DMSet/Set 皆过），owner 键的写径本就在业主面、值域仍兜底。
var valueDomainRules = []valRule{
	// —— 布尔域：停机/闸/开关类 ——
	{"gate.kill.*", valBool},
	{"gate.enabled", valBool},
	{"*.enabled", valBool},
	{"*_enabled", valBool},
	{"*.halted", valBool},
	{"*.kill_switch", valBool},
	{"risk.kill_switch", valBool},
	{"exec.xvenue_open", valBool},
	{"invest.lifecycle.live_switch", valBool},
	{"invest.paper.*_enabled", valBool},
	{"deploy.*_enabled", valBool},
	{"cmc.ws_disabled", valBool},
	// —— 资金/阈界数值域 ——
	{"risk.max_notional_usd", valPos},
	{"ledger.recon_tol_usd", valNonNeg},
	{"earn.residual_clear_min_usd", valNonNeg},
	{"net.budget_kbps", valNonNeg},
	{"risk.*_pct", valFrac}, // 方向书外阈同分数实（risk.max_venue_pct=0.35 在册）
	{"trx.poll_seconds", valPos},
	{"annualized.deploy_floor_usd", valPos}, // 部署本金年化判据下限（O-34963——非正=判据恒部署口径=小分母病灶复辟口）
	{"*notional_usd", valPos},
	{"*_min_usd", valNonNeg},
	{"*_bps", valNonNeg},
	{"okxws.kline_census_hours", valNonNeg},
	{"okxws.kline_topn", valNonNeg},
	{"okxws.tick_scan_s", valPos},
	{"*.scan_timeout_s", valPos},
	{"guard.funding_max_age_min", valNonNeg},
	{"guard.max_hold_*", valNonNeg},
	{"exec.*_timeout_*", valPos},
	{"exec.*_cooldown_*", valNonNeg},
	{"lifecycle.*_ttl_*", valNonNeg},
	{"lifecycle.shape_gate.*", valPos}, // #26436 判例#1形态闸判据参数——lev_assumed/loss_frac 非正写入即拒（读器 v>0 采纳口径镜像）
	{"strategy.*_ttl_*", valNonNeg},
	// —— 分数域 [0,1]：pct 名分数实（在册值实证）——
	{"risk.dir.*_pct", valFrac},
	{"risk.dir.*_frac", valFrac},
	{"lifecycle.dir.*_frac", valFrac},
	{"lifecycle.dir.*_pct", valFrac},
	{"*.dir.stop_frac", valFrac},
	{"*.dir.take_frac", valFrac},
	{"lifecycle.death_factor", valPos},
	// —— 百分域 [0,100]：真百分键 ——
	{"trx.*_pct", numRange(0, 100)}, // floor_pct=8/vote_*_pct=0.1·1.0 在册皆界内
	{"invest.lifecycle.*_pct", numRange(0, 100)},
	{"invest.paper.*_pct", numRange(0, 100)},
	{"guard.*_pct", numRange(0, 100)},
	{"research.*_pct", numRange(0, 100)},
	{"deploy.*_pct", numRange(0, 100)},
}

// numRange 数值域规则构造（闭界 [lo,hi]；kind 串内嵌界值免扩类型面）。
func numRange(lo, hi float64) string {
	return fmt.Sprintf("num:%g:%g", lo, hi)
}

// CheckValueDomain 值域判定导出面（闸外写径共用——review/gatefx 等直写
// SQL 件照过本闸）。未中规则=nil（开放域）；中规则错形/越界=error。
func CheckValueDomain(key string, value json.RawMessage) error {
	kind := ""
	for _, r := range valueDomainRules {
		if m, _ := path.Match(r.glob, key); m {
			kind = r.kind
			break
		}
	}
	if kind == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal(value, &v); err != nil {
		return fmt.Errorf("非法 JSON: %w", err)
	}
	switch {
	case kind == valBool:
		switch t := v.(type) {
		case bool:
			return nil
		case string:
			if t == "true" || t == "false" {
				return nil // 读器双形镜像——串形布尔在册实证
			}
		}
		return fmt.Errorf("布尔域外（%s——bool 或 \"true\"/\"false\" 串形外拒）", string(value))
	case kind == valNum || strings.HasPrefix(kind, "num:"):
		f, ok := numValue(v)
		if !ok {
			return fmt.Errorf("数值域外（%s——数字或数字串形外拒）", string(value))
		}
		return numBound(kind, f)
	case kind == valFrac:
		f, ok := numValue(v)
		if !ok {
			return fmt.Errorf("分数域外（%s——数字或数字串形外拒）", string(value))
		}
		if f < 0 || f > 1 {
			return fmt.Errorf("分数越界（%v 出 [0,1]——名 pct 实 frac 在案）", f)
		}
		return nil
	case kind == valPos:
		f, ok := numValue(v)
		if !ok {
			return fmt.Errorf("正数域外（%s——数字或数字串形外拒）", string(value))
		}
		if f <= 0 {
			return fmt.Errorf("正数域外（%v≤0——(0,∞) 开界拒）", f)
		}
		return nil
	case kind == valNonNeg:
		f, ok := numValue(v)
		if !ok {
			return fmt.Errorf("非负数域外（%s——数字或数字串形外拒）", string(value))
		}
		if f < 0 {
			return fmt.Errorf("非负数域外（%v<0——[0,∞) 界拒，0=回缺省约定在案）", f)
		}
		return nil
	}
	return nil
}

// numBound num:lo:hi 闭界核（kind 编码见 numRange）。
func numBound(kind string, f float64) error {
	if kind == valNum {
		return nil // 纯数值域——无形界
	}
	var lo, hi float64
	if n, err := fmt.Sscanf(kind, "num:%g:%g", &lo, &hi); err != nil || n != 2 {
		return fmt.Errorf("值域册坏行（%q——numRange 构造外拒）", kind)
	}
	if f < lo || f > hi {
		return fmt.Errorf("数值越界（%v 出 [%v,%v]）", f, lo, hi)
	}
	return nil
}

// numValue 数值双形解（float64 原生 ∨ 数字串——GetFloat 读器镜像）。
func numValue(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}
