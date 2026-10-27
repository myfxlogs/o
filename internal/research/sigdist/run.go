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
var datedFutRe = regexp.MustCompile(`^([A-Z0-9]+-[A-Z]+)-([0-9]{6})$`)

// Runner 信号蒸馏岗（桥面常驻节拍——随采集器进程）。
type Runner struct {
	pool *pgxpool.Pool
	cfg  *config.Client // T-0141 跨所族映射读件（AttachConfig 可选）
	rej  *rejects.Door
}

// New 构造。
func New(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool,
		rej: rejects.New("sigdist", "basis_read_failed", "funding_read_failed", "sig_write_err",
			"xvenue_map_missing")}
}

// Run 在岗（蒸馏节拍——写协程语义同桥面：失败计 rejects 下一拍再算，fail-closed 不造零）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "信号蒸馏", "data",
		"桥面三族信号蒸馏岗（basis_ann/funding_view/depth5→md_signal，D-0022）", nil); err != nil {
		errc <- fmt.Errorf("信号蒸馏 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "信号蒸馏", 30*time.Second, errc) // T-0184：BeatLoop 补线——Register 有 BeatLoop 缺=活性台账假死病灶根治
	t := time.NewTicker(distillPeriod)                            // sched-exempt-ok 桥面蒸馏拍（慢特征计算节拍非轮询拉取）
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

// fundingView per SWAP 当期 funding rate（md_funding 最新行）+meta 附下一期
// 预测（next_funding_rate venue 域内含）+7d 滚动均值（窗口缺数据=字段缺席如实）。
func (r *Runner) fundingView(ctx context.Context) error {
	// T-0196 血统闸：ts=源行 recv_ts（观测戳非写拍——md_funding.ts=结算
	// 拍可未来值；leastTs 同构即最新参与行观测龄）。
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (inst_id)
			inst_id, ts, funding_rate, next_funding_rate, next_funding_ts, recv_ts
		FROM md_funding ORDER BY inst_id, ts DESC`)
	if err != nil {
		return err
	}
	type fv struct {
		inst  string
		ts    time.Time
		recv  time.Time
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
		if err := rows.Scan(&f.inst, &f.ts, &f.rate, &nextR, &nextT, &f.recv); err != nil {
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
	for inst, f := range vals {
		wts := f.recv // 血统戳=源行 recv_ts（T-0196——非写拍 now）
		if wts.IsZero() {
			wts = f.ts // recv_ts 缺=ts 兜底（源行观测戳唯一候选）
		}
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
			inst, f.rate, string(metaB), wts); err != nil {
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
	// T-0199 三巡：种类自适闸——funding_view=当值期判据（阶梯序列鲜=值
	// 仍当值），其余=120s 帧龄（每帧数据正确闸守）。
	err := pool.QueryRow(ctx, `SELECT val, meta, ts FROM md_signal
		WHERE kind=$1 AND key=$2 AND (
			(kind='funding_view' AND (meta->>'next_funding_ts')::timestamptz > now())
			OR (kind<>'funding_view' AND ts > now()-interval '120 seconds'))`,
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
	// T-0199 三巡：种类自适闸（funding_view=当值期，余=120s 帧龄——
	// 评估燃料「榜 top3」读闸全族恒 0 行断供病灶根治）。
	rows, err := r.pool.Query(ctx, `SELECT key, val, meta FROM md_signal
		WHERE kind=$1 AND (
			(kind='funding_view' AND (meta->>'next_funding_ts')::timestamptz > now())
			OR (kind<>'funding_view' AND ts > now()-interval '120 seconds'))`, kind)
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
