package execution

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"o/internal/trading/okxexec"
)

// Execute 双腿原子开仓：令牌→live 闸→对冲带→leg1→leg2→（败则 LEGGED 回滚）。
// 返回值只表执行结局（该不该开已过 G6——本件只保证「开成/全收回」）。
func (r *Runner) Execute(ctx context.Context, in Intent) (int64, error) {
	// ①令牌验证为第一件事（四径拒+单次消费原子）。
	tok, err := r.verifyToken(ctx, in.DecisionID)
	if err != nil {
		return 0, err
	}
	// ②live 闸（委员会开关位——开仓闸非退仓闸：close intent 免检，
	// 缺键/非 true fail-closed 只对 open）。
	if in.Kind != "close" {
		live, err := r.getB("invest.lifecycle.live_switch")
		if err != nil || !live {
			r.rej.Inc("live_switch_off")
			return 0, fmt.Errorf("live_switch_off")
		}
	}
	// ③对冲带：双腿名义额差容忍 exec.hedge_band_bps（缺键 fail-closed）。
	band, err := r.getF("exec.hedge_band_bps")
	if err != nil {
		r.rej.Inc("missing_param")
		return 0, fmt.Errorf("missing_param exec.hedge_band_bps")
	}
	l1, l2 := in.Legs[0], in.Legs[1]
	if diff := math.Abs(l1.Notional-l2.Notional) / max(l1.Notional, l2.Notional); diff > band/1e4 {
		r.rej.Inc("hedge_band")
		return 0, fmt.Errorf("hedge_band: 双腿名义差 %.4f > %.4f", diff, band/1e4)
	}
	var execID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES($1,$2,$3,$4,$5,'open') RETURNING id`,
		in.StrategyRef, in.DecisionID, tok, in.InstID, in.NotionalUSD).Scan(&execID); err != nil {
		r.rej.Inc("write_err")
		return 0, err
	}
	// T-0083 腿级现金流对称：现货腿现金行 source_ref=腿行 id+position_ref 归仓
	// ——close 径仓已存按 inst 现读归仓；open 径仓行在双腿成交后建（linkLegCash 补挂）。
	var posRef *int64
	if in.Kind == "close" {
		var pid int64
		if err := r.pool.QueryRow(ctx, `SELECT id FROM fin_position
			WHERE inst_id=$1 AND closed_at IS NULL ORDER BY id DESC LIMIT 1`,
			in.InstID).Scan(&pid); err == nil {
			posRef = &pid
		}
	}
	// ③.5 前置赎回（D-0006 停泊态：闲资默认停 Earn——开仓先赎回 notional+buffer 回拢；
	// 败=fail-closed 不开仓，留痕=exec 行翻 failed+计数门。close 免赎——退仓不进钱径）。
	if r.earn != nil && in.Kind != "close" {
		if err := r.earn.RedeemToTrading(ctx, in.NotionalUSD); err != nil {
			r.setState(ctx, execID, "failed")
			r.rej.Inc("earn_redeem_failed")
			return execID, fmt.Errorf("前置赎回: %w", err)
		}
	}
	// ④leg1
	res1, err := r.placeAndResolve(ctx, execID, 1, l1, posRef)
	if err != nil {
		// T-0145：leg1 歧态势（传输败/live/部分成→Resolve 定不了真态）——
		// venue 可能已成仓（裸腿残留）=告警升件人查，禁静默当败。
		if res1.State == "unknown" || res1.State == "live" || res1.State == "partially_filled" {
			r.alert(ctx, "leg1_ambiguous",
				fmt.Sprintf("exec#%d leg1 %s@%s 歧态势（state=%s）——venue 可能已成仓，残留敞口人查", execID, l1.InstID, l1.Venue, res1.State))
		}
		r.setState(ctx, execID, "failed")
		return execID, fmt.Errorf("leg1: %w", err)
	}
	// ⑤leg2——败/歧义=LEGGED→自动回滚 leg1（不留半成品敞口）。
	res2, err := r.placeAndResolve(ctx, execID, 2, l2, posRef)
	if err != nil || res2.State == "ambiguous" {
		// T-0145：leg2 歧义（回滚 leg1 时 leg2 真态未定——两腿可能同存=
		// 双敞口事故）=告警升件人查定真态。
		if res2.State == "unknown" || res2.State == "live" || res2.State == "partially_filled" {
			r.alert(ctx, "leg2_ambiguous",
				fmt.Sprintf("exec#%d leg2 %s@%s 歧态势（state=%s）——回滚 leg1 后 leg2 实存须人查", execID, l2.InstID, l2.Venue, res2.State))
		}
		r.setState(ctx, execID, "legged")
		r.rollback(ctx, execID, l1, res1)
		// T-0145 返修①：败径 %w 透传——venue APIError（okx/mtexec 终局码）吞不得，
		// xvenue 终局分类靠 errors.As 判（无 %w=untradeable 联动死径，armed 重试
		// 开平 churn 烧真钱）；err==nil 纯歧义分支保原文（无码可透）。
		if err != nil {
			return execID, fmt.Errorf("leg2 败/歧义→LEGGED 回滚 leg1: %w", err)
		}
		return execID, fmt.Errorf("leg2 败/歧义→LEGGED 回滚 leg1")
	}
	// ⑥仓行（delta 中性标——funding 对冲天然）——仅 open 径建（T-0083 返修①：
	// close 径禁建新仓行——被平的是既有仓（guard markClosed 关原行），close 建
	// 行=空壳壳行+cash 错绑病灶源）；linkLegCash 同律只挂 open 径（返修②：
	// close 腿 posRef 已按开仓仓现读对，execution_id 回溯会错绑本 exec 壳行）。
	if in.Kind != "close" {
		var posID int64
		if err := r.pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd)
			VALUES($1,$2,$3,$4) RETURNING id`,
			execID, in.StrategyRef, in.InstID, in.NotionalUSD).Scan(&posID); err != nil {
			r.rej.Inc("write_err")
		} else {
			r.linkLegCash(ctx, execID, posID)
		}
	}
	log.Printf("[execution] #%d %s 双腿成交（clOrdId %s/%s）", execID, in.InstID, res1.ClOrdID, res2.ClOrdID)
	return execID, nil
}

// placeAndResolve 下单+歧义定态（okx-exec 契约：传输败→Resolve 定真态；
// unknown=ambiguous 禁补单）。腿行每拍落 fin_order_leg。
func (r *Runner) placeAndResolve(ctx context.Context, execID int64, no int, leg Leg, posRef *int64) (OrderResult, error) {
	// clOrdId=大小写英数 only（OKX 实拍 51000——T-0038：连字符非法；字母位分隔）。
	clOrdID := fmt.Sprintf("ph%dx%dx%d", execID, no, time.Now().UnixNano()%1e6)
	// T-0238 maker 优先径：exec.maker_enabled 且 okx 腿且盘口鲜活→post_only
	// 挂价等时；时限/未成/拒→取消定态降级 market（成=低费，败=原径无损）。
	if mres, ok := r.makerAttempt(ctx, execID, no, leg, posRef); ok {
		if mres.State != "partial" {
			return mres, nil
		}
		leg = scaleLegRemainder(leg, mres) // 碎量余腿缩放（maker.go——行棘轮拆位）
	}
	res, err := r.exec.Place(ctx, leg, clOrdID)
	// envelope 业务拒（*APIError）分码族定态（T-0042 返修）：
	// 51xxx=参数/业务拒族——确定单未建，absent+拒码进 leg 行 venue_err_*；
	// 非 51xxx（50004 超时/50011 限速/50013 系统忙等）——单或已建=歧态势，
	// 留痕后仍走 Resolve 定真态（禁直接 absent）。
	var vc, vm string
	var ae *okxexec.APIError
	if errors.As(err, &ae) {
		vc, vm = ae.Code, ae.Msg
		if ae.SCode != "" {
			// 内层真码优先（envelope code=1「All operations failed」壳+data[0]
			// sCode=真因——T-0145 返修①：51 族判与 leg 留痕同取内层码）。
			vc, vm = ae.SCode, orVal(ae.SMsg, ae.Msg)
		}
		if strings.HasPrefix(vc, "51") {
			r.leg(ctx, execID, no, leg, clOrdID, "failed", vc, vm)
			return OrderResult{ClOrdID: clOrdID, State: "absent"}, fmt.Errorf("venue 拒: %w", err)
		}
	}
	perr := err // Place 原 err 留存——Resolve 重赋后兜底透传（venue APIError 禁吞）
	if err != nil {
		// 传输败/非族 envelope 拒→clOrdId 定态三分支；查询亦败=ambiguous 禁补单
		//（拒码 vc/vm 透传留痕——禁走 err 径丢码）。
		res, err = r.exec.Resolve(ctx, leg, clOrdID)
		if err != nil || res.State == "unknown" {
			r.leg(ctx, execID, no, leg, clOrdID, "ambiguous", vc, vm)
			return res, fmt.Errorf("ambiguous")
		}
	}
	switch res.State {
	case "filled":
		legID := r.legPx(ctx, execID, no, leg, clOrdID, "filled", vc, vm, res.FillPx, res.FillSz, "market", res.FillTicket)
		r.legCash(ctx, legID, leg, res.FillPx, res.FillSz, posRef, res.Fee, res.FeeCcy)
		return res, nil
	case "live", "partially_filled":
		r.leg(ctx, execID, no, leg, clOrdID, "ack", vc, vm)
		return res, fmt.Errorf("未即时成交（live 确认轨未落）")
	default:
		// T-0046 ⑤：envelope 拒码兜底——Resolve 后 absent 态 res 无码（Resolve 不产
		// 拒码），先 Place 已抽的 vc/vm 顶上，禁留空吞真拒因。
		r.leg(ctx, execID, no, leg, clOrdID, "failed", orVal(res.VenueCode, vc), orVal(res.VenueMsg, vm))
		// T-0145 返修①：Place 原 err %w 透传——mtexec.APIError/非 51 envelope 拒走
		// Resolve=absent 径原码吞不得（终局分类靠 err 链 errors.As 判）。
		if perr != nil {
			return res, fmt.Errorf("state=%s: %w", res.State, perr)
		}
		return res, fmt.Errorf("state=%s", res.State)
	}
}

// orVal 首非空值（venue_err 兜底装配）。
func orVal(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (r *Runner) setState(ctx context.Context, execID int64, state string) {
	if _, err := r.pool.Exec(ctx, "UPDATE fin_execution SET state=$2 WHERE id=$1", execID, state); err != nil {
		r.rej.Inc("write_err")
	}
}
