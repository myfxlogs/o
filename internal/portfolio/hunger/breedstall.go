// breedstall.go —— 注册族卡零产出饿涸探针（工单O-39330——「族卡→评估臂」
// 无主转换感知器）：fin_opp_template_roster std_state='candidate' 卡
// 归一名下 fin_opp_eval/fin_strategy 零产出 ∧ 假设锚龄超
// dm.hunger.breed_stall_days → 升件自带归因卡。供给侧落册即完成态、
// 消费侧（臂建=工程活）无主发起——同型病灶第三处（教训矿 O-38942 已修、
// 采集缺口转换未建、本件补族卡段）。
//
// 判据三分（spec 钦定序）：
// ①候选面=roster std_state='candidate'（视图=fin_opp_template 投影——
//
//	live/disabled/retired 态不查）；②产出面=归一名（_↔- 双向归一——
//	eval 记 'stake-carry' vs roster 'stake_carry' 方言实证）下
//	fin_opp_eval ∪ fin_strategy.template_ref 零行（proposals 经 eval_id
//	回链只作归因计数不作判据——eval 在册即产出已立）；③龄锚=
//	fin_opp_hypothesis.family 归一最新 ts——缺锚照报 note 载「锚缺」
//	（卡存在零产出即事实，龄只修措辞——同 livedrought 无史先例降档）。
//
// 升件形：seat=探测席/event=升件/esc_class='breed_stall'/esc_scope=
// template（去重等值键）/24h 同窗去重（droughtDedupHours 同例）/
// note=归因卡（出生锚+evals/strategies/proposals 三计数+建议语——
// 探针只报不裁：禁自动立工单/改 roster 态/建臂，立票分配归唤知判归席）。
// 品类盲硬条款：谓词只读通用登记面（roster/eval/strategy/hypothesis/
// proposal），禁引用品类/venue 语义白名单——新交易类别进册自动覆盖。
// fail-closed：dm.hunger.breed_stall_days 缺/非正=报错不探（防哑针，
// 迁移 fin_0253 落键）。
//
// 附带面裁定（spec D 偏差申报在案）：fin_surface_gap 权威队列在 PG-md
// （oci——gapscan/classify/hypothesis 三写家全走 md 池），本探针 fin 池
// （SG）物理不可达；SG 副本系分域前孤儿表（md_1044 唯一建表件、fin 链
// 零建表零写家、last_seen 2026-10-27 死表）——对死表探针=谎报，gap 断流
// 探针须挂 md 侧器官另案。
package hunger

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// breedStallEscClass 升件类（fin_0253 迁移固化——ops_esc_valence
// 'breed_stall'=65 在案；scope=template 供 24h 同窗去重等值键）。
const breedStallEscClass = "breed_stall"

// breedCandCard candidate 族卡聚合行（roster 视图按 template 收拢——
// 同 template 多 venue/surface 行归并一升件，esc_scope=template 等值键）。
type breedCandCard struct {
	template string
	rows     int
	venues   string
	surfaces string
}

// ProbeBreedStall 族卡断流探针一拍（hungerTick 同槽——ProbeDeskOrphan
// 后 Detect 前，败情隔离位同款）：逐 candidate 族卡判零产出+超龄→升件；
// 全卡静默=静默。败=上抛归 hungerTick 自报位非静默。
func (s *Sensor) ProbeBreedStall(ctx context.Context) error {
	days, err := s.cfg.GetInt("dm.hunger.breed_stall_days")
	if err != nil {
		return fmt.Errorf("族卡断流窗键不可读: %w", err)
	}
	if days <= 0 {
		return fmt.Errorf("族卡断流窗非正（%d——键值坏 fail-closed）", days)
	}
	rows, err := s.pool.Query(ctx, `SELECT template, count(*),
		string_agg(DISTINCT venue,',' ORDER BY venue),
		string_agg(DISTINCT coalesce(surface,'-'),',' ORDER BY coalesce(surface,'-'))
		FROM fin_opp_template_roster WHERE std_state='candidate'
		GROUP BY template ORDER BY template`)
	if err != nil {
		return fmt.Errorf("候选族卡面读败: %w", err)
	}
	var cards []breedCandCard
	for rows.Next() {
		var c breedCandCard
		if err := rows.Scan(&c.template, &c.rows, &c.venues, &c.surfaces); err != nil {
			rows.Close()
			return fmt.Errorf("候选族卡扫败: %w", err)
		}
		cards = append(cards, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("候选族卡迭代败: %w", err)
	}
	for _, c := range cards {
		if err := s.breedStallOne(ctx, c, days); err != nil {
			return err
		}
	}
	return nil
}

// breedStallOne 单族卡判别+升件。归一名=replace(_→-)——eval 侧连字符
// 方言（'stake-carry'）与 roster 下划线方言（'stake_carry'）双向归一
// 硬条款，三处同用（eval/strategy/hypothesis）。
func (s *Sensor) breedStallOne(ctx context.Context, c breedCandCard, days int64) error {
	norm := strings.ReplaceAll(c.template, "_", "-")
	var evals, strats, props int
	if err := s.pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM fin_opp_eval e WHERE replace(e.template,'_','-')=$1),
		(SELECT count(*) FROM fin_strategy s WHERE replace(coalesce(s.template_ref,''),'_','-')=$1),
		(SELECT count(*) FROM fin_proposal p JOIN fin_opp_eval e ON e.id=p.eval_id
		 WHERE replace(e.template,'_','-')=$1)`, norm).Scan(&evals, &strats, &props); err != nil {
		return fmt.Errorf("族卡产出面读败 template=%s: %w", c.template, err)
	}
	if evals > 0 || strats > 0 {
		return nil // 产出已立——评估臂转换发生过，不查
	}
	var hypTs *time.Time
	if err := s.pool.QueryRow(ctx,
		`SELECT max(ts) FROM fin_opp_hypothesis WHERE replace(family,'_','-')=$1`,
		norm).Scan(&hypTs); err != nil {
		return fmt.Errorf("族卡假设锚读败 template=%s: %w", c.template, err)
	}
	ageText := "锚缺"
	stalled := true // 缺锚照报——零产出事实不虚构龄（livedrought 无史先例降档）
	if hypTs != nil {
		d := s.now().Sub(*hypTs).Hours() / 24
		ageText = fmt.Sprintf("%.1fd(假设锚=%s)", d, hypTs.UTC().Format("01-02 15:04"))
		stalled = d > float64(days)
	}
	if !stalled {
		return nil
	}
	var dup int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		breedStallEscClass, c.template, droughtDedupHours).Scan(&dup); err != nil {
		return fmt.Errorf("族卡断流去重读败 template=%s: %w", c.template, err)
	}
	if dup > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES($1,'升件',$2,$3,$4,$5)`,
		hungerSeat, hungerTicket,
		fmt.Sprintf("族卡断流 %s：candidate在册×%d(venue=%s surface=%s) 龄=%s evals=%d strategies=%d proposals=%d——族卡→评估臂无主转换——臂建工单需立（探针只报不裁——立票分配归唤知，判归席）",
			c.template, c.rows, c.venues, c.surfaces, ageText, evals, strats, props),
		breedStallEscClass, c.template); err != nil {
		return fmt.Errorf("族卡断流升件行 template=%s: %w", c.template, err)
	}
	return nil
}
