// feed_esc.go —— ops_event 升件环（业主令「探针只上 UI 不落 ops_event=病灶
// 本体——后端对前端零信任」+「资金分配决策权整体下放 RSI/system」）：
// 60s 拍评六类信号面——
//
//	①鲜度死：ops_face 同源 healthComps 注册表 dead 态→升件 esc_class=断流
//	②部署缺口：净值/闸值/族需额对拍→升件 esc_class=资金闸
//	③歧态腿：fin_order_leg ambiguous 在案→升件 esc_class=对账
//	  （venue 回报歧态势=钱径事实,原径只推 Feishu 人端——机读面此处补）
//	④孤儿账：注册表册 pending 积压超 TTL=写了没人读→升件 esc_class=孤儿账
//	⑤唤知面死：未决升件在案而处置行 24h 缺席→升件 esc_class=唤知
//	  +Feishu 人端兜底（脑死时 ops_event 只是日志,人是最后兜底面）
//	⑥数据缺：fin_opp_eval unevaluable 同真因持续在案=结构缺
//	  →升件 esc_class=数据缺（把「安静等数据」变「缺口信号」）
//
// 转死=升件（esc_scope=键,24h 同窗去重防重启风暴）；复绿=恢复行携升件#id
// 自闭（唤知环 dedup 视引证=已处置,自愈案例零认知开销）。
// 对账/孤儿账/唤知/数据缺类 dedup=「未销才抑」——同窗有未销升件不暴,
// 销案后新事证重升（销案≠事实消失,新批再来不可误吞）。
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/platform/notify"
)

// escEvery 升件拍节（dead 阈最小 2min——60s 拍保证死不超拍级误报又不过频）。
const escEvery = 60 * time.Second

// escDedupHours 同窗去重——同键 24h 至多一报（进程重启后 DB 同窗
// 兜底内存面丢忆；覆盖失明探针同契）。
const escDedupHours = 24

// escLoop 常驻拍（o-core 装配环起——转死/复绿状态机在内存,
// 24h 同窗 dedup 在库）。ctx 终=进程退。al=人端兜底告警器
// （唤知面死用——nil=只落账不人推）。
func (c coreService) escLoop(ctx context.Context, al *notify.Alerter) {
	t := time.NewTicker(escEvery) // sched-exempt-ok：信号判定期界非业务轮询
	defer t.Stop()
	deadSince := map[string]time.Time{} // scope → 本次死起始（恢复行注死窗）
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		c.feedEscCheck(ctx, deadSince)
		c.deployEscCheck(ctx, deadSince)
		c.reconEscCheck(ctx, deadSince)
		c.orphanEscCheck(ctx, deadSince)
		c.cognitionEscCheck(ctx, deadSince, al)
		c.cognitionSpawnFailEscCheck(ctx, deadSince)
		c.auditStallEscCheck(ctx, deadSince)
		c.commercialEscCheck(ctx, deadSince)
		c.capabilityEscCheck(ctx, deadSince)
		c.dataGapEscCheck(ctx, deadSince)
		c.oppSurfaceStallEscCheck(ctx, deadSince)
		c.liveDudEscCheck(ctx, deadSince)
		c.stuckEscCheck(ctx, deadSince)
		c.lessonBacklogEscCheck(ctx, deadSince)
		c.pairZeroEscCheck(ctx, deadSince)
		c.intakeDryEscCheck(ctx, deadSince)
		c.navStaleEscCheck(ctx, deadSince)
		c.codeworkStallEscCheck(ctx, deadSince)
		c.expectProdEscCheck(ctx, deadSince) // 元探通用形——生产面期望产出 vs 实产差分(工单#10605 B)
		c.armedDistanceBeat(ctx)             // 量表探（非升件——armed 距阈日行照）
		c.ownerLedgerBeat(ctx)               // 量表探（业主令账+点破率曲线日行照）
		c.profitDailyBeat(ctx)               // 量表探（盈利报告日行照——业主唯一例行读件）
		c.selfStateBeat(ctx)                 // 量表探（自态账——会话冷启动即读件,工单#10358 C）
		c.booksBeat(ctx, deadSince)          // 账房（ΔNAV三桶对账+鬼账升件,业主令11-12立件）
		c.pnlAttrBeat(ctx, deadSince)        // 策略盈亏归因（业主令11-12①——分列+账房咬合,账房器官零改动）
	}
}

// feedEscCheck 鲜度面逐件判——面死转升件/复绿转自闭。
func (c coreService) feedEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	for _, b := range healthComps() {
		var ts *time.Time
		if err := c.pool.QueryRow(ctx, b.query).Scan(&ts); err != nil {
			continue // 读件败本拍让位——非死证据不误报
		}
		isDead := false
		if ts == nil {
			isDead = b.zeroNote == "" // 语义空态面（zeroNote 在案）零行非死
		} else {
			isDead = time.Since(*ts) > b.dead
		}
		c.escEdgeUV(ctx, deadSince, "断流", b.name, "@", isDead, func() string {
			lastTxt := "零行"
			if ts != nil {
				lastTxt = fmt.Sprintf("末鲜 %s（陈 %s）", ts.UTC().Format("01-02 15:04Z"), time.Since(*ts).Round(time.Second))
			}
			return fmt.Sprintf("面断流 %s：%s——超 dead 阈 %s。%s", b.name, lastTxt, b.dead, b.fix)
		})
	}
}

// deployEscCheck 部署面缺口探（业主令「资金分配归 RSI 决」进水口）：
// ①cap_hit——live+probe 族需额 ≥90% 敞口闸（闸到顶=事实非判词）
// ②idle_capital——需额 <25% 净值 ∧ live 族在册（资本闲置=事实非判词）
// 判词归唤知会话（runbook 资金闸档）——探针只述实数不带结论。
func (c coreService) deployEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	var eq *float64
	if err := c.pool.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' AND ts>now()-interval '48 hours'
		ORDER BY ts DESC LIMIT 1`).Scan(&eq); err != nil || eq == nil || *eq <= 0 {
		return // 净值读件缺/陈值=不判（禁假证据升件——NAV 日拍件陈>48h 与零行同语义，升件#9575）
	}
	var demand float64
	if err := c.pool.QueryRow(ctx, `SELECT coalesce(sum(notional_usd),0)
		FROM fin_strategy WHERE state IN ('live','probe')`).Scan(&demand); err != nil {
		return
	}
	var liveN int
	_ = c.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE state='live'`).Scan(&liveN)
	exp := c.cfgF("risk.max_exposure_usd")
	notional := c.cfgF("risk.max_notional_usd")
	util := fmt.Sprintf("%.0f", 100*demand/maxf(exp, 1))
	facts := fmt.Sprintf("净值=%.0f 需额=%.0f(live×%d+probe) 敞口闸=%.0f 单笔闸=%.0f 利用率=%s%%",
		*eq, demand, liveN, exp, notional, util)
	c.escEdgeUV(ctx, deadSince, "资金闸", "cap_hit", util,
		exp > 0 && demand >= 0.9*exp, func() string {
			return fmt.Sprintf("敞口闸到顶：%s——live/probe 族需额已贴闸（部署面事实，阈值判词归唤知会话）", facts)
		})
	c.escEdgeUV(ctx, deadSince, "资金闸", "idle_capital", util,
		liveN > 0 && eq != nil && demand < 0.25**eq, func() string {
			return fmt.Sprintf("资本闲置：%s——已证族在册而部署不足净值 1/4（部署面事实，是否真缺口归唤知会话裁）", facts)
		})
}

// cfgF 闸值浮点读（缺席/坏件=0——0 值天然不满足触发式,fail-safe）。
func (c coreService) cfgF(key string) float64 {
	v, err := c.cfg.GetFloat(key)
	if err != nil {
		return 0
	}
	return v
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// escEdgeU 未销才抑沿——死态期间每拍过 escalateUnresolved（其内 DB 判
// 未决:开着静默/销了重火/火完再静默）——dedup 粒=未决事件非「沿」。
// 2026-11-10 根修:原沿触发版 deadSince 键占死不再发——「定性要码修」
// 类销案（分类≠修好,条件仍真）后信号成黑洞唯重启才回流,与注释钦定的
// 「销案后再发=新事证重升」语义自相矛盾（歧态批无碍=销即修好=条件翻假;
// 能力债类销案=定性,债还在）。deadSince 仍记首死供恢复行陈窗。
// 2026-11-11 全探针归一本版:旧 escEdge 沿触发版(仅 feed/资金闸三调用点
// 在残)+escalate(同窗任意升件 dedup)删除——票抑复岗靠「销了重火」,
// 沿触发版在抑撤后永哑(死沿早被消费)。
func (c coreService) escEdgeU(ctx context.Context, deadSince map[string]time.Time,
	class, scope string, dead bool, note func() string) {
	c.escEdgeUV(ctx, deadSince, class, scope, "", dead, note)
}

// escEdgeUV 计量类边缘触发（值指纹版）：语义同 escEdgeU 而升件行
// ticket 载量值指纹，dedup 扩到「销案行」——本 scope 末行（开或销）
// 指纹同=同值已阅不火；异值=新现实重火。常设态信号（stall/dud/
// capital 族）由「每拍一火同判词 churn」改「一变一火」——
// 2026-11-11 业主令「常亮告警是状态不是事件」建模根修。
// fp 调用点自供量化串（探针保薄壳——指纹比对是机械运算零判词）；
// 事件类探针（每次发生=新事件）仍走 escEdgeU 传空 fp=旧语义。
func (c coreService) escEdgeUV(ctx context.Context, deadSince map[string]time.Time,
	class, scope, fp string, dead bool, note func() string) {
	key := class + "/" + scope
	if dead {
		if _, ok := deadSince[key]; !ok {
			deadSince[key] = time.Now()
		}
		if fp == "@" {
			fp = deadSince[key].UTC().Format("0601021504") // 二元死态类：回合起刻=指纹（同回合销后不火,复绿再死=新回合照火）
		}
		c.escalateUnresolved(ctx, class, scope, fp, note())
		return
	}
	if since, ok := deadSince[key]; ok {
		delete(deadSince, key)
		c.recoverNote(ctx, class, scope, since)
	}
}

// escalateUnresolved 未销才抑升件（对账类 dedup）——同窗 24h 内存未销升件
// （无引证行）才 dedup;已销案（处置/恢复行引 升件#id）后再发=新事证重升。
// 歧态批销案后新批腿再来不可被同窗误吞——dedup 粒=「未决事件」非「时间窗」。
func (c coreService) escalateUnresolved(ctx context.Context, class, scope, fp, note string) {
	if fp != "" {
		// 值指纹闸（计量类）：本 scope 30d 内末行升件（开或销）note 前缀
		// 指纹同=同值已阅不火；30d 窗=指纹记忆期，超龄同值再火=定期复核
		// 点名防静默烂账。指纹载 note 前缀不载 ticket（ticket=工单流
		// 分组键,face 面有消费）——读败不抑（fail-open 升件优先）。
		var lastID int64
		var last string
		if err := c.pool.QueryRow(ctx, `SELECT e.id, e.note FROM ops_event e
			WHERE e.event='升件' AND e.esc_class=$1 AND e.esc_scope=$2
			AND e.ts>now()-interval '30 days'
			ORDER BY e.id DESC LIMIT 1`, class, scope).
			Scan(&lastID, &last); err == nil && strings.HasPrefix(last, "v="+fp+" ") {
			// 销票破纹：末行升件后存在本 scope 票已终（非挂起销票∨
			// 挂票过期）——票终而条件存=抑撤复验，指纹失效照火
			// （销票复岗安全网不破——TestTicketScopeSuppress 契约）。
			var tickEnded bool
			if err := c.pool.QueryRow(ctx, `SELECT EXISTS(
				SELECT 1 FROM ops_event t
				WHERE t.event='工单' AND t.id>$1 AND t.ts>now()-interval '30 days'
					AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$2||'([^a-zA-Z0-9_]|$)')
					AND (EXISTS (SELECT 1 FROM ops_event d
							WHERE d.event='处置' AND d.id>t.id
								AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
								AND d.note NOT LIKE '%挂起条件=%')
						OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)
								*interval '1 hour' < now()
							AND EXISTS (SELECT 1 FROM ops_event d
								WHERE d.event='处置' AND d.id>t.id
									AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
									AND d.note LIKE '%挂起条件=%'))))`,
				lastID, scope).Scan(&tickEnded); err == nil && !tickEnded {
				return
			}
		}
		note = "v=" + fp + " " + note
	}
	var open int
	// x 先 LIKE 剪非引用行再 regex（SG CPU 烧实证——无剪=逐 x 行
	// 全价 regex 求值,引件行远少于全表）。
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='升件' AND e.esc_class=$1 AND e.esc_scope=$2
		AND e.ts>now()-make_interval(hours=>$3)
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note LIKE '%升件#%'
			AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
				OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
					WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))`,
		class, scope, escDedupHours).Scan(&open); err != nil || open > 0 {
		return // 读败静默下拍再试；未销在案不暴
	}
	c.escInsert(ctx, class, scope, note)
}

// escInsert 升件落账行（escalateUnresolved 末段）。
// 票抑:同 scope 有开工单(无处置引「工单#id」∨挂起在期限内)静默让位——
// 持续条件转票后探针不再重火同一问题,票自载期限管催(挂票亦抑:
// 「票开着但不该干」≠「没人管」,过期抑撤自然复验)。销票/挂票过期
// =抑撤,条件还在探针自然复岗(escalateUnresolved 未销才抑=抑撤即重火)。
func (c coreService) escInsert(ctx context.Context, class, scope, note string) {
	var tick int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		scope).Scan(&tick); err == nil && tick > 0 {
		return // 开工单在案——票管催,探针静默(读败不抑=fail-open升件优先)
	}
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','',$1,$2,$3)`,
		note, class, scope); err != nil {
		log.Printf("[esc] 升件写败 %s/%s: %v", class, scope, err)
		return
	}
	log.Printf("[esc] 升件 %s/%s——%s", class, scope, note)
}

// recoverNote 复绿恢复行：引 升件#<最近同窗未销 id> 自闭。
func (c coreService) recoverNote(ctx context.Context, class, scope string, since time.Time) {
	var id int64
	if err := c.pool.QueryRow(ctx, `SELECT e.id FROM ops_event e
		WHERE e.event='升件' AND e.esc_class=$1 AND e.esc_scope=$2
		AND e.ts>now()-make_interval(hours=>$3)
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note LIKE '%升件#%'
			AND (x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')
				OR EXISTS (SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
					WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint)))
		ORDER BY e.id DESC LIMIT 1`,
		class, scope, escDedupHours).Scan(&id); err != nil {
		return // 无在案升件=静默
	}
	note := fmt.Sprintf("升件#%d 复绿——%s/%s 恢复（死窗 %s 内自愈/重启/处置；处置席按 repair/配置账定性销账）",
		id, class, scope, time.Since(since).Round(time.Second))
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','恢复','',$1)`, note); err != nil {
		log.Printf("[esc] 恢复写败 %s/%s: %v", class, scope, err)
	}
}
