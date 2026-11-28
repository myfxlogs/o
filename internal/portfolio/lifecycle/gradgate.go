// gradgate.go —— 毕业闸升级件（T-0521 FinRL 镜照②：时长/净回报/回撤裸阈升
// 置信下界证伪制+族进入条件判词层）。
// 置信下界口径（策略评估与仓位 skill 前提标注正本：N<30 用 Student-t 非正态；
// 事件独立假设——N≥30 符号游程检验 Z<−1.96=同向聚簇证伪 fail，N<30 判词附
// 「证据期」显式行；O-41389 硬化）：
// 收入流按结算笔序列（kind=funding/swap/pnl——fee/snapshot 等非收入项不入列；
// fee 是确定性成本由 net≥minNet 兜，不污染随机序列）→
// mean−t(0.975,N−1)·sd/√N>0 才算收入可证伪为正。N<5=样本不足 fail-closed；
// sd=0 零方差序列须 N≥8 方放（杀三笔等额正秒过捷径）。
// 进入条件判词（按族风险结构判「现在该不该进」——机会仍在条件复满即解，
// 非否决）：
//
//	funding/basis/xvenue carry 族（perp 腿）= 最新资金费率方向判词——
//	  short_perp 须 rate>0（收租位）long_perp 须 rate<0；逆号开仓=进场即
//	  付租；md_funding 行缺=事实缺 fail-closed 延评不虚构。
//	dcd/vol_sell 族=申报 APR 分位判词——最新 annualized_yield 须≥自身
//	  30d 史中位（IV 穷不卖波动率）；史样本缺/最新缺=延评。
//	dir 方向族（dir_perp_mom/perp-trend-daily/O-35489 三族骨架）=信号续存
//	  判词——本向最新评估行仍 proposed 且 72h 内（毕业再进场=信念续存核验；
//	  信号弱化/翻面=延评）。
package lifecycle

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	confMinFills   = 5     // 置信序列最小结算笔数——不足 fail-closed（O-41389：3→5）
	confMinZeroVar = 8     // 零方差序列最小笔数——sd=0 同值流须 N≥8 方放
	runsMinN       = 30    // 独立性游程检验最小样本——不足判词升「证据期」显式行
	runsZCrit      = -1.96 // 游程检验单侧 5% 下牙——Z<此值=同向聚簇=独立假设证伪
	volPctileReq   = 0.5   // vol-sell 入场 APR 分位线（自史中位）
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

// dirAlphaTmpl 方向动量族=价变 pnl 系 alpha 流（O-36961 取证实锤：生产
// 259 paper 中 256=dir_perp_mom，confGate 混采 funding+swap+pnl——funding
// 租金系 carry 附带流非该族 alpha，混入使 LB 测的是租金符号非动量；
// 分流后置信序列=alpha 本体。funding-extreme-fade 例外留混采——其 alpha
// =资金费率极值回归，funding 收租本身即收入流成分非附带噪声）。
var dirAlphaTmpl = map[string]bool{
	"dir_perp_mom": true, "mt5_dir_mom": true, "perp-trend-daily": true,
	"xsec-perp-mom-basket": true, "mark-last-disloc-fade": true,
}

// confGate 置信下界证伪闸：收入流按结算笔序列（kind=funding/swap/pnl——
// funding 结算笔/dcd 日摊笔/方向 MTM 摊提笔=真实观测粒度；fee/snapshot/
// open/close 等非收入项不入列，fee 是确定性成本由 net≥minNet 兜）。
// O-36961 族分流：dirAlphaTmpl 族只采 kind='pnl'（价变 alpha 本体——
// funding 租金属 carry 附带流，混采稀释 alpha 信号）；余族维持混采
// （basis/dcd/stakecarry 族 alpha 流本就写 funding kind，xvenue 写 swap）。
// LB=mean−t(0.975,N−1)·sd/√N>0 才放。sd 用样本标准差（ddof=1）。
// 返回 (放?, 判词)。
func (r *Runner) confGate(ctx context.Context, strategyID int64, tmpl string) (bool, string) {
	kinds := "'funding','swap','pnl'"
	stream := "funding+swap+pnl"
	if dirAlphaTmpl[tmpl] {
		kinds = "'pnl'"
		stream = "pnl"
	}
	rows, err := r.pool.Query(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind IN (`+kinds+`) ORDER BY ts, id`, strategyID)
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
		return false, fmt.Sprintf("置信样本不足（%s 收入笔数=%d<%d——证据不够不宣判）", stream, int(n), confMinFills)
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
	// O-41389①零方差条款：sd=0 同值序列=regime 巧合嫌疑——须 N≥8 方放，
	// N<8 fail-closed「零方差捷径」（三笔等额正秒过捷径根修）。
	if sd == 0 && n < confMinZeroVar {
		return false, fmt.Sprintf("零方差捷径（%s 同值序列 N=%d<%d sd=0——等额流须%d连方放）",
			stream, int(n), confMinZeroVar, confMinZeroVar)
	}
	tq := tQ975At(int(n) - 1)
	lb := mean - tq*sd/math.Sqrt(n)
	// O-41389②独立性上牙：N≥30 符号游程检验——同向聚簇=事件独立假设证伪，
	// LB 按独立样本系统性高估置信→fail；N<30 判词升显式「证据期」行。
	indep, indepFail := runsCheck(ds)
	if indepFail != "" {
		return false, fmt.Sprintf("%s（%s N=%d——不宣判）", indepFail, stream, int(n))
	}
	if lb <= 0 {
		return false, fmt.Sprintf("收入流置信下界未过（%s N=%d mean=%.4f sd=%.4f t=%.3f LB=%.4f≤0 %s）",
			stream, int(n), mean, sd, tq, lb, indep)
	}
	return true, fmt.Sprintf("LB=%.4f>0(%s N=%d t=%.3f %s)", lb, stream, int(n), tq, indep)
}

// runsCheck 符号游程检验（O-41389②独立性上牙）：N≥runsMinN 对正/非正
// （零计入非正）符号序列算游程数 R——E[R]=2n₁n₂/N+1，
// Var=2n₁n₂(2n₁n₂−N)/(N²(N−1))，Z=(R−E)/√Var；Z<runsZCrit=游程显著过少=
// 同向聚簇=独立假设下 LB 高估置信→fail 判词。n₁∨n₂=0（全同号）检验退化=
// 按零方差条款判放行注记（同值嫌疑归 sd=0 条款管辖）。N<runsMinN=证据期——
// 检验未达样本阈，判词升显式行。返回 (判词注记, fail判词)——fail 非空即拦。
func runsCheck(ds []float64) (note, failMsg string) {
	n := len(ds)
	if n < runsMinN {
		return fmt.Sprintf("证据期——独立性检验未达样本阈(N<%d)，LB按独立样本计", runsMinN), ""
	}
	n1, n2, runs := 0, 0, 0
	prevPos := false
	for i, v := range ds {
		pos := v > 0
		if i == 0 || pos != prevPos {
			runs++
		}
		if pos {
			n1++
		} else {
			n2++
		}
		prevPos = pos
	}
	if n1 == 0 || n2 == 0 {
		return fmt.Sprintf("独立性退化（全同号 N=%d——游程不可算，按零方差条款判）", n), ""
	}
	nf := float64(n)
	e := 2*float64(n1)*float64(n2)/nf + 1
	variance := 2 * float64(n1) * float64(n2) * (2*float64(n1)*float64(n2) - nf) / (nf * nf * (nf - 1))
	z := (float64(runs) - e) / math.Sqrt(variance)
	if z < runsZCrit {
		return "", fmt.Sprintf("独立性证伪（runs=%d Z=%.2f——同向聚簇，LB按独立样本高估置信）", runs, z)
	}
	return fmt.Sprintf("独立性过（runs=%d Z=%.2f）", runs, z), ""
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
	case tmpl == "dir_perp_mom" || tmpl == "perp-trend-daily" ||
		tmpl == "xsec-perp-mom-basket" || tmpl == "funding-extreme-fade" ||
		tmpl == "mark-last-disloc-fade":
		// O-35489 三族骨架同判词——eval→proposal→paper 链评估驱动族，
		// 信号续存核验同构（篮族=再平衡信号续存；eval 行缺=延评不虚构）。
		return r.dirEntryGate(ctx, inst, tmpl, dir)
	default: // perp 腿 carry 族（funding-hedge/basis/xvenue——方向判词）
		return r.fundingEntryGate(ctx, inst, dir)
	}
}

// dirEntryGate 方向族入场判词：本向最新 fin_opp_eval 行仍 verdict='proposed'
// 且 72h 内（信号续存核验——评估臂每拍覆写，陈行=臂停/断供分型延评不虚构；
// 行缺/已拒=延评——信念消亡不进）。
func (r *Runner) dirEntryGate(ctx context.Context, inst, tmpl, dir string) (bool, string) {
	var v string
	var ts time.Time
	if err := r.pool.QueryRow(ctx, `SELECT verdict,ts FROM fin_opp_eval
		WHERE inst_id=$1 AND template=$2 AND direction=$3
		ORDER BY ts DESC LIMIT 1`, inst, tmpl, dir).Scan(&v, &ts); err != nil {
		return false, fmt.Sprintf("方向评估行缺（%s %s 无 eval——延评不虚构）", inst, dir)
	}
	if time.Since(ts) > 72*time.Hour {
		return false, fmt.Sprintf("方向评估行陈（%s %s 末评 %s>72h——臂停/断供延评）", inst, dir, ts.Format("01-02 15:04"))
	}
	if v != "proposed" {
		return false, fmt.Sprintf("信号弱化（%s %s 最新判词=%s——信念消亡延评）", inst, dir, v)
	}
	return true, "信号续存"
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
