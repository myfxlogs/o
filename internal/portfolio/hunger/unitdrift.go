// unitdrift.go —— 建制单元差分探针（工单#26444——o-self.timer disable 静默
// 躺尸 2.5d 实证坑根修：宿主建制单元注册表 canon+期望 vs 实际差分即账）。
// hunger 探针族第六实例（watcher-heartbeat #25803 模子同契）：
// 判一·单元死活——本机查件 unit_watch.sh 每拍 systemctl is-enabled/
//
//	is-active 实查，差分/复常才落「单态」行（seat=建制机）；探针侧
//	取各单元末行态 vs 注册表期望再判（注册表随时变→判定以探针拍时
//	canon 为准非行内 want 段）→ 异=unit_state_drift 升件。
//
// 判二·产出鲜度——注册表行钉产出行签名（prod_seat+prod_event+
//
//	prod_period_min）→ ops_event 末行龄>3×period=<unit>_heartbeat_stale
//	——单元活着不等于拍子在产（service 连败面）。
//
// 查件死活——ops_headless beat 龄>3×period_sec=unit_watch_dead（查件
//
//	死=全盲一等事件非静默）。
//
// 注册表缺/读败/空——fail-closed 升件 unit_registry_missing 不静默。
// 抑制同契（closeoutprobe 同款）：同 scope+unit 未销升件在案即抑、同
// scope 开工单在案即抑；查败 fail-open。只报不修——复跳归处置席。
package hunger

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// unitDriftEscClass 升件类（判类册自由词——建制盲类）。
	unitDriftEscClass = "建制盲"
	// unitDriftEscScope 单元态差分域键（票面钦定）。
	unitDriftEscScope = "unit_state_drift"
	// unitWatchDeadScope 查件死域键（beat 陈=查件死=全盲）。
	unitWatchDeadScope = "unit_watch_dead"
	// unitRegistryMissScope 注册表缺域键（fail-closed 不静默）。
	unitRegistryMissScope = "unit_registry_missing"
)

// unitReg 注册表单元行（ops.unit_registry JSON——expect_*=期望字面，
// prod_*=产出行签名可选三件套）。
type unitReg struct {
	Unit          string `json:"unit"`
	ExpectEnabled string `json:"expect_enabled"`
	ExpectActive  string `json:"expect_active"`
	ProdSeat      string `json:"prod_seat"`
	ProdEvent     string `json:"prod_event"`
	ProdPeriodMin int    `json:"prod_period_min"`
}

// unitRegistry 注册表全形（host=models 面；SG/OCI 面同键扩展位在案）。
type unitRegistry struct {
	Host  string `json:"host"`
	Watch struct {
		PeriodSec  int    `json:"period_sec"`
		BeatTicket string `json:"beat_ticket"`
		FactSeat   string `json:"fact_seat"`
		FactEvent  string `json:"fact_event"`
	} `json:"watch"`
	Units []unitReg `json:"units"`
}

// unitFactRE 单态行解析（「unit=X enabled=E active=A」——active 可为空）。
var unitFactRE = regexp.MustCompile(`enabled=([^ ]+) active=([^ ]*)`)

// ProbeUnitDrift 建制差分探针一拍（hungerTick 6h 同槽；零差分=静默零行）。
func (s *Sensor) ProbeUnitDrift(ctx context.Context) error {
	// ① 注册表（canon 单源——读败/缺/空=fail-closed 升件不静默）。
	var reg unitRegistry
	failClosed := false
	raw, err := s.cfg.Get("ops.unit_registry")
	if err != nil {
		failClosed = true
	} else if err := json.Unmarshal(raw, &reg); err != nil || len(reg.Units) == 0 {
		failClosed = true
	}
	if failClosed {
		return s.unitEsc(ctx, unitRegistryMissScope, "registry",
			"建制注册表 ops.unit_registry 缺/读败/空——fail-closed 升件不静默"+
				"（修表/补 seed 归处置席；探针只报不修）")
	}

	beatTicket := reg.Watch.BeatTicket
	if beatTicket == "" {
		beatTicket = "UNITW"
	}
	factSeat, factEvent := reg.Watch.FactSeat, reg.Watch.FactEvent
	if factSeat == "" {
		factSeat = "建制机"
	}
	if factEvent == "" {
		factEvent = "单态"
	}
	periodSec := reg.Watch.PeriodSec
	if periodSec <= 0 {
		periodSec = 300
	}

	// ② 查件死活（beat 陈>3×拍距=查件死→全盲一等事件非静默）。
	var lastBeat *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT last_activity FROM ops_headless WHERE ticket=$1`,
		beatTicket).Scan(&lastBeat); err != nil {
		lastBeat = nil // 行缺=查件从未起拍
	}
	watchDead := lastBeat == nil ||
		s.now().Sub(*lastBeat) > 3*time.Duration(periodSec)*time.Second
	if watchDead {
		var ageStr string
		if lastBeat == nil {
			ageStr = "从无beat行"
		} else {
			ageStr = fmt.Sprintf("beat陈%.0fm", s.now().Sub(*lastBeat).Minutes())
		}
		if err := s.unitEsc(ctx, unitWatchDeadScope, beatTicket,
			fmt.Sprintf("建制查件死 unit=%s——%s（阈=%dmin=3×%ds拍距）——判一盲面全开",
				beatTicket, ageStr, 3*periodSec/60, periodSec)); err != nil {
			return err
		}
	}

	// ③ 判一：逐单元末行态 vs 期望（查件活着才有实查供数——死了归②）。
	if !watchDead {
		for _, u := range reg.Units {
			var note string
			err := s.pool.QueryRow(ctx,
				`SELECT note FROM ops_event WHERE event=$1 AND seat=$2
				  AND note LIKE 'unit='||$3||' %'
				 ORDER BY ts DESC LIMIT 1`, factEvent, factSeat, u.Unit).Scan(&note)
			if err != nil {
				continue // 无单态行=实查静默全配（agent 差分契——常态零写）
			}
			m := unitFactRE.FindStringSubmatch(note)
			if m == nil {
				continue
			}
			gotEn, gotAct := m[1], m[2]
			var drifts []string
			if gotEn != u.ExpectEnabled {
				drifts = append(drifts,
					fmt.Sprintf("enabled=%s(期望%s)", gotEn, u.ExpectEnabled))
			}
			if u.ExpectActive != "" && gotAct != u.ExpectActive {
				drifts = append(drifts,
					fmt.Sprintf("active=%s(期望%s)", gotAct, u.ExpectActive))
			}
			if len(drifts) == 0 {
				continue
			}
			if err := s.unitEsc(ctx, unitDriftEscScope, u.Unit,
				fmt.Sprintf("单元态漂移 unit=%s——%s——处置=核变更意图：误停"+
					"→enable+start复跳∨钦定态→注册表先修码后生效（审计轨契）",
					u.Unit, strings.Join(drifts, "，"))); err != nil {
				return err
			}
		}
	}

	// ④ 判二：产出行鲜度（签名钉了才查——单元活着≠拍子在产）。
	for _, u := range reg.Units {
		if u.ProdSeat == "" || u.ProdEvent == "" || u.ProdPeriodMin <= 0 {
			continue
		}
		var last *time.Time
		if err := s.pool.QueryRow(ctx,
			`SELECT max(ts) FROM ops_event WHERE seat=$1 AND event=$2`,
			u.ProdSeat, u.ProdEvent).Scan(&last); err != nil {
			last = nil
		}
		thresh := 3 * time.Duration(u.ProdPeriodMin) * time.Minute
		if last != nil && s.now().Sub(*last) <= thresh {
			continue
		}
		var ageStr string
		if last == nil {
			ageStr = "从未产出行"
		} else {
			ageStr = fmt.Sprintf("末行陈%.0fm", s.now().Sub(*last).Minutes())
		}
		if err := s.unitEsc(ctx, u.Unit+"_heartbeat_stale", u.Unit,
			fmt.Sprintf("单元产出行陈 unit=%s——签名=%s/%s 阈=%dm：%s——"+
				"单元活着≠拍子在产（service连败/逻辑早退面）",
				u.Unit, u.ProdSeat, u.ProdEvent, u.ProdPeriodMin*3, ageStr)); err != nil {
			return err
		}
	}
	return nil
}

// unitEsc 升件落账（dedup=同 scope+unitTag 未销升件在案即抑；
// 票抑=同 scope 开工单在案静默——closeoutprobe 同契语义层）。
func (s *Sensor) unitEsc(ctx context.Context, scope, unitTag, note string) error {
	var open bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM ops_event e WHERE e.event='升件' AND e.esc_scope=$1
		AND e.note LIKE '%'||$2||'%'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)')))`,
		scope, unitTag).Scan(&open); err != nil {
		open = false // 查败 fail-open——升件优先不吞
	}
	if open {
		return nil
	}
	var tick int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.note ~ ('(^|[^a-zA-Z0-9_])scope='||$1||'([^a-zA-Z0-9_]|$)')
		AND (NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
				AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)') AND d.note !~ ('工单#'||t.id||' ?接续'))
			OR (t.ts + COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour' > now()
				AND EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置'
					AND d.id>t.id AND d.note ~ ('工单#'||t.id||'([^0-9]|$)')
					AND d.note LIKE '%挂起条件=%')))`,
		scope).Scan(&tick); err == nil && tick > 0 {
		return nil // 开工单在案——票管催，探针静默
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket, note, unitDriftEscClass, scope); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}
