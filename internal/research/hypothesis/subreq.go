// subreq.go —— 晋级径临时订阅请求面（T-0401 两级采集制②）：假想臂评估
// 候选符（粗扫+信号筛过阈而 inst ∉ activeSet）→fin_sub_request open 行
// （临时订阅请求——collector syncActiveSet 经 legset.SubRequestArm UNION
// 拾取流式采）；评估判词出：过=提案臂接管常驻（提案/策略面在集→扫尾闭
// 请），拒=闭请退订。扫尾面 reconcileSubs 防订阅泄漏（超龄未转常驻亦闭
// ——滞留另有 0394 探针族升件；T-0412 user_intent 常驻意图行豁免扫尾）。
package hypothesis

import (
	"context"
	"log"
)

// subReqAgeHours 请求超龄上限（小时）——app_config research.subreq.
// max_age_hours 热启，缺=24h 保守（提名是增益面非常驻权）。
const subReqAgeHours = 24.0

// AttachCatalog 挂宇宙扩件目录存在性闸（T-0416 顺手闸——MUU 型假符进盒
// 前拒掉：inst ∉ 宇宙目录不提名；目录读败=提名让位 fail-closed 不虚构）。
func (r *Runner) AttachCatalog(fn func(ctx context.Context) (map[string]bool, error)) *Runner {
	r.cat = fn
	return r
}

// nominateSub 评估臂提名：inst ∉ activeSet→open 请求行（幂等同键复开）；
// 已在集=零动作（提案/策略/派生臂已常驻——提名是增益面非阻断面，读败让位）。
// T-0416 catalog 闸：挂线后 inst ∉ 宇宙目录=拒提名（venue 不存在符订阅
// error 帧洪泛病灶闸——提名面未核 catalog 根修）。
func (r *Runner) nominateSub(ctx context.Context, inst, requester, reason string) {
	if r.cat != nil {
		uni, err := r.cat(ctx)
		if err != nil {
			log.Printf("[research/hypo] 晋级提名 %s 让位——宇宙目录读败（fail-closed 不虚构）: %v", inst, err)
			return
		}
		if !uni[inst] {
			r.rej.Inc("nom_offcatalog")
			log.Printf("[research/hypo] 晋级提名 %s 拒——inst 宇宙目录外（假符进盒前闸）", inst)
			return
		}
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM (`+permSet+`) t
		WHERE t.inst_id=$1`, inst).Scan(&n); err != nil || n > 0 {
		return
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_sub_request(venue,inst_id,requester,state,reason)
		VALUES('okx',$1,$2,'open',$3)
		ON CONFLICT (venue,inst_id,requester) DO UPDATE SET state='open',
		reason=EXCLUDED.reason,ts=now()`, inst, requester, reason); err != nil {
		r.rej.Inc("hypo_write_err")
		return
	}
	log.Printf("[research/hypo] 晋级提名 %s（%s——临时订阅请求 activeSet UNION 拾取）", inst, reason)
}

// closeSub 提名闭请（拒=退订——同键 open 行翻 closed，UNION 臂即出集）。
func (r *Runner) closeSub(ctx context.Context, inst, requester string) {
	if _, err := r.pool.Exec(ctx, `UPDATE fin_sub_request SET state='closed',recv_ts=now()
		WHERE venue='okx' AND inst_id=$1 AND requester=$2 AND state='open'`,
		inst, requester); err != nil {
		r.rej.Inc("hypo_write_err")
	}
}

// permSet 常驻集谓词（晋级臂除外的既有臂面——提案/策略在集即「经提案常驻」，
// 判常驻不许引 SubRequestArm 本体否则 open 行自证常驻死循环）。
const permSet = `SELECT inst_id FROM fin_proposal WHERE state IN ('fin_proposed','fin_armed')
	UNION SELECT inst_id FROM fin_strategy WHERE state<>'retired'`

// reconcileSubs 请求面扫尾（ScanSignals 尾调——三闭请件）：
//
//	①常驻已就位：inst 入 permSet（提案/策略面接管）→闭请——「过=经提案常驻」；
//	②判词拒：inst 最新评估行 verdict='rejected'→闭请——「拒=退订」；
//	③超龄未转：open>max_age_hours→闭请防订阅泄漏（滞留由探针升件非本面修）。
//
// T-0412 豁免：requester='user_intent' 常驻意图行不扫尾——用户亲手要的符
// 非评估判词可裁，生命周期归用户面（滞留由 0394 subreq 探针升件同径罩）。
func (r *Runner) reconcileSubs(ctx context.Context) {
	ageH := subReqAgeHours
	if v, err := r.cfg.GetFloat("research.subreq.max_age_hours"); err == nil && v > 0 {
		ageH = v
	}
	tag, err := r.pool.Exec(ctx, `UPDATE fin_sub_request q SET state='closed',recv_ts=now()
		WHERE q.state='open' AND q.requester<>'user_intent' AND (
			EXISTS (SELECT 1 FROM (`+permSet+`) t WHERE t.inst_id=q.inst_id)
			OR EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.inst_id=q.inst_id
				AND e.verdict='rejected' AND e.ts>now()-interval '1 day')
			OR q.ts < now()-make_interval(hours=>$1))`, ageH)
	if err != nil {
		r.rej.Inc("hypo_write_err")
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("[research/hypo] 晋级请求扫尾闭请 %d 行（常驻/判拒/超龄三件套）", n)
	}
}
