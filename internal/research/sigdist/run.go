// Package sigdist —— 桥二期信号蒸馏（T-0139/D-0022）：采集拍后由桥面算
// 三族信号 upsert md_signal（有界状态表同键最新赢）——SG 假想官消费的
// 第一批燃料。本件=REST 计算族（basis_ann/funding_view）；depth5=WS 帧
// 蒸馏在 okxws Collector（books5 复活域）。
//
// 信号=派生面非事实源（D-0022 契约钉死）：消费必过 Fresh 闸（陈=缺席
// 如实，禁回退行情现算冒充）；信号永不触执行径。
package sigdist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

// distillPeriod 蒸馏拍周期（basis/funding 均为慢特征——60s 鲜度足够假想
// 输入；Fresh 闸窗=2×拍 同款口径）。
const distillPeriod = 60 * time.Second

// datedFutRe 季度期货 instId 尾 6 位到期日（YYMMDD）。
var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+)-USDT-([0-9]{6})$`)

// Runner 信号蒸馏岗（桥面常驻节拍——随采集器进程）。
type Runner struct {
	pool *pgxpool.Pool
	cfg  *config.Client // T-0141 跨所族映射读件（AttachConfig 可选）
	rej  *rejects.Door
}

// New 构造。
func New(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool,
		rej: rejects.New("sigdist", "basis_read_failed", "funding_read_failed", "sig_write_err")}
}

// Run 在岗（蒸馏节拍——写协程语义同桥面：失败计 rejects 下一拍再算，fail-closed 不造零）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "信号蒸馏", "data",
		"桥面三族信号蒸馏岗（basis_ann/funding_view/depth5→md_signal，D-0022）", nil); err != nil {
		errc <- fmt.Errorf("信号蒸馏 register: %w", err)
		return
	}
	t := time.NewTicker(distillPeriod) // sched-exempt-ok 桥面蒸馏拍（慢特征计算节拍非轮询拉取）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.Distill(ctx); err != nil {
				log.Printf("[sigdist] 蒸馏拍让位: %v", err)
			}
		}
	}
}

// Distill 一拍两族（depth5 帧驱动不在拍内）。
func (r *Runner) Distill(ctx context.Context) error {
	if err := r.basisAnn(ctx); err != nil {
		r.rej.Inc("basis_read_failed")
		return fmt.Errorf("basis_ann: %w", err)
	}
	if err := r.fundingView(ctx); err != nil {
		r.rej.Inc("funding_read_failed")
		return fmt.Errorf("funding_view: %w", err)
	}
	if err := r.Xvenue(ctx); err != nil { // T-0141 跨所族（映射缺=让位不炸 OKX 族）
		log.Printf("[sigdist] xvenue 族让位: %v", err)
	}
	return nil
}

// basisAnn 季度基差年化（bps）：mark-price（md_mark_index channel='mark_price'
// FUTURES inst）对同基币 spot_last——(mark-index)/index×365/到期日数×10⁴。
// 无到期解析的 inst 跳过如实。
func (r *Runner) basisAnn(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT f.inst_id, f.px, s.px FROM md_mark_index f
		JOIN md_mark_index s
		  ON s.channel='spot_last'
		 AND s.inst_id = regexp_replace(f.inst_id, '-[0-9]{6}$', '')
		WHERE f.channel='mark_price' AND f.inst_id ~ '-[0-9]{6}$'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	now := time.Now()
	n := 0
	for rows.Next() {
		var instID string
		var mark, index float64
		if err := rows.Scan(&instID, &mark, &index); err != nil {
			return err
		}
		m := datedFutRe.FindStringSubmatch(instID)
		if m == nil || index == 0 {
			continue // 无到期解析/零指数——跳过如实
		}
		exp, err := time.Parse("060102", m[2])
		if err != nil {
			continue // 到期日坏形——跳过如实
		}
		days := exp.Sub(now).Hours() / 24
		if days <= 0 {
			continue // 到期邻近/已过——基差年化无意义
		}
		bps := (mark - index) / index * 365 / days * 10000
		meta, _ := json.Marshal(map[string]any{
			"expiry": exp.Format("2006-01-02"), "mark": mark, "index": index,
			"days": int(days) + 1})
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES('basis_ann',$1,$2,$3::jsonb,$4)
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
			meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
			instID, bps, string(meta), now); err != nil {
			return err
		}
		n++
	}
	log.Printf("[sigdist] basis_ann 拍：%d 行", n)
	return nil
}

// fundingView per SWAP 当期 funding rate（md_funding 最新行）+meta 附下一期
// 预测（next_funding_rate venue 域内含）+7d 滚动均值（窗口缺数据=字段缺席如实）。
func (r *Runner) fundingView(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst_id)
			inst_id, ts, funding_rate, next_funding_rate, next_funding_ts
		FROM md_funding ORDER BY inst_id, ts DESC`)
	if err != nil {
		return err
	}
	type fv struct {
		inst  string
		ts    time.Time
		rate  float64
		nextR *float64
		nextT *time.Time
		avg7d *float64
	}
	vals := map[string]*fv{}
	for rows.Next() {
		var f fv
		var nextR *float64
		var nextT *time.Time
		if err := rows.Scan(&f.inst, &f.ts, &f.rate, &nextR, &nextT); err != nil {
			rows.Close()
			return err
		}
		f.nextR, f.nextT = nextR, nextT
		vals[f.inst] = &f
	}
	rows.Close()
	avg, err := r.pool.Query(ctx, `SELECT inst_id, AVG(funding_rate) FROM md_funding
		WHERE ts > now()-interval '7 days' GROUP BY inst_id`)
	if err != nil {
		return err
	}
	defer avg.Close()
	for avg.Next() {
		var inst string
		var a float64
		if err := avg.Scan(&inst, &a); err != nil {
			return err
		}
		if f, ok := vals[inst]; ok {
			f.avg7d = &a
		}
	}
	avg.Close()
	n := 0
	now := time.Now()
	for inst, f := range vals {
		meta := map[string]any{}
		if f.nextR != nil {
			meta["next_funding_rate"] = *f.nextR
		}
		if f.nextT != nil {
			meta["next_funding_ts"] = f.nextT.UTC().Format(time.RFC3339)
		}
		if f.avg7d != nil {
			meta["avg_7d"] = *f.avg7d
		}
		metaB, _ := json.Marshal(meta)
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
			VALUES('funding_view',$1,$2,$3::jsonb,$4)
			ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
			meta=EXCLUDED.meta, ts=EXCLUDED.ts`,
			inst, f.rate, string(metaB), now); err != nil {
			return err
		}
		n++
	}
	log.Printf("[sigdist] funding_view 拍：%d 行", n)
	return nil
}

// Sig 新鲜信号值（消费闸出口——陈/缺席=ok false 如实）。
type Sig struct {
	Kind string
	Key  string
	Val  float64
	Meta map[string]any
}

// Fresh 读闸：信号 ts 超 2×蒸馏拍=缺席（陈信号当事实=假想官吃陈价——禁）。
func Fresh(ctx context.Context, pool *pgxpool.Pool, kind, key string) (Sig, bool, error) {
	var s Sig
	var metaB []byte
	err := pool.QueryRow(ctx, `SELECT val, meta, ts FROM md_signal
		WHERE kind=$1 AND key=$2 AND ts > now()-interval '120 seconds'`,
		kind, key).Scan(&s.Val, &metaB, new(time.Time))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, false, nil // 缺席/陈——如实（调用方零区分：皆=信号不在）
		}
		return s, false, err // 读败声张（fail-closed 非静默缺席）
	}
	s.Kind, s.Key = kind, key
	s.Meta = map[string]any{}
	_ = json.Unmarshal(metaB, &s.Meta)
	return s, true, nil
}

// freshByKind 全行读闸（kind 域新鲜信号——ts 超 2×蒸馏拍=缺席）。
func (r *Runner) freshByKind(ctx context.Context, kind string) (map[string]Sig, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, val, meta FROM md_signal
		WHERE kind=$1 AND ts > now()-interval '120 seconds'`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Sig{}
	for rows.Next() {
		var s Sig
		var metaB []byte
		if err := rows.Scan(&s.Key, &s.Val, &metaB); err != nil {
			return nil, err
		}
		s.Kind = kind
		s.Meta = map[string]any{}
		_ = json.Unmarshal(metaB, &s.Meta)
		out[s.Key] = s
	}
	return out, rows.Err()
}

// SurfaceSignals gap surface→信号绑定语义映射（T-0140 键域修正正解——
// surface=能力账面名域，非 instId；映射：
//
//	funding 面→funding_view 榜 |val| top3（carry 候选燃料）；
//	quarter/basis 面→basis_ann 全行；
//	book/depth 面→depth5 全行（桥面标的集=armed 域，配置时已窄）。
//
// 陈=缺席如实（Fresh 闸同款 120s 窗）。
func (r *Runner) SurfaceSignals(ctx context.Context, venue, surface string) ([]Sig, bool, error) {
	kind := ""
	switch {
	case strings.Contains(surface, "funding"):
		kind = "funding_view"
	case strings.Contains(surface, "quarter"), strings.Contains(surface, "basis"):
		kind = "basis_ann"
	case strings.Contains(surface, "book"), strings.Contains(surface, "depth"):
		kind = "depth5"
	default:
		return nil, false, nil // 非信号敏感面——零绑定如实
	}
	all, err := r.freshByKind(ctx, kind)
	if err != nil {
		return nil, false, err
	}
	if len(all) == 0 {
		return nil, false, nil
	}
	sigs := make([]Sig, 0, len(all))
	for _, s := range all {
		sigs = append(sigs, s)
	}
	if kind == "funding_view" { // 榜 top3 |val|（carry 候选燃料防挂载全宇宙）
		sort.Slice(sigs, func(i, j int) bool { return math.Abs(sigs[i].Val) > math.Abs(sigs[j].Val) })
		if len(sigs) > 3 {
			sigs = sigs[:3]
		}
	}
	return sigs, true, nil
}
