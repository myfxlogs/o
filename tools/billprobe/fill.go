package main

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// billCand 回填候选行（fin_venue_bill 列子集+occurred_at 原表 ts）。
type billCand struct {
	id      int64
	billID  string
	instID  string
	ccy     string
	subType string
	balChg  float64
	fee     float64
	ts      time.Time
}

// fillOne 单腿补记——镜像 funding.spotNative 口径（口径变更须同步）：
// gross=balChg−fee；kind 按币对角色定向（基币入=买/出=卖，计价币入=卖/出=买）；
// fee 腿按扣款币另记；币对解不出=unmapped_bill 残差。position_ref 同
// post() 径——USDT/USD 归仓窗查（venue 侧现货无仓=NULL 如实）。
// 幂等=分量级：gross 域（spot_*/unmapped_bill）与 fee 域各自查存量——
// 历史半残行（fee 在账 gross 缺）只补缺分量不双计。covered/feeCovered
// 镜像 spotQuote 判别（legCash∨residual_clear gross 域；legCash fee 域）。
// 返新增行数。
func fillOne(ctx context.Context, pool *pgxpool.Pool, c billCand, covered, feeCovered bool) (int, error) {
	gross := c.balChg - c.fee
	kind := ""
	if parts := strings.SplitN(c.instID, "-", 2); len(parts) == 2 {
		inBase := c.ccy == parts[0]
		if inBase || c.ccy == parts[1] {
			buy := (inBase && gross > 0) || (!inBase && gross < 0)
			if buy {
				kind = "spot_buy"
			} else {
				kind = "spot_sell"
			}
		}
	}
	has := func(kinds ...string) (bool, error) {
		var ok bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_cashflow
			WHERE source_table='fin_venue_bill' AND source_ref=$1 AND kind=ANY($2))`,
			c.id, kinds).Scan(&ok)
		return ok, err
	}
	post := func(k string, amt float64) error {
		var posRef *int64
		if c.ccy == "USDT" || c.ccy == "USD" {
			var pid int64
			if err := pool.QueryRow(ctx, `SELECT id FROM fin_position
				WHERE inst_id=$1 AND opened_at<=$2 AND (state<>'closed' OR closed_at>=$2)
				ORDER BY id DESC LIMIT 1`, c.instID, c.ts).Scan(&pid); err == nil {
				posRef = &pid
			}
		}
		_, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
			VALUES($1,'okx',$2,$3,$4,$5,'fin_venue_bill',$6)`,
			k, c.ccy, amt, posRef, c.id, c.ts)
		return err
	}
	added := 0
	// gross 分量：!covered（无 legCash/residual_clear 覆盖）才补——
	// kind 解出=gross!=0 时记 spot_*；kind 解不出=unmapped_bill 残差。
	// 存量判据=同 bill 已有任一 gross 域行（spot_buy/spot_sell/unmapped_bill）。
	if !covered && kind != "" && gross != 0 {
		ex, err := has("spot_buy", "spot_sell", "unmapped_bill")
		if err != nil {
			return added, err
		}
		if !ex {
			if err := post(kind, gross); err != nil {
				return added, err
			}
			added++
		}
	} else if !covered && kind == "" {
		ex, err := has("unmapped_bill")
		if err != nil {
			return added, err
		}
		if !ex {
			if err := post("unmapped_bill", c.balChg); err != nil {
				return added, err
			}
			added++
		}
	}
	if c.fee != 0 && !feeCovered {
		ex, err := has("fee")
		if err != nil {
			return added, err
		}
		if !ex {
			if err := post("fee", c.fee); err != nil {
				return added, err
			}
			added++
		}
	}
	return added, nil
}
