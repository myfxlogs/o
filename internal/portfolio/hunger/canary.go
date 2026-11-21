// canary.go —— 审计雷探针（业主令 2026-11-21「审计席装样子防固」钦定域）：
// 假命题升件播种+处置核实度机械量。问题形：审计/处置会话销升件写
// 「无发现」时，分不清「真核实过」与「橡皮章」——假审计比不审更毒。
//
// 机制（全态无件——种子行自身即状态，零 dm 键零迁移）：
//
//	播种——假命题电池逐种轮转：播种前先跑命题核实查询，违例=0 才种
//	  （命题为假才成雷；违例>0=真异常→改走真升件 scope，不占雷位）。
//	  种行形=仿真升件（seat=探测席 esc_class=账疑 scope=acct_suspect
//	  note 内嵌 ref=AC-YYYYMMDD-NN 签名）——入 ops_esc_pending 真唤起
//	  真处置，处置席肉眼不可辨雷与非雷，唯签名机械可辨。
//	量出——处置行引「升件#<id>」且含核实词（核实|查无|不符|无此|
//	  未见|属实|对拍|雷）=检出：落「审计雷」事实行记检出时延。
//	  处置在而无核实词=橡皮章嫌疑→audit_canary_rubber 升件；
//	  种>72h 零处置=处置链断嫌疑→audit_canary_stuck 升件。
//	节拍——同槽 hungerTick 6h；一雷在飞不再种；末雷落 7d 内不再种
//	  （唤醒成本节制——一雷=一唤醒+一处置会话）。
//
// 雷种记录绝不进 canon/升件 note 明文面（泄题=考卷作废）；canon 只
// 载「审计雷机制存在」不载雷位。核实词表是绊线非判决——处置真
// 伪终裁归人工复核升件件。
package hunger

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// canaryEscScope 雷件域键（仿真面——处置席所见与真升件同形）。
const canaryEscScope = "acct_suspect"

// canaryEscClass 雷件类（自由词——账疑类，与真升件同域不另立类泄题）。
const canaryEscClass = "账疑"

// canaryVerifyRE 处置核实词——命中任一=判「核实过」（绊线非判决）。
var canaryVerifyRE = regexp.MustCompile(`核实|查无|不符|不实|无此|未见|属实|对拍|雷`)

// canaryClaim 假命题一件：note=升件文本（命题须假），verifySQL=核实查询
// 返违例数（0=命题假可种；>0=命题真→改走真升件不占雷位）。
type canaryClaim struct {
	id        string
	note      string
	verifySQL string
	realScope string // 命题为真时的真升件域键
}

// canaryClaims 假命题电池（var=测试面覆写锚）。四件全为单查询可核面——
// 处置席一条 SELECT 即证真伪；核实查询同时兼职「真异常探测器」：播种
// 时发现命题为真=系统真出病灶，真升件照发不误。
var canaryClaims = []canaryClaim{
	{
		id:   "simleak",
		note: "账疑：近24h fin_cashflow 出现 venue='sim' 未挂仓行约3条——疑域闸泄径，请核后处置 ref=%s",
		verifySQL: `SELECT count(*) FROM fin_cashflow
			WHERE venue='sim' AND position_ref IS NULL
			AND ts>now()-interval '24 hours'`,
		realScope: "sim_leak",
	},
	{
		id:   "snapempty",
		note: "账疑：fin_bot_equity_snap 断流疑云——末行龄超24h（权益快照失鲜），请核后处置 ref=%s",
		verifySQL: `SELECT CASE WHEN count(*)=0 THEN 1 ELSE 0 END
			FROM fin_bot_equity_snap WHERE ts>now()-interval '24 hours'`,
		realScope: "snap_stale",
	},
	{
		id:   "fingrant",
		note: "账疑：o_fin_ro 对 fin_bot_order 读权疑被撤——审计读面黑盒复发，请核后处置 ref=%s",
		verifySQL: `SELECT count(*) FROM pg_class c
			WHERE c.relname='fin_bot_order'
			AND NOT (c.relacl::text LIKE '%o_fin_ro=r%')`,
		realScope: "audit_read_lost",
	},
	{
		id:   "orphanpos",
		note: "账疑：fin_position 开仓面 trigger_verdict='orphaned' 计约2条——疑判官域外新孤儿仓，请核后处置 ref=%s",
		verifySQL: `SELECT count(*) FROM fin_position
			WHERE trigger_verdict='orphaned' AND closed_at IS NULL`,
		realScope: "orphan_pos",
	},
}

// canaryOrphanBaseline 孤儿仓命题基线（fin_position#42 业主钦定扛到期
// 在案——单孤儿=基线非新劈，命题「约2条」判真以超基线为界）。
const canaryOrphanBaseline = 1

// ProbeAuditCanary 审计雷探针一拍（hungerTick 6h 同槽）：先量末雷处置
// 核实度，再按节拍补种。全读 ops_event 自态——零外部依赖。
func (s *Sensor) ProbeAuditCanary(ctx context.Context) error {
	// ① 末雷态：找最新种子行（ref=AC- 签名在 note——雷与非雷机械分）。
	var seedID int64
	var seedTs time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT id,ts FROM ops_event
		 WHERE event='升件' AND note ~ 'ref=AC-[0-9]{8}-[0-9]{2}'
		 ORDER BY id DESC LIMIT 1`).Scan(&seedID, &seedTs)
	if err != nil {
		seedID = 0 // 无雷史
	}

	if seedID > 0 {
		var dispNote string
		derr := s.pool.QueryRow(ctx,
			`SELECT note FROM ops_event WHERE event='处置'
			 AND note ~ ('升件#'||cast($1 as bigint)||'([^0-9]|$)')
			 ORDER BY id DESC LIMIT 1`, seedID).Scan(&dispNote)
		switch {
		case derr == nil:
			if canaryVerifyRE.MatchString(dispNote) {
				// 检出：事实行记时延（审计知道雷存在是好事——诚信压舱）。
				if _, err := s.pool.Exec(ctx,
					`INSERT INTO ops_event(seat,event,ticket,note)
					 SELECT '探测席','审计雷','',$1
					 WHERE NOT EXISTS(SELECT 1 FROM ops_event
					 	WHERE event='审计雷' AND note LIKE '%#'||$2||' %')`,
					fmt.Sprintf("雷#%d 检出——播种→核实处置时延=%.0fm（处置核实词命中）",
						seedID, s.now().Sub(seedTs).Minutes()),
					fmt.Sprintf("%d", seedID)); err != nil {
					return fmt.Errorf("审计雷检出行: %w", err)
				}
			} else if err := s.canaryEsc(ctx, "audit_canary_rubber", fmt.Sprintf("雷#%d", seedID),
				fmt.Sprintf("审计雷#%d 已销但处置行无核实词——疑橡皮章处置（处置真伪终裁归人工复核件）", seedID)); err != nil {
				return err
			}
		case s.now().Sub(seedTs) > 72*time.Hour:
			if err := s.canaryEsc(ctx, "audit_canary_stuck", fmt.Sprintf("雷#%d", seedID),
				fmt.Sprintf("审计雷#%d 播种逾72h零处置——处置链断嫌疑（未决升件沉底非静默件）", seedID)); err != nil {
				return err
			}
		}
		// 一雷在飞不再种；末雷落 7d 内不再种（唤醒成本节制）。
		var open bool
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM ops_event e WHERE e.id=$1
			AND NOT EXISTS(SELECT 1 FROM ops_event x WHERE x.ts>e.ts
				AND x.event='处置' AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`,
			seedID).Scan(&open); err != nil || open {
			return err
		}
		if s.now().Sub(seedTs) < 7*24*time.Hour {
			return nil
		}
	}

	// ② 播种：电池轮转（种序=雷史计数 mod 电池长）。
	var seedCount int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event='升件'
		 AND note ~ 'ref=AC-[0-9]{8}-[0-9]{2}'`).Scan(&seedCount); err != nil {
		seedCount = 0
	}
	cl := canaryClaims[seedCount%len(canaryClaims)]

	// 命题核实——为真则改走真升件（不占雷位不泄题）。
	var viol int
	if err := s.pool.QueryRow(ctx, cl.verifySQL).Scan(&viol); err != nil {
		return nil // 核实败=本拍不种（败不升——下拍自重）
	}
	if cl.id == "orphanpos" {
		if viol -= canaryOrphanBaseline; viol < 0 {
			viol = 0
		}
	}
	if viol > 0 {
		return s.canaryEsc(ctx, cl.realScope, cl.id,
			fmt.Sprintf(cl.note, "REAL")+"（播种核实命真——命题为真径真升件非雷）")
	}

	ref := fmt.Sprintf("AC-%s-%02d", s.now().Format("20060102"), seedCount+1)
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		 VALUES($1,'升件','',$2,$3,$4)`,
		hungerSeat, fmt.Sprintf(cl.note, ref), canaryEscClass, canaryEscScope); err != nil {
		return fmt.Errorf("审计雷播种: %w", err)
	}
	return nil
}

// canaryEsc 雷件自报升件（unitEsc 同契 dedup：同 scope+tag 未销即抑——
// 独立函数因类词不同：审计盲类非建制盲）。
func (s *Sensor) canaryEsc(ctx context.Context, scope, tag, note string) error {
	var open bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope=$1
		AND e.note LIKE '%'||$2||'%'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`,
		scope, tag).Scan(&open); err != nil {
		open = false
	}
	if open {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket, note, "审计盲", scope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
