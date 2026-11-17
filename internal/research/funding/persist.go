// persist.go —— 评估卡落表面（T-0162 自 run.go 拆出：写拍批内合流事务面）。
package funding

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"o/internal/platform/execfam"
	"o/internal/research/evalgate"
)

// spotLegOK 现货腿存在性预核（起草闸）：短_perp_long_spot 族的 spot 腿
// （基币-USDT）须在 venue 目录实有——perp-only 合成器（股权/指数映射类）
// 永续有 perp 无 spot，提案肥数字全是空气（T-0150 实案）。黑名单
// fin_venue_untradeable 同读面复用。目录缓存 10min。读败=fail-closed
// 不起草（宁缺不空气）。
func (r *Runner) spotLegOK(ctx context.Context, instID string) (bool, string) {
	base := strings.TrimSuffix(instID, "-USDT-SWAP")
	spotInst := base + "-USDT"
	var banned int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
		WHERE inst_id IN ($1,$2)`, instID, spotInst).Scan(&banned); err == nil && banned > 0 {
		return false, "blacklisted"
	}
	if r.spotFeed == nil {
		return true, "" // 读件未挂=预核缺席（装配层挂线前存量行为）
	}
	if r.spots == nil || time.Since(r.spotAt) > 10*time.Minute {
		raw, err := r.spotFeed(ctx)
		if err != nil {
			return false, "spot_catalog_read_failed"
		}
		var rows []struct {
			InstID string `json:"instId"`
		}
		if err := json.Unmarshal(raw, &rows); err != nil {
			return false, "spot_catalog_parse_failed"
		}
		r.spots = map[string]bool{}
		for _, row := range rows {
			r.spots[row.InstID] = true
		}
		r.spotAt = time.Now()
	}
	if !r.spots[spotInst] {
		return false, "no_spot_leg"
	}
	return true, ""
}

// persist 评估卡落表；proposed 且无 open 提案→fin_proposal 行（闸后唯一提案径）。
// tx=拍级合流事务（T-0162——评估/提案写与读一致面同事务，批内 NOTIFY 去重在位）。
func (r *Runner) persist(ctx context.Context, tx pgx.Tx, e Eval, p Params) error {
	// T-0330 前置拒：执行径族册出局=proposed 翻 rejected/exec_family_missing
	// 如实，不进提案队列（族册 execfam 移除本族即自动生效——与其他臂同闸）。
	if e.Verdict == "proposed" && !execfam.Has("funding-hedge") {
		e.Verdict = "rejected"
		e.Evidence["reason"] = "exec_family_missing"
	}
	// T-0150 起草闸前置：proposed 提案落卡前核现货腿存在性——无 spot 腿
	// （perp-only 合成器）=评估行改 rejected 如实拒绝由，不进提案队列。
	// 前置于指纹闸：终判词须先定——proposed↔rejected 摆动每拍产新版=闸失效
	// （原插后 UPDATE 两行写径并一：终态直落单行）。
	if e.Verdict == "proposed" {
		if ok, why := r.spotLegOK(ctx, e.InstID); !ok {
			// verdict 闭集（0006 三值）——infeasible 语义落 rejected+拒绝由
			// （neg_funding_infeasible 同构先例）。
			if e.Evidence == nil {
				e.Evidence = map[string]any{}
			}
			e.Evidence["reason"] = why
			e.Verdict = "rejected"
			r.rej.Inc("no_spot_leg")
			log.Printf("[research] %s 起草闸拒（%s——perp-only 合成器不进提案队列）", e.InstID, why)
		}
	}
	legs, _ := json.Marshal(e.Legs)
	ev, _ := json.Marshal(e.Evidence)
	// 工单#22904 指纹闸：同(inst,template,direction)最新行 verdict+net_apr
	// 未变=信念未迁→跳 INSERT 复用行 id 续提案径（md_1097 insert-only 后
	// 同指纹每拍重写=盘烧；判词/净额变迁仍产新版史）。
	evalID, same, err := evalgate.SameFingerprint(ctx, tx, e.InstID, "funding-hedge", e.Direction, e.Verdict, e.NetAPR)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	if !same {
		if err := tx.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
			VALUES($1,'funding-hedge',$2,$3,$4,$5,$6,$7) RETURNING id`,
			e.InstID, e.Direction, e.GrossAPR, e.NetAPR, e.Verdict, legs, ev).Scan(&evalID); err != nil {
			return err
		}
	}
	if e.Verdict != "proposed" {
		return nil
	}
	var open int
	if err := tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM fin_proposal
		WHERE inst_id=$1 AND direction=$2 AND state IN ('fin_proposed','fin_armed'))
		+ (SELECT count(*) FROM fin_strategy
		WHERE inst_id=$1 AND direction=$2 AND state<>'retired')`,
		e.InstID, e.Direction).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return nil // 已挂提案不重复提案（滞留禁超 SLA 由生命周期票管）
	}
	_, err = tx.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,$4)`, evalID, e.InstID, e.Direction, e.NetAPR)
	if err == nil {
		log.Printf("[research] 提案落卡 %s %s net_apr=%.4f（验证官闸过）", e.InstID, e.Direction, e.NetAPR)
	}
	return err
}
