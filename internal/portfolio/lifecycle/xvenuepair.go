// xvenuepair.go —— 跨所 carry 双腿 paper 开户（T-0144）：主从策略对——主行
// venue='okx' inst=perp（收/付 funding 端），副行 venue='mt5' inst=CFD（swap
// 端）；两腿 proposal_ref 同引（同提案生双腿），p_and_l 聚合在提案级
// （fin_paper_fill 按 strategy_id 聚合可查）。副行建行时锁 mt5_swap 信号快照
// 入 fin_paper_fill kind='swap_snapshot'（防回改——票面钦定）。
// MT5 CFD inst 反查=app_config bridge.xvenue.map 同映射源（不另造）。
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// admitXvenuePair 主从双腿开户（主 okx perp+副 mt5 CFD）。
func (r *Runner) admitXvenuePair(ctx context.Context, proposalID int64, inst, dir string,
	notional, minH, minNet, maxDD float64) error {
	// 幂等：提案已挂任一腿=复用不叠仓。
	var exists int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE proposal_ref=$1`, proposalID).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", proposalID); err != nil {
			return err
		}
		return nil
	}
	// CFD 反查（bridge.xvenue.map okx→mt5 同映射源）。
	var mapRaw []byte
	if err := r.pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key='bridge.xvenue.map'`).Scan(&mapRaw); err != nil {
		return fmt.Errorf("bridge.xvenue.map 缺——跨所双腿无映射 fail-closed: %w", err)
	}
	xm := map[string]string{}
	if err := json.Unmarshal(mapRaw, &xm); err != nil {
		return err
	}
	cfd, ok := xm[inst]
	if !ok || cfd == "" {
		return fmt.Errorf("xvenue.map 无 %s 映射——双腿缺副腿 fail-closed", inst)
	}
	crit, _ := json.Marshal(map[string]float64{"min_hours": minH, "min_net": minNet, "max_dd": maxDD})
	// 主行（okx perp——funding 端）。
	var mainID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria,direction,venue)
		VALUES($1,'xvenue-carry',$2,'paper',$3,$4,$5,'okx')
		ON CONFLICT (venue,inst_id,direction,template_ref) WHERE state<>'retired' DO NOTHING
		RETURNING id`, inst, proposalID, notional, crit, dir).Scan(&mainID); err != nil {
		if err.Error() == "no rows in result set" {
			return nil // 同键主行在挂
		}
		return err
	}
	// 副行（mt5 CFD——swap 端；向取反：主长 perp→副空 CFD）。
	cfdDir := dir
	if dir == "long_perp_short_cfd" {
		cfdDir = "short_cfd"
	} else if dir == "short_perp_long_cfd" {
		cfdDir = "long_cfd"
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,criteria,direction,venue)
		VALUES($1,'xvenue-carry',$2,'paper',$3,$4,$5,'mt5')
		ON CONFLICT (venue,inst_id,direction,template_ref) WHERE state<>'retired' DO NOTHING`,
		cfd, proposalID, notional, crit, cfdDir); err != nil {
		return err
	}
	var cfdID int64
	if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_strategy
		WHERE proposal_ref=$1 AND venue='mt5'`, proposalID).Scan(&cfdID); err != nil {
		return err
	}
	// 双腿 Transition 留痕。
	for _, leg := range []struct {
		id int64
		v  string
	}{{mainID, "okx"}, {cfdID, "mt5"}} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
			VALUES($1,'proposed','paper',$2)`, leg.id,
			"跨所 carry 双腿 paper 开户（T-0144——venue="+leg.v+"）"); err != nil {
			return err
		}
	}
	// swap 入账快照（拍时锁 mt5_swap meta raw 防回改——票面钦定）。
	var snap []byte
	if err := r.pool.QueryRow(ctx, `SELECT meta FROM md_signal
		WHERE kind='mt5_swap' AND key=$1 ORDER BY ts DESC LIMIT 1`, cfd).Scan(&snap); err == nil {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'swap_snapshot',$2,0,$3,now())`, cfdID, cfd, snap); err != nil {
			return err
		}
	}
	if _, err := r.pool.Exec(ctx,
		"UPDATE fin_proposal SET state='fin_armed' WHERE id=$1", proposalID); err != nil {
		return err
	}
	log.Printf("[lifecycle] xvenue_carry 双腿 paper 开户 #%d（okx %s + mt5 %s——armed 止，真执行 T-0145）",
		proposalID, inst, cfd)
	return nil
}
