// Package projector —— 投影臂（T-0168/D-0030 §四：桥面唯一正式接口的搬运工）。
// LISTEN PG-md 本地 ev_*（零轮询，D-0008 律）→活跃集谓词查最新行→写 SG 镜像表；
// fin_proposal 唯一双向面（SG state 列组回写）；反向桥（T-0172/D-0030 §五延伸）
// fin_strategy/fin_venue_untradeable SG 正本→PG-md 只读副本（ev_strategy 帧+
// 全表 diff）。谓词自算 fin_position/
// fin_strategy（不赖 fin_active_set 种子——0069 无回填，reconciler 同径兼治）；
// 启动全表对账自愈+活性兜底重对账（events.go 期界豁免类——失联自检非工作扫描）。
package projector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/events"
	"o/internal/platform/notify"
	"o/internal/platform/roster"
	"o/migrations"
)

const (
	seatName      = "投影官"
	seatDept      = "bridge"
	seatTitle     = "PG-md→SG 投影臂（八桥镜像+proposal 双向）"
	beatEvery     = 30 * time.Second
	livenessEvery = 10 * time.Minute // 活性兜底重对账（事件失联自检——钦定豁免类）
	mirrorStale   = 2 * time.Minute  // 镜像 max(ts) 陈阈——告警 projector_silent（D-0030 §七）
)

// Runner 投影臂本体。md=PG-md 正本池；sg=SG 镜像池（隧道径）。
type Runner struct {
	md, sg    *pgxpool.Pool
	alert     *notify.Alerter
	dualwrite bool // dualwrite=对拍期旁写泵（无谓词全量搬运；serve=活跃集谓词正形）

	mu     sync.Mutex
	active map[string]bool // 活跃集（SG fin_position 未平仓 ∪ fin_strategy 非 retired——谓词自算）
	dcd    []string        // dcd_call 活策略 product_id（=fin_strategy.inst_id）

	wmMu       sync.Mutex
	wm         map[string]time.Time     // per-table ts 水印（ticks/markidx/funding/signal/dcd/eval）
	propWm     int64                    // fin_proposal id 水印（md 正本 bigserial 单调）
	tsLastPass map[string]time.Time     // T-0193：per-channel 过拍最小间隔闸（串行 worker 挤兑根修）
	tsMinPass  map[string]time.Duration // 过拍间隔册（New 装配默认=tsMinPassInterval 注册表；测试可零化）
}

// New 装配（桥表集对册 fail-closed——消费面与 domain.tsv 单源禁漂移）。
// mode=serve（正形，缺省）|dualwrite（对拍旁写泵）。
func New(md, sg *pgxpool.Pool, alert *notify.Alerter, mode string) (*Runner, error) {
	if mode != "serve" && mode != "dualwrite" {
		return nil, fmt.Errorf("mode %q 非法（serve|dualwrite）", mode)
	}
	reg, err := migrations.BridgeTables()
	if err != nil {
		return nil, err
	}
	// 投影搬运面八桥表须 ⊆ 册 bridge 行（册可超——app_config 第九桥副本走
	// syncConfig 专径、fin_data_retention 分库独立账不经投影，见 T-0169）。
	want := map[string]bool{"fin_opp_eval": true, "fin_proposal": true, "md_dcd_product": true,
		"md_funding": true, "md_macro_event": true, "md_mark_index": true,
		"md_signal": true, "md_ticks": true}
	regSet := map[string]bool{}
	for _, t := range reg {
		regSet[t] = true
	}
	for t := range want {
		if !regSet[t] {
			return nil, fmt.Errorf("域册 bridge 缺投影面表 %s——册面先行", t)
		}
	}
	// 反向桥搬运面须 ⊆ 册 bridge-reverse 行（T-0172——SG 正本→PG-md 只读副本，
	// 同 fail-closed 对册）。
	regRev, err := migrations.ReverseBridgeTables()
	if err != nil {
		return nil, err
	}
	regRevSet := map[string]bool{}
	for _, t := range regRev {
		regRevSet[t] = true
	}
	for _, t := range []string{"fin_strategy", "fin_venue_untradeable"} {
		if !regRevSet[t] {
			return nil, fmt.Errorf("域册 bridge-reverse 缺反向桥表 %s——册面先行", t)
		}
	}
	return &Runner{md: md, sg: sg, alert: alert, dualwrite: mode == "dualwrite",
		wm: map[string]time.Time{}, active: map[string]bool{}}, nil
}

// Run 主环：守卫→启动对账→双侧听→beat/silent→活性兜底。败经 errc 岗终
// （systemd 重启→reconciler 自愈——验收锚「杀连自愈」的运行时形态）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	// 席位上岗（beatLoop 只 Beat 不 Register——册行缺席=岗终环，先 Register 幂等）。
	if err := roster.Register(ctx, r.sg, seatName, seatDept, seatTitle, nil); err != nil {
		errc <- fmt.Errorf("[projector] roster register: %w", err)
		return
	}
	if err := r.guards(ctx); err != nil {
		errc <- fmt.Errorf("[projector] boot 守卫: %w", err)
		return
	}
	if err := r.Reconcile(ctx, true); err != nil {
		errc <- fmt.Errorf("[projector] 启动对账: %w", err)
		return
	}
	log.Printf("[projector] 对账毕——活跃集 %d inst、dcd 活策略 %d 品", r.activeCount(), len(r.dcdSet()))

	mdCh := []string{"ev_ticks", "ev_markidx", "ev_funding", "ev_signal", "ev_eval", "ev_proposal", "ev_dcd"}
	sgCh := []string{"ev_active_set", "ev_proposal", "ev_strategy", "config_changed"}
	go events.ListenLoopChan(ctx, r.md, "projector-md", mdCh, r.mdWake)
	go events.ListenLoopChan(ctx, r.sg, "projector-sg", sgCh, r.sgWake)
	go r.beatLoop(ctx, errc)
	go r.livenessLoop(ctx, errc)
}

// mdWake/sgWake 帧分派面在 ops.go（执行域同居——projector.go 只留主环/装配）。

// refreshSets 谓词自算（SG 源表现读——不赖 fin_active_set 种子；D-0032 裁，
// 同径兼治 0069 无回填）。
func (r *Runner) refreshSets(ctx context.Context) error {
	rows, err := r.sg.Query(ctx, `SELECT inst_id FROM fin_position WHERE closed_at IS NULL
		UNION SELECT inst_id FROM fin_strategy WHERE state<>'retired'`)
	if err != nil {
		return fmt.Errorf("活跃集自算: %w", err)
	}
	active := map[string]bool{}
	for rows.Next() {
		var inst string
		if err := rows.Scan(&inst); err != nil {
			rows.Close()
			return err
		}
		active[inst] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	var dcd []string
	if err := r.sg.QueryRow(ctx, `SELECT COALESCE(array_agg(DISTINCT inst_id),'{}'::text[]) FROM fin_strategy
		WHERE state<>'retired' AND template_ref='dcd_call'`).Scan(&dcd); err != nil {
		return fmt.Errorf("dcd 活策略集: %w", err)
	}
	r.mu.Lock()
	r.active, r.dcd = active, dcd
	r.mu.Unlock()
	return nil
}

func (r *Runner) activeCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.active)
}

func (r *Runner) activeList() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.active))
	for k := range r.active {
		out = append(out, k)
	}
	return out
}

func (r *Runner) dcdSet() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.dcd...)
}

func (r *Runner) getWm(table string) time.Time {
	r.wmMu.Lock()
	defer r.wmMu.Unlock()
	return r.wm[table]
}

func (r *Runner) resetWm() {
	r.wmMu.Lock()
	defer r.wmMu.Unlock()
	r.wm = map[string]time.Time{}
	r.tsLastPass = map[string]time.Time{}
	r.tsMinPass = tsMinPassInterval
}

func (r *Runner) bumpWm(table string, ts time.Time) {
	r.wmMu.Lock()
	defer r.wmMu.Unlock()
	if ts.After(r.wm[table]) {
		r.wm[table] = ts
	}
}

func (r *Runner) getPropWm() int64 {
	r.wmMu.Lock()
	defer r.wmMu.Unlock()
	return r.propWm
}

func (r *Runner) bumpPropWm(id int64) {
	r.wmMu.Lock()
	defer r.wmMu.Unlock()
	if id > r.propWm {
		r.propWm = id
	}
}

// beatLoop 席位心跳+镜像静默哨（30s 拍——roster 台账+projector_silent 转沿告警）。
func (r *Runner) beatLoop(ctx context.Context, errc chan<- error) {
	t := time.NewTicker(beatEvery) // sched-exempt-ok 席位心跳+镜像静默哨（roster 台账活性类，非工作扫描）
	defer t.Stop()
	silent := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := roster.Beat(ctx, r.sg, seatName); err != nil {
				errc <- fmt.Errorf("[projector] beat: %w", err)
				return
			}
			stale, err := r.mirrorStaleSince(ctx)
			if err != nil {
				log.Printf("[projector] silent 哨读败: %v", err)
				continue
			}
			switch {
			case stale && !silent && r.alert != nil:
				r.alert.Alert(ctx, "projector_silent",
					"SG 镜像 max(ts) 陈>"+mirrorStale.String()+"——投影断流（projector 死或 PG-md 写停）")
			case !stale && silent:
				log.Printf("[projector] 镜像复鲜——projector_silent 解除")
			}
			silent = stale
		}
	}
}

// mirrorStaleSince SG ts 桥面最大值是否陈于阈（全空=未始不判陈）。
func (r *Runner) mirrorStaleSince(ctx context.Context) (bool, error) {
	var latest *time.Time
	err := r.sg.QueryRow(ctx, `SELECT max(ts) FROM (
		SELECT max(ts) ts FROM md_ticks UNION ALL SELECT max(ts) FROM md_mark_index
		UNION ALL SELECT max(ts) FROM md_signal UNION ALL SELECT max(ts) FROM md_dcd_product
		UNION ALL SELECT max(ts) FROM md_funding UNION ALL SELECT max(ts) FROM fin_opp_eval) q`).Scan(&latest)
	if err != nil {
		return false, err
	}
	return latest != nil && time.Since(*latest) > mirrorStale, nil
}

// livenessLoop 活性兜底（10min 期界——事件失联自检非工作扫描，events.go 钦定
// 豁免类）：断窗行与低频桥表靠期界重对账收敛。
func (r *Runner) livenessLoop(ctx context.Context, errc chan<- error) {
	t := time.NewTicker(livenessEvery) // sched-exempt-ok 活性兜底重对账（事件失联自检——events.go 钦定豁免类，非工作扫描）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := r.Reconcile(ctx, false); err != nil {
				errc <- fmt.Errorf("[projector] 活性对账: %w", err)
				return
			}
		}
	}
}
