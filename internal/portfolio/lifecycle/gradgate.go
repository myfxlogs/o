// gradgate.go —— 毕业闸升级件（T-0521 FinRL 镜照②：时长/净回报/回撤裸阈升
// 置信下界证伪制+族进入条件判词层）。
// 置信下界口径（策略评估与仓位 skill 前提标注正本：N<30 用 Student-t 非正态；
// 事件独立假设——N<30 时游程检验不可行，判词附「独立性未检」降级标注）：
// 收入流按结算笔序列（kind=funding/swap——fee/snapshot 等非收入项不入列；
// fee 是确定性成本由 net≥minNet 兜，不污染随机序列）→
// mean−t(0.975,N−1)·sd/√N>0 才算收入可证伪为正。N<3=样本不足 fail-closed。
// 进入条件判词（按族风险结构判「现在该不该进」——机会仍在条件复满即解，
// 非否决）：
//
//	funding/basis/xvenue carry 族（perp 腿）= 最新资金费率方向判词——
//	  short_perp 须 rate>0（收租位）long_perp 须 rate<0；逆号开仓=进场即
//	  付租；md_funding 行缺=事实缺 fail-closed 延评不虚构。
//	dcd/vol_sell 族=申报 APR 分位判词——最新 annualized_yield 须≥自身
//	  30d 史中位（IV 穷不卖波动率）；史样本缺/最新缺=延评。
package lifecycle

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	confMinFills = 3   // 置信序列最小结算笔数——不足 fail-closed
	volPctileReq = 0.5 // vol-sell 入场 APR 分位线（自史中位）
)

// tQ975 单侧 97.5% Student-t 分位（index=自由度 df=N−1；df>30 收敛正态 1.96）。
// 策略评估与仓位 skill：N<30 小样本用 t 不用 z——z=2.0 在 df=2 时相当于
// t=4.303 的 3 倍放水，是正本钦定的修正面。
var tQ975 = [...]float64{
	0, 12.706, 4.303, 3.182, 2.776, 2.571, 2.447, 2.365, 2.306, 2.262, 2.228,
	2.201, 2.179, 2.160, 2.145, 2.131, 2.120, 2.110, 2.101, 2.093, 2.086,
	2.080, 2.074, 2.069, 2.064, 2.060, 2.056, 2.052, 2.048, 2.045, 2.042,
}

func tQ975At(df int) float64 {
	if df > len(tQ975)-1 {
		return 1.96
	}
	if df < 1 {
		df = 1
	}
	return tQ975[df]
}

// confGate 置信下界证伪闸：收入流按结算笔序列（kind=funding/swap——
// funding 结算笔/dcd 日摊笔=真实观测粒度；fee/snapshot 等非收入项不入列，
// fee 是确定性成本由 net≥minNet 兜）LB=mean−t(0.975,N−1)·sd/√N>0 才放。
// sd 用样本标准差（ddof=1）。返回 (放?, 判词)。
func (r *Runner) confGate(ctx context.Context, strategyID int64) (bool, string) {
	rows, err := r.pool.Query(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind IN ('funding','swap') ORDER BY ts`, strategyID)
	if err != nil {
		return false, fmt.Sprintf("置信序列读败 fail-closed: %v", err)
	}
	var ds []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return false, fmt.Sprintf("置信序列扫败: %v", err)
		}
		ds = append(ds, v)
	}
	rows.Close()
	n := float64(len(ds))
	if n < confMinFills {
		return false, fmt.Sprintf("置信样本不足（收入笔数=%d<%d——证据不够不宣判）", int(n), confMinFills)
	}
	var mean float64
	for _, v := range ds {
		mean += v
	}
	mean /= n
	var sd float64
	for _, v := range ds {
		sd += (v - mean) * (v - mean)
	}
	sd = math.Sqrt(sd / (n - 1))
	tq := tQ975At(int(n) - 1)
	lb := mean - tq*sd/math.Sqrt(n)
	indep := ""
	if n < 30 {
		indep = "独立性未检(N<30)"
	}
	if lb <= 0 {
		return false, fmt.Sprintf("收入流置信下界未过（N=%d mean=%.4f sd=%.4f t=%.3f LB=%.4f≤0 %s）",
			int(n), mean, sd, tq, lb, indep)
	}
	return true, fmt.Sprintf("LB=%.4f>0(N=%d t=%.3f %s)", lb, int(n), tq, indep)
}

// entryCondition 进入条件判词（逆 regime 延评非否决——条件复满即自解）。
// 首判=宏观爆点窗（T-0521 FinRL 镜照①——ctx 采集的 md_macro_event
// 此前只喂飞书提醒（人端），决策面零消费：fomc/cpi/nfp 爆点窗内全族延开——
// 爆点成交=价失位+尾部风险；事件历空=放行记日志（历缺不是闸由——开仓非
// fail-closed 域，断供面归数据缺探针另行显形）。
func (r *Runner) entryCondition(ctx context.Context, inst, tmpl, dir string, refAPR float64) (bool, string) {
	if ok, why := r.macroEventGate(ctx); !ok {
		return false, why
	}
	switch {
	case tmpl == "dcd_call" || tmpl == "dcd_put" ||
		tmpl == "vol_sell_single_call" || tmpl == "vol_sell_strangle":
		return r.volEntryGate(ctx, inst, refAPR)
	default: // perp 腿 carry 族（funding-hedge/basis/xvenue——方向判词）
		return r.fundingEntryGate(ctx, inst, dir)
	}
}

// fundingEntryGate 资金费率方向判词：perp 腿方向×最新 rate 同号才收租——
// 逆号=延评；md_funding 行缺=延评（事实缺不虚构）。
// 非 perp 方向腿（xvenue mt5 cfd 腿 dir=short_cfd/long_cfd 等）=无条件放行——
// 判词只钉在真收付租的 perp 腿上（对侧腿由本提案 perp 行同判词覆盖）。
func (r *Runner) fundingEntryGate(ctx context.Context, inst, dir string) (bool, string) {
	perpShort := strings.HasPrefix(dir, "short_perp")
	perpLong := strings.HasPrefix(dir, "long_perp")
	if !perpShort && !perpLong {
		return true, "非 perp 方向——无资金费率判词"
	}
	var rate float64
	if err := r.pool.QueryRow(ctx, `SELECT funding_rate FROM md_funding
		WHERE inst_id=$1 ORDER BY ts DESC LIMIT 1`, inst).Scan(&rate); err != nil {
		return false, fmt.Sprintf("funding 事实缺（%s 无 rate 行——延评不虚构）", inst)
	}
	if perpShort && rate <= 0 {
		return false, fmt.Sprintf("funding 逆号（%s rate=%.6f≤0 short_perp 收租位失守——延评）", inst, rate)
	}
	if perpLong && rate >= 0 {
		return false, fmt.Sprintf("funding 逆号（%s rate=%.6f≥0 long_perp 付租位——延评）", inst, rate)
	}
	return true, fmt.Sprintf("rate=%.6f 同号", rate)
}

// macroEventGate 宏观爆点日判词：md_macro_event event_date=date 粒度——
// 当日有 High 事件（fomc/cpi/nfp）=全天延开（爆点日成交价失位+尾部风险；
// 日粒度无时刻窗可切——全天延开=保守同语义）。
// 返回 (放?, 判词)；历空/读败=放行不虚构（历缺归数据缺探针显形非闸由）。
func (r *Runner) macroEventGate(ctx context.Context) (bool, string) {
	var kind, title string
	var ev time.Time
	err := r.pool.QueryRow(ctx, `SELECT kind,title,event_date FROM md_macro_event
		WHERE event_date=current_date ORDER BY event_date LIMIT 1`).Scan(&kind, &title, &ev)
	if err != nil {
		return true, "当日无爆点事件（或历缺——放行不虚构）"
	}
	return false, fmt.Sprintf("宏观爆点日（%s %s@%s——事件日延开）", kind, title, ev.Format("01-02"))
}

// volEntryGate vol-sell/dcd 入场判词：inst 键（strangle 复合键拆双腿）各产品
// 最新申报 APR（md_dcd_product 状态行=现值锚）≥提案立项 net_apr×volPctileReq
// ——「进场收益率塌于立项承诺半值不卖波动率」。参照系=提案自记 net_apr：
// APR 史两侧皆状态表无史（md_dcd_product/fin_opp_eval 均 upsert 末帧），
// 不虚构史——立项值即该产品被判定值得卖时的价，塌于半值=IV 穷 regime。
// refAPR≤0=立项值缺=延评；报价行缺=延评（全 fail-closed）。
func (r *Runner) volEntryGate(ctx context.Context, inst string, refAPR float64) (bool, string) {
	if refAPR <= 0 {
		return false, "立项 APR 缺（提案 net_apr 读不到——延评不虚构基准）"
	}
	for _, pid := range strings.Split(inst, "+") {
		var latest float64
		if err := r.pool.QueryRow(ctx, `SELECT annualized_yield FROM md_dcd_product
			WHERE product_id=$1 ORDER BY ts DESC LIMIT 1`, pid).Scan(&latest); err != nil {
			return false, fmt.Sprintf("vol 报价缺（%s 无快照——延评不虚构）", pid)
		}
		if latest < refAPR*volPctileReq {
			return false, fmt.Sprintf("IV 分位穷（%s 现价 APR=%.4f<立项%.4f×%.2f——收益率塌于立项半值不卖）",
				pid, latest, refAPR, volPctileReq)
		}
	}
	return true, "APR 守立项基准"
}
