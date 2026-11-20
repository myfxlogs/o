// shapegate.go —— 亏算封顶形态闸消费面（工单#26436——判例#1器官级物化：
// ops_event seat=自裁 event=判例 ticket=okxgrid-shape 在案——本金线内偏好
// 亏算封顶工具形（现货网格/套利/earn 停泊：亏≤投入+带内浮亏，无爆仓无
// 资金费拖磨）；杠杆不禁但逐案过仓位判据保证金面——杠杆把潜在亏损推出
// 本金总额线外=越线归业主物理前提）。
//
// 消费位双闸：①admitProposals intake（提案→paper 开户——execfam.Has/
// legset.InstFitOK 同槽）；②openLive live 拉起（live 策略→Opener 缝真钱
// 出闸前最后判）。判词唯一真源=execfam.ShapeGateVerdict；本金总额线=
// md_equity_curve NAV 现值（48h 鲜度窗同 guard.dirline/death_grad 口径，
// 本金读缺/陈=0→判据缺 fail-closed——断供非越线，nav_stale 探另报）；
// 判据参数=lifecycle.shape_gate.* cfg 键（缺/坏值=码内钦定缺省 lev=1/
// loss_frac=1.0——保守满额保证金+全损假设，valdomain 在册）。
//
// 硬拒升件记案：demand>本金线=判例「越线归业主物理前提」——ops_event
// 升件行 esc_class=资金闸 esc_scope=shape_gate_principal，note 前缀 24h
// 去重（hungerEscalate 同构——同根因不暴）；判据缺拦不升件（数据/配置
// 面病灶非资本越线——如实留 proposed/live 滞留待参数面自解）。
package lifecycle

import (
	"context"
	"fmt"
	"log"

	"o/internal/platform/execfam"
)

// shapeGateSeat/shapeGateTicket 升件行署名（lifecycle 器官自报行——
// hungerEscalate/minerEscalate 同席「探测席」先例）。
const (
	shapeGateSeat   = "探测席"
	shapeGateTicket = ""
)

// shapeGateNav 本金总额线读面（md_equity_curve NAV 现值——48h 鲜度窗同
// guard.dirline 口径；读败/无行/非正=0 返——判据缺 fail-closed 由判词
// 件裁，断供病灶归 nav_stale 探不在本闸职）。
func (r *Runner) shapeGateNav(ctx context.Context) float64 {
	var nav float64
	if err := r.pool.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' AND ts>now()-interval '48 hours'
		ORDER BY ts DESC LIMIT 1`).Scan(&nav); err != nil {
		return 0
	}
	return nav
}

// shapeGateParams 保证金面审判据参数（lev_assumed=期望保证金按名义/杠杆
// 折算分母；loss_frac=极端情形潜在亏损占名义比——钦定缺省 1.0/1.0=保守
// 满额保证金+全损假设；cfg 显式非正值不采纳回缺省——armed_ttl_h 同判）。
func (r *Runner) shapeGateParams() (lev, lossFrac float64) {
	lev, lossFrac = 1.0, 1.0
	if v, err := r.cfg.GetFloat("lifecycle.shape_gate.lev_assumed"); err == nil && v > 0 {
		lev = v
	}
	if v, err := r.cfg.GetFloat("lifecycle.shape_gate.loss_frac"); err == nil && v > 0 {
		lossFrac = v
	}
	return
}

// shapeGateReview 形态闸判词缝（bounded 直过先于 NAV 读——免审族零 DB
// 成本；nav 懒读件穿参，bounded/判据路径只在确需本金时触发一次读）。
// 返回 (放行, 判词, 越本金线归业主标)。
func (r *Runner) shapeGateReview(ctx context.Context, tmpl string, notional float64, nav *shapeGateNavLazy) (bool, string, bool) {
	if execfam.ShapeBoundedOnly(tmpl) {
		return true, fmt.Sprintf("族=%s bounded 单标直过（判例#1——亏算封顶免保证金面审）", tmpl), false
	}
	lev, lossFrac := r.shapeGateParams()
	return execfam.ShapeGateVerdict(tmpl, notional, nav.get(ctx, r), lev, lossFrac)
}

// shapeGateEsc 越本金线硬拒升件记案（判例「越线归业主物理前提」——提案/
// 策略滞留待业主处置非静默吞；note 前缀同窗 24h 至多一行防每拍爆量——
// hungerEscalate 同构去重；写败=记案径自报末环静默退）。
func (r *Runner) shapeGateEsc(ctx context.Context, ref, tmpl string, notional, nav float64, why string) {
	prefix := fmt.Sprintf("shape_gate %s %s", ref, tmpl)
	if _, err := r.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'资金闸','shape_gate_principal'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND esc_scope='shape_gate_principal'
			AND note LIKE $4 AND ts>now()-interval '24 hours')`,
		shapeGateSeat, shapeGateTicket,
		fmt.Sprintf("%s: %s（notional=%.2f 本金线=%.2f）——杠杆潜在亏损越线归业主物理前提，滞留待处置", prefix, why, notional, nav),
		prefix+"%"); err != nil {
		log.Printf("[lifecycle] shape_gate 升件行落败（记案径自报末环静默退）: %v", err)
	}
}

// shapeGateNavLazy 本拍本金线懒读缓存（bounded 直过先于 NAV 读——全册
// bounded 拍零 DB 成本；读败缓存 0 复用判词不重复打库）。
type shapeGateNavLazy struct {
	nav  float64
	read bool
}

func (l *shapeGateNavLazy) get(ctx context.Context, r *Runner) float64 {
	if !l.read {
		l.nav = r.shapeGateNav(ctx)
		l.read = true
	}
	return l.nav
}

// shapeGateBlock 判形态闸一件（ref=提案#/策略# 归因锚）：拦=升件（越线
// 归业主标）+判词回 caller 落拒因。nav 懒读件穿参——bounded 直过零读。
func (r *Runner) shapeGateBlock(ctx context.Context, ref, tmpl string, notional float64, nav *shapeGateNavLazy) (bool, string) {
	ok, why, ownerEsc := r.shapeGateReview(ctx, tmpl, notional, nav)
	if !ok && ownerEsc {
		r.shapeGateEsc(ctx, ref, tmpl, notional, nav.nav, why)
	}
	return ok, why
}
