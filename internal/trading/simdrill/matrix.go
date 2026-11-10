// matrix.go —— simvenue 全周期演习矩阵批跑机（T-0461——部署前置闸凭据机；
// 腿形径/仿真 venue 装配/DCD 径见 matrix_exec.go·matrix_legs.go·
// matrix_venue.go——行数棘轮拆件）。
//
// 格集=excell.Enumerate 唯一真源（机枚举禁手编）；每格每面值档跑
// 「开→持→强平→对账」全径——venue 全仿真（simvenue=OKX 域/mt5sim=MT5 CFD
// 域/dcdSim=DCD 产品域），消费件一律产线同码：execution.Runner 双腿原子
// （RouterExecutor 按腿 venue 分派）、mtexec.Executor（ticket/OrderClose/
// OpenedOrders 产线同径）、dcdexec.Open（产线 hedgeOpener.dcdOpen 抽立件
// 同径）、guard.Warden 哨戒平仓（venue 路由现读面 AttachVenueSizer）。
//
// 凭据落 fin_exec_drill（cell_key=excell.CellKey 唯一编码，strategy_ref=
// ephemeral 演习行——24h TTL 自退役同 T-0426）；任一格 fail=ops_event 升件
// （seat='演习' esc_class='执行径演习' 禁静默红）+批次汇总 Failed 计数
// （armed→open 格闸消费 latest verdict fail-closed）。
package simdrill

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/excell"
	"o/internal/risk"
)

// legShape 腿形注册（模板→腿构法——票面钦定映射登记表；新增执行族模板
// 必先登记此表否则矩阵 fail-closed 拒跑不静默漏）。
type legShape string

const (
	shapeOKXTwoLeg legShape = "okx_two_leg" // 现货多+合约空（funding/basis 族）
	shapeXvenue    legShape = "xvenue"      // mt5 CFD 腿先+okx perp 腿后
	shapeDCD       legShape = "dcd"         // DCD 产品三步径（quote→trade→status）
)

var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z0-9]+)-[0-9]{6}$`)

// legShapeOf 模板→腿形映射（注册表——execfam 册内模板全覆盖断言锚）。
func legShapeOf(tmpl string) (legShape, bool) {
	switch {
	case tmpl == "funding-hedge":
		return shapeOKXTwoLeg, true
	case tmpl == "quarter_basis_long" || tmpl == "perp_quarter_spread" ||
		tmpl == "quarter_basis_short":
		return shapeOKXTwoLeg, true
	case tmpl == "xvenue-carry":
		return shapeXvenue, true
	case strings.HasPrefix(tmpl, "dcd_") || strings.HasPrefix(tmpl, "vol_sell"):
		// vol_sell=DCD 产品径（strangle 复合键 runDCD 内拆腿）——原误映
		// okx_two_leg 会拿产品键当现货/合约键跑错腿（T-0522 假想卡骨架批）。
		return shapeDCD, true
	}
	return "", false
}

// Matrix 矩阵批跑机（venue 三面全仿真——每格新 venue 实例隔离残留对拍）。
type Matrix struct {
	Pool *pgxpool.Pool
	Cfg  *config.Client
	Gate *risk.Gate
}

// CellResult 单格单档演习结论（凭据行载体——cell_key 机读键唯一编码）。
type CellResult struct {
	CellKey     string
	Verdict     string // pass|fail
	Detail      string
	StrategyRef int64 // ephemeral 演习行（凭据归属锚）
	OpenExecID  int64
	CloseExecID int64
}

// MatrixResult 批跑汇总（任一格 fail=Failed>0——部署前置闸 fail-closed 判据）。
type MatrixResult struct {
	Results []CellResult
	Failed  int
}

// RulesVersionMatrix 矩阵凭据引擎指纹（simvenue/1+mt5sim/1+dcdsim/1 合成——
// 「哪套规则判的」审计锚；任一子件语义变更必翻本值）。
const RulesVersionMatrix = "simvenue-matrix/1"

// Run 矩阵全格全档遍历（excell 唯一真源——免演登记格跳过不产凭据）。
// 返回 error 仅限机械败（枚举/凭据落账败）；格病=verdict=fail+升件不炸批。
func (m *Matrix) Run(ctx context.Context) (MatrixResult, error) {
	sweepStaleEphemeral(ctx, m.Pool)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		return MatrixResult{}, err
	}
	out := MatrixResult{}
	for _, c := range cells {
		if c.Exempt {
			log.Printf("[simdrill-matrix] 免演登记跳过: %s（词表零占）", c.Template)
			continue
		}
		shape, ok := legShapeOf(c.Template)
		if !ok {
			// 占用格腿形未登记=矩阵面缺口——显式 fail 凭据不静默漏。
			out.Results = append(out.Results, CellResult{
				Verdict: "fail",
				CellKey: fmt.Sprintf("%s@%s/%s#unshaped", c.Template, c.VenuePair, c.Direction),
				Detail:  "腿形未登记（legShapeOf 无映射——矩阵面缺口）"})
			out.Failed++
			continue
		}
		for _, t := range c.Tiers {
			res := m.runCell(ctx, c, t, shape)
			out.Results = append(out.Results, res)
			if res.Verdict != "pass" {
				out.Failed++
			}
		}
	}
	return out, nil
}

// runCell 单格单档全径（开→持→强平→对账→凭据落账——格 fail 亦落凭据）。
func (m *Matrix) runCell(ctx context.Context, c excell.Cell, t excell.Tier, shape legShape) CellResult {
	res := CellResult{CellKey: excell.CellKey(c.Template, c.VenuePair, c.Direction, t.Notional),
		Verdict: "pass"}
	inst := t.InstID
	if inst == "" {
		inst = defaultInst()
	}
	res.Detail = fmt.Sprintf("shape=%s inst=%s", shape, inst)

	// 演习 ephemeral 策略行（同 T-0426——state='degraded' 哨戒降级联动必平
	// 径免判据链依赖；24h TTL 自退役——模板打 simdrill: 前缀不产占用格防自激）。
	venueTag := "okx"
	if shape == shapeXvenue {
		venueTag = excell.VenuePairXvenue
	}
	var stratID int64
	// template_ref 嵌 cell_key——fin_strategy_active_dedupe 唯一键
	// (venue,inst,direction,template_ref) 下同格多档/同 inst 异向行
	// 免撞（'simdrill%' 前缀=excell 占用排除+级联清理同律）。
	if err := m.Pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,direction,venue)
		VALUES($1,$2,'degraded',$3,$4,$5) RETURNING id`,
		inst, StratTemplate+":cell:"+res.CellKey, t.Notional, c.Direction, venueTag).Scan(&stratID); err != nil {
		res.Verdict, res.Detail = "fail", res.Detail+" 演习策略行建败: "+err.Error()
		return m.record(ctx, res, inst)
	}
	res.StrategyRef = stratID
	// 夭折防护同 Drill.Run——演习未平仓位即时关账防产线仓巡拾取。
	defer func() {
		_, _ = m.Pool.Exec(ctx,
			`UPDATE fin_position SET closed_at=now() WHERE strategy_ref=$1 AND closed_at IS NULL`, stratID)
		// T-0511：position 关账后残 'closing' 行=僵尸（探针假升件）——
		// 清扫径同关 guard_close。
		_, _ = m.Pool.Exec(ctx,
			`UPDATE fin_guard_close SET state='closed',closed_at=now(),updated_at=now(),
			 last_error='reconcile:drill_cleanup' WHERE state='closing'
			 AND position_id IN (SELECT id FROM fin_position WHERE strategy_ref=$1)`, stratID)
		retireEphemeral(ctx, m.Pool, stratID)
	}()

	if shape == shapeDCD {
		m.runDCD(ctx, &res, c, t, stratID, inst)
	} else {
		m.runLegged(ctx, &res, c, t, stratID, inst, shape)
	}
	return m.record(ctx, res, inst)
}

// record 凭据落账+病面升件（fail 亦落——闸可读败因；升件禁静默红；
// inst_id 列载本格代表乐器——cell_key 列载机读格键，语义不混）。
func (m *Matrix) record(ctx context.Context, res CellResult, inst string) CellResult {
	if _, err := m.Pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
		VALUES($1,$2,$3,$4,$5,$6)`,
		res.StrategyRef, inst, res.Verdict, RulesVersionMatrix, res.Detail, res.CellKey); err != nil {
		log.Printf("ERROR [simdrill-matrix] 凭据落账败（静默红风险） cell=%s: %v", res.CellKey, err)
	}
	if res.Verdict != "pass" {
		m.fail(ctx, res)
	}
	return res
}

// fail 病面收口：ops_event 升件行（seat='演习'——DM 消费锚同 T-0426）。
func (m *Matrix) fail(ctx context.Context, res CellResult) {
	if _, err := m.Pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,'fin')`,
		Seat, Ticket, fmt.Sprintf("矩阵演习 %s：%s", res.CellKey, res.Detail), EscClass); err != nil {
		log.Printf("ERROR [simdrill-matrix] 升件行落账败: %v", err)
	}
	log.Printf("[simdrill-matrix] %s fail：%s", res.CellKey, res.Detail)
}
