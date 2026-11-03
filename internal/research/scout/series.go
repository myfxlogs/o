// series.go —— scout 度量序列件（行数闸拆分）：canon 归一+事实行加载+
// 逐日净 APR 序列+中位/占比算子（T-0300 返修①——metrics.go 判据面读此）。
package scout

import (
	"context"
	"strings"
	"time"
)

// canon 符号归一（剥方言→base ccy——返修①：BTCUSDT/BTC-USDT-SWAP/BTCUSD/
// 1000PEPEUSDT 全归 BTC/PEPE；join 语义键，inst 原值仍随行留）。
func canon(s string) string {
	s = strings.ToUpper(s)
	for _, suf := range []string{"-SWAP", "-PERP", "_PERP", "SWAP", "PERP"} {
		s = strings.TrimSuffix(s, suf)
	}
	s = strings.NewReplacer("-", "", "_", "").Replace(s)
	for _, q := range []string{"USDT", "USDC", "USD"} {
		if strings.HasSuffix(s, q) && len(s) > len(q) {
			s = s[:len(s)-len(q)]
			break
		}
	}
	i := 0
	for i < len(s)-1 && s[i] >= '0' && s[i] <= '9' {
		i++ // 1000PEPE 类数量前缀剥除（全数字名保留防剥空）
	}
	return s[i:]
}

// instRows (canon,day)→最新 rate 读件。
type instRows map[string]map[string]float64 // canon → day → rate

func loadVenue(ctx context.Context, r *Runner, venue string) (instRows, error) {
	out := instRows{}
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}
	var err error
	if venue == "okx" { // 返修②：OKX 基准行读 md_funding 正本（非 scout 自采行）
		rows, err = r.pool.Query(ctx, `SELECT inst_id, funding_rate, date_trunc('day',ts) FROM
			(SELECT DISTINCT ON (inst_id,date_trunc('day',ts)) inst_id,funding_rate,ts
			 FROM md_funding WHERE funding_rate IS NOT NULL
			 AND ts>now()-interval '30 days'
			 ORDER BY inst_id,date_trunc('day',ts),ts DESC) t`)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT inst, funding_rate, date_trunc('day',ts) FROM
			(SELECT DISTINCT ON (inst,date_trunc('day',ts)) inst,funding_rate,ts
			 FROM md_venue_funding WHERE venue=$1 AND funding_rate IS NOT NULL
			 AND ts>now()-interval '30 days'
			 ORDER BY inst,date_trunc('day',ts),ts DESC) t`, venue)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var inst string
		var rate float64
		var day time.Time
		if err := rows.Scan(&inst, &rate, &day); err != nil {
			return nil, err
		}
		c := canon(inst)
		if out[c] == nil {
			out[c] = map[string]float64{}
		}
		out[c][day.Format("2006-01-02")] = rate
	}
	return out, rows.Err()
}

// dailySeries 日值序列：每 day 各 canon 净值→top-N 中位（netFn=rate→net 映射）。
func dailySeries(m instRows, topN int, netFn func(rate float64) float64) []float64 {
	days := map[string][]float64{}
	for _, dm := range m {
		for d, rate := range dm {
			days[d] = append(days[d], netFn(rate))
		}
	}
	var series []float64
	for _, d := range sortedKeys(days) {
		series = append(series, midTopN(days[d], topN))
	}
	return series
}

// midTopN top-N 中位（降序取前 N 的中位值——骨架期确定性度量）。
func midTopN(xs []float64, n int) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] > s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	if len(s) > n {
		s = s[:n]
	}
	return s[len(s)/2]
}

// med 序列中位（升序排后取中）。
func med(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	return s[len(s)/2]
}

// fracOver 序列>baseline 占比。
func fracOver(xs []float64, base float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	n := 0
	for _, x := range xs {
		if x > base {
			n++
		}
	}
	return float64(n) / 30.0 // 30d 窗归一分母（缺测日按未达标计——保守面）
}

// sortedKeys map 键升序（日序列确定性序）。
func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	for i := 0; i < len(ks); i++ {
		for j := i + 1; j < len(ks); j++ {
			if ks[j] < ks[i] {
				ks[i], ks[j] = ks[j], ks[i]
			}
		}
	}
	return ks
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
