// basis.go —— basis 族 paper 记账面（T-0216 返修①补装——quarter_basis_long/
// perp_quarter_spread 入 paperTemplates 而无记账径=paper 恒零 fill 毕业判据
// 永假；本件=记账循环+腿键归一链，与 paper.go 三族面分文件守行数棘轮）。
package lifecycle

import (
	"context"
	"regexp"
	"strings"
	"time"
)

// basisRe 到期日期货键（-YYMMDD 尾）——基键派生现货/perp 腿+距到期解析
// （grammar.datedFutRe 同形双组：m[1]=基键 m[2]=YYMMDD）。
var basisRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z0-9]+)-([0-9]{6})$`)

// basisSpotLeg 期货基键→USDT 现货对（同 activeSet/dcd 归一链：
// BTC-USD-260626→BTC-USD→BTC-USDT；-USDG 指数键同归一；USDT 本位基键原形保）。
func basisSpotLeg(futInst string) string {
	m := basisRe.FindStringSubmatch(futInst)
	if m == nil {
		return ""
	}
	base := m[1]
	switch {
	case strings.HasSuffix(base, "-USDG"):
		base = strings.TrimSuffix(base, "-USDG") + "-USDT"
	case strings.HasSuffix(base, "-USD"):
		base = strings.TrimSuffix(base, "-USD") + "-USDT"
	}
	return base
}

// basisPaperAccounting basis 族 paper 记账（T-0216 返修①补装——quarter_basis_long/
// perp_quarter_spread 入 paperTemplates 而无记账径=paper 恒零 fill 毕业判据永假）。
// 收入面=基差溢价线性衰减摊提：amt = notional × p/days_left × elapsed
// （p=(fut−spot)/spot 现价重算——线性衰减下 p/days_left 恒=R₀/T，跨拍
// telescoping 收敛全溢价；elapsed 夹到 days_left=断窗后至多补记全剩余溢价；
// sign 随结构如实——contango 正收益/backwardation 负如实）。
// 价源（T-0171 律——价格消费只准 WS 流径行）：期货腿=md_mark_index mark-price
// （桥面行——paper 策略 inst 在投影活跃集谓词内）；现货腿=md_ticks 本地行
// （activeSet 已补 basis 现货腿——T-0216 返修同票），exec.tick_max_age_s 鲜度闸
// 同 dcd 律；任一腿缺/陈=fail-closed 让位不造零。
// perp_quarter_spread 加计 perp 腿 funding：md_funding 行自上次 funding fill 起
// 逐行 −rate×notional（long_perp 付正租——sign 翻转，paperAccounting 注同源）。
func (r *Runner) basisPaperAccounting(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.inst_id, s.notional_usd, s.template_ref,
		COALESCE((SELECT max(ts) FROM fin_paper_fill f
			WHERE f.strategy_id=s.id AND f.inst_id=s.inst_id), s.entered_at)
		FROM fin_strategy s WHERE s.state='paper'
		AND s.template_ref IN ('quarter_basis_long','perp_quarter_spread')`)
	if err != nil {
		return err
	}
	type st struct {
		id       int64
		inst     string
		notional float64
		tmpl     string
		last     time.Time
	}
	var sts []st
	for rows.Next() {
		var x st
		if err := rows.Scan(&x.id, &x.inst, &x.notional, &x.tmpl, &x.last); err != nil {
			rows.Close()
			return err
		}
		sts = append(sts, x)
	}
	rows.Close()
	maxAge := 120.0
	if v, err := r.cfg.GetFloat("exec.tick_max_age_s"); err == nil && v > 0 {
		maxAge = v
	}
	fresh := func(ts time.Time) bool {
		return time.Since(ts) <= time.Duration(maxAge*float64(time.Second))
	}
	for _, x := range sts {
		m := basisRe.FindStringSubmatch(x.inst)
		if m == nil {
			continue // 非到期日期货键=装配错位——不记账不虚构
		}
		exp, perr := time.Parse("060102", m[2])
		if perr != nil {
			continue // 到期尾非日期形——装配错位不虚构
		}
		daysLeft := time.Until(exp).Hours() / 24
		if daysLeft <= 0 {
			continue // 到期/过期末——溢价收敛已尽，无可摊提
		}
		// 期货腿价=mark-price 桥面行（freshness 闸同律）
		var fut float64
		var futTs time.Time
		if err := r.pool.QueryRow(ctx, `SELECT px,ts FROM md_mark_index
			WHERE inst_id=$1 AND channel='mark-price' ORDER BY ts DESC LIMIT 1`, x.inst).Scan(&fut, &futTs); err != nil || fut <= 0 || !fresh(futTs) {
			r.rej.Inc("basis_mark_stale")
			continue
		}
		// 现货腿价=本地 tickers 行（activeSet 补订——T-0216 返修）
		spotInst := basisSpotLeg(x.inst)
		if spotInst == "" {
			continue
		}
		var spot float64
		var spotTs time.Time
		if err := r.pool.QueryRow(ctx, `SELECT last,ts FROM md_ticks
			WHERE inst_id=$1`, spotInst).Scan(&spot, &spotTs); err != nil || spot <= 0 || !fresh(spotTs) {
			r.rej.Inc("basis_mark_stale")
			continue
		}
		elapsed := time.Since(x.last).Hours() / 24
		if elapsed < 1.0 {
			continue // 日摊节拍同 dcd 径
		}
		if elapsed > daysLeft {
			elapsed = daysLeft // 断窗夹取：至多补记全剩余溢价（不超收）
		}
		p := (fut - spot) / spot
		amt := x.notional * p / daysLeft * elapsed
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
			VALUES($1,'funding',$2,$3,$4,now())`,
			x.id, x.inst, amt,
			[]byte(`{"p":`+ftoa(p)+`,"fut":`+ftoa(fut)+`,"spot":`+ftoa(spot)+
				`,"days_left":`+ftoa(daysLeft)+`}`)); err != nil {
			return err
		}
		// perp 腿 funding（perp_quarter_spread 专属）——inst_id=perp 键与溢价
		// 摊提 fill 分键（互不污染双方 last-ts 基准）。
		if x.tmpl == "perp_quarter_spread" {
			perpInst := m[1] + "-SWAP"
			var flast time.Time
			if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT max(ts) FROM fin_paper_fill f
				WHERE f.strategy_id=$1 AND f.kind='funding' AND f.inst_id=$2), s.entered_at)
				FROM fin_strategy s WHERE s.id=$1`, x.id, perpInst).Scan(&flast); err != nil {
				continue // perp funding 基准读败=让位本拍（溢价摊提已落不回滚）
			}
			frows, err := r.pool.Query(ctx, `SELECT ts, funding_rate FROM md_funding
				WHERE inst_id=$1 AND ts>$2 ORDER BY ts`, perpInst, flast)
			if err != nil {
				continue
			}
			for frows.Next() {
				var ts time.Time
				var rate float64
				if err := frows.Scan(&ts, &rate); err != nil {
					frows.Close()
					break
				}
				if _, err := r.pool.Exec(ctx, `INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,detail,ts)
					VALUES($1,'funding',$2,$3,$4,$5)`,
					x.id, perpInst, -rate*x.notional,
					[]byte(`{"rate":`+ftoa(rate)+`,"leg":"perp_long"}`), ts); err != nil {
					frows.Close()
					break
				}
			}
			frows.Close()
		}
	}
	return nil
}
