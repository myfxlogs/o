package hunger

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// persist 全量 stale 标+逐行幂等 upsert（ON CONFLICT (inst,mechanism)
// DO UPDATE——陈值 stale=false 刷新；读败/出宇宙行留 stale=true 不删）。
func (c *censusCtx) persist(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `UPDATE fin_opp_map SET stale=true`); err != nil {
		return fmt.Errorf("普查 stale 标败: %w", err)
	}
	for _, r := range c.rows {
		legsJ, _ := json.Marshal(r.legs)
		ev := map[string]any{}
		for k, v := range r.evidence {
			ev[k] = v
		}
		// 本行 inst 的 spread 样本续存（当拍值入列+封顶——四机制行同值）。
		if bps, ok := spreadBpsNow(c.ticks[r.inst]); ok {
			smp := append(append([]float64(nil), c.spreadSmp[r.inst]...), bps)
			if len(smp) > censusSpreadCap {
				smp = smp[len(smp)-censusSpreadCap:]
			}
			ev["spread_samples"] = smp
		} else if len(c.spreadSmp[r.inst]) > 0 {
			ev["spread_samples"] = c.spreadSmp[r.inst]
		}
		evJ, _ := json.Marshal(ev)
		if _, err := pool.Exec(ctx,
			`INSERT INTO fin_opp_map(inst_id,mechanism,legs,net_est_bps,verdict,stale,evidence,updated_at)
			 VALUES($1,$2,$3,$4,$5,false,$6,now())
			 ON CONFLICT (inst_id,mechanism) DO UPDATE SET
			   legs=EXCLUDED.legs, net_est_bps=EXCLUDED.net_est_bps,
			   verdict=EXCLUDED.verdict, stale=false,
			   evidence=EXCLUDED.evidence, updated_at=now()`,
			r.inst, r.mechanism, legsJ, r.netBps, r.verdict, evJ); err != nil {
			return fmt.Errorf("opp_map upsert 败 inst=%s mech=%s: %w", r.inst, r.mechanism, err)
		}
	}
	return nil
}

// escalate 升件筛：executable∧net>minNet∧eval 臂从未评过→dedup→净估降序
// top escMax 落升件行（note=净估/腿齐/eval 无记录实证——禁对策词）。
func (c *censusCtx) escalate(ctx context.Context, s *Sensor, cfg censusCfg) error {
	type cand struct {
		inst, mech, evalInst string
		net                  float64
	}
	var cands []cand
	for _, r := range c.rows {
		if r.verdict != "executable" || r.netBps == nil || *r.netBps <= cfg.minNet {
			continue
		}
		ei := r.evalInst
		if ei == "" {
			ei = r.inst
		}
		evald, err := s.censusEvald(ctx, ei, r.mechanism)
		if err != nil {
			return err
		}
		if evald {
			continue // eval 臂评过——不重复催
		}
		cands = append(cands, cand{r.inst, r.mechanism, ei, *r.netBps})
	}
	if len(cands) == 0 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].net > cands[j].net })
	fired := 0
	for _, cd := range cands {
		if fired >= int(cfg.escMax) {
			break
		}
		scope := cd.inst + ":" + cd.mech
		dup, err := s.escDup(ctx, censusEscClass, scope, cfg.dedupH)
		if err != nil {
			return err
		}
		if dup {
			continue
		}
		note := fmt.Sprintf("普查格 %s：verdict=executable net_est=%.1fbps>阈%.0f 腿=%v——"+
			"eval臂该族 template 无评估记录（桌上有菜没点——只述事实，定性归唤知）",
			scope, cd.net, cfg.minNet, rlegs(c.rows, cd.inst, cd.mech))
		if err := s.escInsert(ctx, censusEscClass, scope, note); err != nil {
			return err
		}
		fired++
	}
	return nil
}

// rlegs 行查腿证（note 实据件——legs jsonb 已落 opp_map，此处取内存行）。
func rlegs(rows []censusRow, inst, mech string) string {
	for _, r := range rows {
		if r.inst == inst && r.mechanism == mech {
			b, _ := json.Marshal(r.legs)
			return string(b)
		}
	}
	return "{}"
}

// censusEvald eval 臂「从未评过」判据：fin_opp_eval 同 inst+机制族归一
// template 任一命中=评过（replace(_→-) 双向归一 breedstall 同构）。
func (s *Sensor) censusEvald(ctx context.Context, inst, mech string) (bool, error) {
	tmpls := censusMechTemplates[mech]
	if len(tmpls) == 0 {
		return false, nil
	}
	var hit bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM fin_opp_eval
		 WHERE inst_id=$1 AND replace(template,'_','-')=ANY($2))`,
		inst, tmpls).Scan(&hit)
	return hit, err
}
