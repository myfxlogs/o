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

// DcdOrderView venue dcd 持仓单投影（okxexec.DcdOrder 装配层翻译——
// ledger 不识 venue 方言，同 bal/mtm/earnings 注入缝先例；nil=读件
// 缺席 resolver fail-closed 回退静态锚）。
type DcdOrderView struct {
	OrdID       string
	State       string
	NotionalCcy string
	NotionalSz  string
}

// residResolver 注册表函数形：resolver 键冒号参数段→现算豁免额（USD）。
// 读败/缺源/缺读件=err 上抛由调用方回退静态锚——resolver 内禁假零。
type residResolver func(ctx context.Context, r *Runner,
	venue, ccy string, params []string) (float64, error)

var residualResolvers = map[string]residResolver{
	"dcd_mtm_drift":    resolveDCDDrift,
	"earn_accrual_gap": resolveEarnGap,
}

// trackedExempt 对拍豁免额（run.go:189 判据喂数+diff_brief 成分段同源）：
// 返回 (总豁免额, 逐行明细, err)。err=行集读败——调用方让位判例同前。
func (r *Runner) trackedExempt(ctx context.Context,
	venue, ccy string) (float64, []exemptPart, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, amount_usd, coalesce(resolver,''), reason
		FROM fin_known_residual WHERE venue=$1 AND ccy=$2 AND expires_at>now()
		ORDER BY id`, venue, ccy)
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

// resolveDCDDrift dcd_mtm_drift：DCD 口径差=venue 产品名义量与账本基线
// 凭据的币量差×标记现价。键形 dcd_mtm_drift:<ordId>——venue 侧名义量
// 走注入读件 r.dcdOrd（okcli.DcdOrders 60s 缓存在案同源——应计收益使
// venue 名义量随时间增厚，基线凭据冻结=口径差增生源；升件#34545 定性
// 「venue真含DCD应计，基线不随应计刷属已知域」）。币量差折 USD=跟踪
// 豁免额——随 BTC 价漂移（mark 现算）满足「现值随市漂移」验收口径。
// 基线缺席/单不在名义在持态/币种不配/读件缺席/读败/无价=err→回退
// 静态锚。近似边界诚实标注：venue 产品 USDT-MTM 真值（fin_position_
// view/venue_bill 径）归 OCI/唤知面，本解析器量的是名义量差域——
// 破带护栏兜模型失真。
func resolveDCDDrift(ctx context.Context, r *Runner,
	venue, ccy string, params []string) (float64, error) {
	if len(params) < 1 || params[0] == "" {
		return 0, fmt.Errorf("dcd_mtm_drift 参数缺 ordId")
	}
	ordID := params[0]
	var amt float64
	var bccy string
	err := r.pool.QueryRow(ctx, `SELECT (intent->>'amt')::numeric, intent->>'ccy'
		FROM fin_risk_decision r
		WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'=$1
		  AND intent->>'venue'=$2
		  AND NOT EXISTS (SELECT 1 FROM fin_risk_decision s
		    WHERE s.intent->>'op'='dcd_settle' AND s.intent->>'ordId'=r.intent->>'ordId')
		ORDER BY id DESC LIMIT 1`, ordID, venue).Scan(&amt, &bccy)
	if err != nil {
		return 0, fmt.Errorf("dcd 基线凭据读败 ordId=%s: %w", ordID, err)
	}
	if r.dcdOrd == nil {
		return 0, fmt.Errorf("dcd venue 读件未挂接 ordId=%s", ordID)
	}
	ords, err := r.dcdOrd(ctx)
	if err != nil {
		return 0, fmt.Errorf("dcd venue 读败 ordId=%s: %w", ordID, err)
	}
	var vsz string
	var vccy, state string
	found := false
	for _, od := range ords {
		if od.OrdID != ordID {
			continue
		}
		found = true
		vsz, vccy, state = od.NotionalSz, od.NotionalCcy, od.State
	}
	if !found {
		return 0, fmt.Errorf("dcd venue 单缺席 ordId=%s", ordID)
	}
	// 名义在持三态才计（restlive addOrd 同集——settled/redeemed 走回款径）。
	switch {
	case strings.EqualFold(state, "live"),
		strings.EqualFold(state, "pending_settle"),
		strings.EqualFold(state, "pending_redeem"):
	default:
		return 0, fmt.Errorf("dcd 单非名义在持态 ordId=%s state=%s", ordID, state)
	}
	if !strings.EqualFold(vccy, bccy) {
		return 0, fmt.Errorf("dcd 名义币种不配 ordId=%s venue=%s 基线=%s",
			ordID, vccy, bccy)
	}
	vamt, perr := strconv.ParseFloat(vsz, 64)
	if perr != nil || vamt < 0 {
		return 0, fmt.Errorf("dcd venue 名义量解析败 ordId=%s sz=%q", ordID, vsz)
	}
	gap := math.Abs(vamt - amt)
	if bccy == "USDT" || bccy == "USD" {
		return gap, nil
	}
	px, perr := markPxUSDT(ctx, r.pool, bccy)
	if perr != nil {
		return 0, perr
	}
	return gap * px, nil
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

// markPxUSDT 非 USDT 币→USDT 现价（run.go:202 判据折算同源——md_ticks
// last∪md_mark_index px 末行；无价=err 上抛回退静态锚）。
func markPxUSDT(ctx context.Context, pool *pgxpool.Pool, ccy string) (float64, error) {
	var px float64
	if err := pool.QueryRow(ctx, `SELECT v FROM (SELECT last v,ts FROM md_ticks
		WHERE inst_id=$1||'-USDT' AND venue<>'sim' UNION ALL SELECT px,ts FROM md_mark_index
		WHERE inst_id=$1||'-USDT') q ORDER BY ts DESC LIMIT 1`, ccy).Scan(&px); err != nil {
		return 0, fmt.Errorf("%s-USDT 现价读败: %w", ccy, err)
	}
	if px <= 0 {
		return 0, fmt.Errorf("%s-USDT 现价非正 %.4f", ccy, px)
	}
	return px, nil
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
	venue, ccy string) (float64, string) {
	total, parts, err := r.trackedExempt(ctx, venue, ccy)
	if err != nil {
		return 0, "豁免读败"
	}
	return total, exemptBrief(parts)
}
