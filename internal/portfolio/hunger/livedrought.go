// livedrought.go —— live 终局断流探针（工单O-35729——业主令「钱闲着」与
// 管道断同格响）：fin_strategy state='live' 在册=0 持续 ≥
// dm.hunger.live_drought_days 天→升件自带归因卡。终局段（真钱在赚没有）
// 此前各段探针全覆盖唯独无盯——各段活≠全链活。
//
// 词面正位（票面写 fin_position state='live'——fin_0218 FSM 词表钉死
// open/closing/closed/orphaned，'live' 在 fin_position 物理不存在；
// 毕业终局正主=fin_strategy.state='live'——live_dud/idle_capital 同源
// 判据。仓位行系 live 策略开仓产物、存活分钟级——以仓位计断流常态误报，
// 不取）：持续锚=fin_strategy_transition 末次涉 live 迁移 ts——在册=0
// 时末次必为出 live 降格（账本自证断流起点）；史无 live 迁移=自策略
// 宇宙诞生起算，连策略史都无=按超阈火（P0 闲置成立）。
//
// 火时归因卡复用 opendrought 卡形（确定性查询先行，判归席不归探针）：
// 提案流鲜度/eval拒谱/paper·probe池计数与最陈/漏斗迁移流/退役因top/
// 断点推定+责任归位行（业主令④——谁在让钱闲着卡先答）+在办证。
// 只报不修、24h 同窗去重（esc_class+esc_scope 等值键）。fail-closed 键缺
// =探针报错不探（迁移 fin_0240 落键防哑针）。
package hunger

import (
	"context"
	"fmt"
	"time"
)

// liveDroughtEscClass 升件类（本件新类——ops_esc_valence 'live_drought'=100
// 顶格在案，fin_0240 迁移固化；scope 同词供 24h 同窗去重等值键）。
const liveDroughtEscClass = "live_drought"

// ProbeLiveDrought live 终局断流探针一拍：live 在册=0 且断流超阈→归因卡
// 升件；在册>0 或未超龄=静默。
func (s *Sensor) ProbeLiveDrought(ctx context.Context) error {
	days, err := s.cfg.GetInt("dm.hunger.live_drought_days")
	if err != nil {
		return fmt.Errorf("live断流窗键不可读: %w", err)
	}
	if days <= 0 {
		return fmt.Errorf("live断流窗非正（%d——键值坏 fail-closed）", days)
	}
	var liveN int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_strategy WHERE state='live'`).Scan(&liveN); err != nil {
		return fmt.Errorf("live 在册读败: %w", err)
	}
	if liveN > 0 {
		return nil // 终局段有活——静默
	}
	var lastLive *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(ts) FROM fin_strategy_transition
		 WHERE from_state='live' OR to_state='live'`).Scan(&lastLive); err != nil {
		return fmt.Errorf("live 迁移锚读败: %w", err)
	}
	ageDays, ageText, err := s.liveDroughtAge(ctx, lastLive)
	if err != nil {
		return err
	}
	if ageDays <= float64(days) {
		return nil // 未超龄——静默
	}
	card, err := s.liveDroughtCard(ctx, ageText)
	if err != nil {
		return err
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		liveDroughtEscClass, "live_drought", droughtDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("同窗去重读败: %w", err)
	}
	if dup > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		"live 终局断流——"+card+"（探针只报不修——归因卡自带，修/裁归席；处置三择一=部署/有理闲置(判据+到期日)/问董事会）",
		liveDroughtEscClass, "live_drought"); err != nil {
		return fmt.Errorf("升件行: %w", err)
	}
	return nil
}

// liveDroughtAge 断流龄机读：末次涉 live 迁移 ts 距今；史无 live 迁移回退
// 策略宇宙诞生（min(created_at)）；连策略史都无=按超阈（1e9 恒过阈比——
// ageText 载「无策略史」事实不虚构天数）。
func (s *Sensor) liveDroughtAge(ctx context.Context, lastLive *time.Time) (float64, string, error) {
	if lastLive != nil {
		d := s.now().Sub(*lastLive).Hours() / 24
		return d, fmt.Sprintf("%.1fd(末次live迁移=%s)", d,
			lastLive.UTC().Format("01-02 15:04")), nil
	}
	var t0 *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT min(created_at) FROM fin_strategy`).Scan(&t0); err != nil {
		return 0, "", fmt.Errorf("策略宇宙诞生锚读败: %w", err)
	}
	if t0 == nil {
		return 1e9, "无策略史", nil
	}
	d := s.now().Sub(*t0).Hours() / 24
	return d, fmt.Sprintf("%.1fd(自策略诞生=%s,史无live迁移)", d,
		t0.UTC().Format("01-02")), nil
}

// liveDroughtCard 归因卡拼装（全确定性查询——任一段读败整卡不拼，
// fail-closed 走 hungerEscalate 探针自报径，同 droughtCard 族款）。
func (s *Sensor) liveDroughtCard(ctx context.Context, ageText string) (string, error) {
	var prop, rej int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE verdict='proposed'), count(*) FILTER (WHERE verdict='rejected')
		 FROM fin_opp_eval WHERE ts>now()-interval '7 days'`).Scan(&prop, &rej); err != nil {
		return "", fmt.Errorf("eval 谱读败: %w", err)
	}
	var reasons string
	if err := s.pool.QueryRow(ctx,
		`SELECT coalesce(string_agg(split_part(r,' ',1)||'×'||rn,',' ORDER BY rn DESC),'(无)') FROM
		  (SELECT evidence->>'reason' r, count(*) rn FROM fin_opp_eval
		   WHERE verdict='rejected' AND ts>now()-interval '7 days'
		   AND evidence->>'reason' IS NOT NULL
		   GROUP BY 1 ORDER BY rn DESC LIMIT 3) x`).Scan(&reasons); err != nil {
		return "", fmt.Errorf("拒因谱读败: %w", err)
	}
	var paper, probe int
	var probeMaxH float64
	var p2p, p2l, pDead int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FILTER (WHERE state='paper'),
		  count(*) FILTER (WHERE state='probe'),
		  COALESCE(max(EXTRACT(EPOCH FROM (now()-entered_at))/3600.0)
		    FILTER (WHERE state='probe'),0),
		  (SELECT count(*) FROM fin_strategy_transition
		    WHERE ts>now()-interval '30 days' AND from_state='paper' AND to_state='probe'),
		  (SELECT count(*) FROM fin_strategy_transition
		    WHERE ts>now()-interval '30 days' AND from_state='probe' AND to_state='live'),
		  (SELECT count(*) FROM fin_strategy_transition
		    WHERE ts>now()-interval '30 days' AND from_state='probe'
		      AND to_state IN ('retired','degraded'))
		 FROM fin_strategy WHERE state IN ('paper','probe')`).Scan(
		&paper, &probe, &probeMaxH, &p2p, &p2l, &pDead); err != nil {
		return "", fmt.Errorf("培育池读败: %w", err)
	}
	var deadTop string
	if err := s.pool.QueryRow(ctx,
		`SELECT coalesce(string_agg(reason||'×'||rn,',' ORDER BY rn DESC),'(无)') FROM
		  (SELECT reason, count(*) rn FROM fin_strategy_transition
		   WHERE ts>now()-interval '30 days' AND to_state IN ('retired','degraded')
		   GROUP BY reason ORDER BY rn DESC LIMIT 3) x`).Scan(&deadTop); err != nil {
		return "", fmt.Errorf("退役因谱读败: %w", err)
	}
	var swVal string
	liveSw := false
	if err := s.pool.QueryRow(ctx,
		`SELECT value FROM app_config WHERE key='invest.lifecycle.live_switch'`).Scan(&swVal); err == nil {
		liveSw = swVal == "true"
	}
	var openPos, inFlight int
	if err := s.pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM fin_position WHERE state<>'closed'),
		  (SELECT count(*) FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed'))`).Scan(
		&openPos, &inFlight); err != nil {
		return "", fmt.Errorf("在办面读败: %w", err)
	}
	swTxt := "off"
	if liveSw {
		swTxt = "on"
	}
	return fmt.Sprintf(
		"live在册=0 持续=%s|eval7d=提案%d/拒%d|拒因top=%s|paper=%d probe=%d(最陈%.1fh)|漏斗30d=paper→probe%d/probe→live%d/probe→退%d|退役因top=%s|live_switch=%s|断点推定=%s|责任=钱径管理面(lifecycle/DM)|在办=提案在途%d/仓位open%d",
		ageText, prop, rej, reasons, paper, probe, probeMaxH,
		p2p, p2l, pDead, deadTop, swTxt,
		liveDroughtPoint(liveSw, prop, rej, paper, probe), inFlight, openPos), nil
}

// liveDroughtPoint 断点推定（阈内派生——顺序=闸门自上而下，先中先出）：
// 委员会 live_switch 关=结构性断（真钱闸未开与策略优劣无关）；
// eval 提案零=上游断供；培育层全空=无在培；probe 空而 paper 在=paper→probe
// 断（判据未越线/闸拦）；probe 在而 live 零=probe→live 断（观察窗/演习闸/
// 净回优）。判归席不归探针——推定只是事实的阈值化标记。
func liveDroughtPoint(liveSw bool, prop, rej, paper, probe int) string {
	if !liveSw {
		return "委员会闸(live_switch=off)"
	}
	if prop == 0 && rej > 0 {
		return "评估层(提案0/全拒)"
	}
	if prop == 0 {
		return "评估层(7d零提案)"
	}
	if paper == 0 && probe == 0 {
		return "培育层空(无paper/probe)"
	}
	if probe == 0 {
		return "paper→probe断(判据未越线/闸拦)"
	}
	return "probe→live断(观察窗/演习闸/净回优)"
}
