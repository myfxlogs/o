// classify.go —— v4 面分类器+自动生候选（判断全机审化——「这面值不值做」
// 不劳会话：分类命中→候选走既有毕业线，paper 零钱径自证，盈亏数据本身=
// 判决书；残余队列 unclassified 等会话——罕见件才用人）。
package gapscan

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// classifyPending 裁 pending/unclassified 行：元数据→族；命中 carry/hedge→
// 生候选（爆面上限）；vol=分类留档生成后期；余者 unclassified 等会话。
func (r *Runner) classifyPending(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT id,venue,surface,metadata FROM fin_surface_gap
		WHERE state IN ('pending','unclassified') ORDER BY id`)
	if err != nil {
		r.rej.Inc("gap_write_err")
		return err
	}
	type gap struct {
		id      int64
		venue   string
		surface string
		meta    map[string]any
	}
	var gs []gap
	for rows.Next() {
		var g gap
		var metaB []byte
		if err := rows.Scan(&g.id, &g.venue, &g.surface, &metaB); err != nil {
			rows.Close()
			return err
		}
		g.meta = map[string]any{}
		_ = json.Unmarshal(metaB, &g.meta)
		gs = append(gs, g)
	}
	rows.Close()
	for _, g := range gs {
		family := classifyFace(g.meta)
		g.meta["family"] = family
		switch family {
		case "carry", "hedge":
			generated, err := r.generateCandidate(ctx, g.venue, g.surface, family, g.meta)
			if err != nil {
				return err
			}
			if generated {
				g.meta["gen"] = "candidate"
			}
		case "vol":
			g.meta["gen"] = "later" // 期权型→波动族〔后期〕——分类留档不生成
		}
		metaB, _ := json.Marshal(g.meta)
		next := "unclassified"
		if family == "carry" || family == "hedge" {
			next = "pending" // 候选已进毕业线——面仍待注册裁
		}
		if _, err := r.pool.Exec(ctx, `UPDATE fin_surface_gap SET state=$2, metadata=$3,
			last_seen=now() WHERE id=$1 AND state IN ('pending','unclassified')`,
			g.id, next, metaB); err != nil {
			r.rej.Inc("gap_write_err")
			return err
		}
	}
	return nil
}

// classifyFace 元数据→模板族映射（v4 钦定规则谱；元数据键由探针 glue 提取）。
func classifyFace(meta map[string]any) string {
	if truthy(meta["yield_bearing"]) {
		return "carry" // 收益锁仓型→carry 族（dcd/earn 阶梯）
	}
	if truthy(meta["funding_bearing"]) {
		return "hedge" // 永续含 funding→对冲族
	}
	if s, ok := meta["swap_sign"].(float64); ok {
		if s > 0 {
			return "carry" // CFD swap 正值→carry 族
		}
		return "hedge" // 负值→对冲腿或趋势族
	}
	if truthy(meta["options"]) {
		return "vol" // 期权型→波动族（后期）
	}
	return "unclassified"
}

// generateCandidate 分类命中→候选行（provenance=surface_gap 标注；幂等=
// 同面在途候选不重生；爆面上限=config research.gap.gen_cap 默认 3——防模板
// 洪水灌爆沙箱；generated 策略只许 paper 段——lifecycle 既有链无特殊通道）。
func (r *Runner) generateCandidate(ctx context.Context, venue, surface, family string, meta map[string]any) (bool, error) {
	instID := venue + "/" + surface
	var open int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id=$1 AND state IN ('fin_proposed','fin_armed')`, instID).Scan(&open); err != nil {
		r.rej.Inc("gap_write_err")
		return false, err
	}
	if open > 0 {
		return false, nil // 同面候选在途——不重生（幂等）
	}
	cap := 3.0
	if v, err := r.cfg.GetFloat("research.gap.gen_cap"); err == nil && v > 0 {
		cap = v
	}
	var gen int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE direction IN ('gap_carry','gap_hedge') AND state IN ('fin_proposed','fin_armed')`).Scan(&gen); err != nil {
		r.rej.Inc("gap_write_err")
		return false, err
	}
	if float64(gen) >= cap {
		r.rej.Inc("gap_gen_cap")
		log.Printf("WARN [research/gap] 候选生成达上限 %d——%s/%s 排队下拍（防模板洪水）", int(cap), venue, surface)
		return false, nil
	}
	direction := "gap_hedge"
	if family == "carry" {
		direction = "gap_carry"
	}
	meta["provenance"] = "surface_gap"
	ev, _ := json.Marshal(meta)
	var evalID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,
		gross_apr,net_apr,verdict,legs,evidence)
		VALUES($1,'surface-gap',$2,NULL,NULL,'proposed','{}',$3)
		ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now(),
		verdict=EXCLUDED.verdict,evidence=EXCLUDED.evidence RETURNING id`,
		instID, direction, ev).Scan(&evalID); err != nil {
		r.rej.Inc("gap_write_err")
		return false, err
	}
	// T-0142 黑名单滤：venue 拒过的 inst 不再起草（同族不再武装——学费一次）。
	var banned int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
		WHERE inst_id=$1`, instID).Scan(&banned); err == nil && banned > 0 {
		log.Printf("[gapscan] %s 黑名单命中（venue_untradeable）——候选不起草", instID)
		return false, nil
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,$2,$3,0)`, evalID, instID, direction); err != nil {
		r.rej.Inc("gap_write_err")
		return false, err
	}
	log.Printf("[research/gap] 面候选生成 %s（family=%s provenance=surface_gap——走毕业线 paper 段）", instID, family)
	return true, nil
}

// PendingCount 待裁面计数（卡面「正在做」露出一行——业主可见「系统发现了几件
// 还没裁的事」）。
func PendingCount(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_surface_gap
		WHERE state IN ('pending','unclassified')`).Scan(&n)
	return n, err
}

func truthy(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func isNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

// asErr 泛型 errors.As 便捷形（switch 分型消费）。
func asErr[T error](err error, dst *T) bool { return errors.As(err, dst) }
