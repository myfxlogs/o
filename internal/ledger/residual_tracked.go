// residual_tracked.go —— 构性残差豁免跟踪化（工单O-36327——业主令④
// 「豁免续命即失职」根修）：fin_known_residual 静态定额豁免对活值残差
// 恒滞后（DCD 口径差随 BTC 漂、Earn 应计逐日增生），每爬过豁口告警一次、
// 人工 +$1 续豁三连实证（#20-22）。resolver 列存解析键非死数——豁免额
// =无 resolver 静态行 + Σ resolver 行现算值。
//
// 失败语义三分（钦定）：
//   - resolver 未注册/读败 → fail-closed 回退静态 amount_usd 锚值
//     （同 run.go:189 读败让位判例——未知按旧行为不放大不缩小）；
//   - |跟踪−锚| > max(2×锚, $25) → 破带不按构性续：该行豁免贡献=0
//     （检测力不稀释——漂移超模型=可能有新成分）+按新族异常升件；
//   - 行集读败 → err 上抛，调用方 recon_probe_failed+本键让位。
package ledger

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// breachDedupHours 破带升件同窗去重（探针族 24h 同窗同钦）。
const breachDedupHours = 24

// breachBandFloor 破带下限——锚小额族跟踪噪声不触发（$25 钦定底）。
const breachBandFloor = 25.0

// exemptPart 豁免逐行明细（diff_brief 成分段+破带升件证据载体；
// O-36964：ID/Reason 供白话段静态行命名——机读段不消费新增字段）。
type exemptPart struct {
	ID       int64   // fin_known_residual 行 id（静态行白话名锚）
	Resolver string  // resolver 键原文（静态行=""）
	Reason   string  // 定性散文原文（静态行白话名料——截段消毒用）
	Anchor   float64 // amount_usd 列=锚值（静态行=豁免额本身）
	Raw      float64 // 解析现算值（静态/回退行=锚值镜像）
	Value    float64 // 实际计入豁免的额（破带=0）
	Mode     string  // static|tracked|fallback|breach
}

// residResolver 注册表函数形：resolver 键冒号参数段→现算豁免额（USD）。
// 读败/缺源/缺读件=err 上抛由调用方回退静态锚——resolver 内禁假零。
type residResolver func(ctx context.Context, r *Runner,
	venue, ccy string, params []string) (float64, error)

var residualResolvers = map[string]residResolver{
	"earn_accrual_gap": resolveEarnGap,
}

// trackedExempt 对拍豁免额（run.go:189 判据喂数+diff_brief 成分段同源）：
// 返回 (总豁免额, 逐行明细, err)。err=行集读败——调用方让位判例同前。
func (r *Runner) trackedExempt(ctx context.Context,
	venue, ccy, acct string) (float64, []exemptPart, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, amount_usd, coalesce(resolver,''), reason
		FROM fin_known_residual WHERE venue=$1 AND ccy=$2 AND expires_at>now()
		  AND coalesce(account_id,'')=$3
		ORDER BY id`, venue, ccy, acct)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var total float64
	var parts []exemptPart
	for rows.Next() {
		var id int64
		var anchor float64
		var res, reason string
		if err := rows.Scan(&id, &anchor, &res, &reason); err != nil {
			return 0, nil, err
		}
		if res == "" {
			total += anchor
			parts = append(parts, exemptPart{ID: id, Reason: reason,
				Anchor: anchor, Raw: anchor, Value: anchor, Mode: "static"})
			continue
		}
		segs := strings.Split(res, ":")
		// O-38064 S2：qty:<units> 币量豁免——币量空间先扣不经价（非本位键
		// 专属；USDT/USD 键出 qty 行=按现行 fallback 语义不改）。豁免额
		// 不计入 USD exempt（Value=0）但成分段照列——落进未注册 fallback
		// 会把 amount_usd 当 USD 豁免加进去=双计病灶。
		if segs[0] == "qty" && !isBaseCCY(ccy) {
			var u float64
			if len(segs) > 1 {
				u, _ = strconv.ParseFloat(segs[1], 64)
			}
			parts = append(parts, exemptPart{ID: id, Resolver: res, Reason: reason,
				Anchor: anchor, Raw: u, Value: 0, Mode: "qty"})
			continue
		}
		fn, ok := residualResolvers[segs[0]]
		var tracked float64
		var rerr error
		if ok {
			tracked, rerr = fn(ctx, r, venue, ccy, segs[1:])
		}
		if !ok || rerr != nil {
			// 未注册/读败→回退静态锚（fail-closed——未知不放大不缩小）。
			total += anchor
			parts = append(parts, exemptPart{ID: id, Resolver: res, Reason: reason,
				Anchor: anchor, Raw: anchor, Value: anchor, Mode: "fallback"})
			continue
		}
		// 诚实护栏：跟踪值越锚带→不按构性续（贡献 0）+新族异常升件。
		if math.Abs(tracked-anchor) > math.Max(2*anchor, breachBandFloor) {
			breachEscalate(ctx, r.pool, venue, ccy, res, tracked, anchor)
			parts = append(parts, exemptPart{ID: id, Resolver: res, Reason: reason,
				Anchor: anchor, Raw: tracked, Value: 0, Mode: "breach"})
			continue
		}
		total += tracked
		parts = append(parts, exemptPart{ID: id, Resolver: res, Reason: reason,
			Anchor: anchor, Raw: tracked, Value: tracked, Mode: "tracked"})
	}
	return total, parts, rows.Err()
}

// qtyExempt 币量豁免合计（O-38064 S2：resolver='qty:<units>' 未过期行
// Σunits——units=该 ccy 币量十进制串；amount_usd 只作立行时参考锚不参与
// 判定。畸形 units 行不计——fail-closed 向多报不向静默）。
func (r *Runner) qtyExempt(ctx context.Context, venue, ccy, acct string) (float64, error) {
	var u float64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(u),0) FROM (
		SELECT (split_part(resolver,':',2))::numeric u FROM fin_known_residual
		WHERE venue=$1 AND ccy=$2 AND expires_at>now()
		  AND coalesce(account_id,'')=$3
		  AND resolver LIKE 'qty:%'
		  AND split_part(resolver,':',2) ~ '^-?[0-9]+(\.[0-9]+)?$') t`,
		venue, ccy, acct).Scan(&u)
	return u, err
}

// resolveEarnGap earn_accrual_gap：Earn 应计时基差=应计累计（账本
// kind='interest' 已入账 accrual 总量——venue amt 面只认实现额，未实现
// 应计=恒差增生源，earn-accrual-vs-venue-read-residual 条目机制定案）。
// 已知盲区：利息实现入 amt（复利/派发/赎回息差）无现金流事件可读——
// 现值=累计应计上界，破带护栏兜实现后虚高。
func resolveEarnGap(ctx context.Context, r *Runner,
	venue, ccy string, _ []string) (float64, error) {
	var s float64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE venue=$1 AND ccy=$2 AND kind='interest'`, venue, ccy).Scan(&s); err != nil {
		return 0, fmt.Errorf("interest 应计累计读败: %w", err)
	}
	return s, nil
}

// breachEscalate 破带升件（新族异常——esc_class=对账 同域；scope=行级
// 键防互压；24h 同窗去重）。只报不修——豁免贡献已归零，告警链自然接管。
func breachEscalate(ctx context.Context, pool *pgxpool.Pool,
	venue, ccy, res string, tracked, anchor float64) {
	scope := fmt.Sprintf("residual_model_breach_%s_%s_%s", venue, ccy,
		strings.NewReplacer(":", "_", "/", "_").Replace(res))
	band := math.Max(2*anchor, breachBandFloor)
	note := fmt.Sprintf("构性豁免跟踪破带——resolver=%s 现算=$%.2f 锚=$%.2f(带=$%.2f)：跟踪模型与实测脱节按新族异常处置，不按构性续（检测力不稀释——漂移超模型=可能有新成分，定性归唤知会话）",
		res, tracked, anchor, band)
	_, _ = pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT '探测席','升件','',$1,'对账',$2
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND esc_class='对账' AND esc_scope=$2
			AND ts>now()-make_interval(hours=>$3))`,
		note, scope, breachDedupHours)
}

// exemptBrief 豁免成分段文本（diff_brief 复用——逐行 mode 定性：
// 跟踪行显现算/锚，破带行显现算归零，回退行显锚值兜底）。
func exemptBrief(parts []exemptPart) string {
	if len(parts) == 0 {
		return "豁免在账=0"
	}
	var b strings.Builder
	b.WriteString("豁免在账")
	for i, p := range parts {
		if i > 0 {
			b.WriteString("|")
		}
		switch p.Mode {
		case "qty":
			fmt.Fprintf(&b, "%s=币量豁免", p.Resolver)
		case "tracked":
			fmt.Fprintf(&b, "%s=现算$%.2f/锚$%.2f", p.Resolver, p.Value, p.Anchor)
		case "breach":
			fmt.Fprintf(&b, "%s=破带现算$%.2f/锚$%.2f→归零", p.Resolver, p.Raw, p.Anchor)
		case "fallback":
			fmt.Fprintf(&b, "%s=回退锚$%.2f", p.Resolver, p.Value)
		default:
			fmt.Fprintf(&b, "静态$%.2f", p.Value)
		}
	}
	return b.String()
}

// trackedExemptBrief diff_brief 使用的封装——与 trackedExempt 同径
// （破带升件 24h 幂等去重，判据拍+brief 双调安全）。
func (r *Runner) trackedExemptBrief(ctx context.Context,
	venue, ccy, acct string) (float64, string) {
	total, parts, err := r.trackedExempt(ctx, venue, ccy, acct)
	if err != nil {
		return 0, "豁免读败"
	}
	return total, exemptBrief(parts)
}
