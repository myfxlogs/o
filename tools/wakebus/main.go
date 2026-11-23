// wakebus —— 工作面事件唤醒总线（工单O-32300——拍边界等待病灶根修：
// ops_event 关键行落库即 NOTIFY 唤醒对应 dispatch；O-34618 漏唤兜底
// 唯一件=本件周期补扫水位后未处理行（裁案 a——o-worker.timer/
// o-cognition.timer 保持 disabled 与 ops.unit_registry 期望态/宿主
// 实态三处一致，timer 非兜底零第三径）；「滞留警报→等拍」一跳消除）。
//
// 形态取舍（spec 钦定二选一，选 Go）：唤起语义=提示非数据送达——
// bash+psql 管道 stdin 非 tty 时空闲不消费异步通知（实证：NOTIFY 积
// socket 缓冲不打印，须心跳查询硬喂=脆弱 hack）；events.ListenLoopChan
// =钦定基建（专读排干+cap1 合流+断线重连退避）现成。通道取舍：新
// ev_work（fin_0231 行级 WHEN 只发关键行）不复用 ev_feed——ev_feed=
// 动作流水全量帧（fin_0185 剔六类外皆发），路由谓词不同源，复用=每帧
// 白醒+谓词双写漂移面；ev_work WHEN 与本件读行谓词同文（双写一致件）。
// 路由表（event/note 词面→踹对应 dispatch 单元一次；管内类精确词表
// 归派发器自闸唯一真源——总线复制=双写漂移面，多踹一发幂等无害）：
//
//	工单 ^求知[:：]|^研究席[:：] → o-research-dispatch.service
//	工单 其余                  → o-worker-dispatch.service + o-cognition-dispatch.service
//	完工回执|候审回执|异event行note携回执词面 → o-cognition-dispatch.service
//	复审过|note「复审过 sha=」  → o-land-queue.service
//	升件                      → o-cognition-dispatch.service（非crit判据阈归
//	                            dispatch.sh 内部——踹侧对齐扩面）
//
// 防抖=6s 攒批窗（feed_push 聚合窗同例）+NOTIFY 同事务合流；单飞闸=
// 派发器自带（本件不造第二把锁）。水位=内存态+seen 补扫集（O-34618：
// pending 读 id>wm-LOOKBACK 兼收晚提交小 id 行——滞后窗只防读早不防
// 读漏，seen 集去重防补扫重踹；重启 seen 空=补扫窗内历史关键行
// 重踹一波幂等无害）。桩测面：WAKEBUS_ONCE/ROWS_FILE/KICK_CMD/WM/
// BATCH_SEC/LAG_SEC/SWEEP_SEC/LOOKBACK_ID（spec 案=行集文件→踹件桩
// 断言；DB 读行径另有 main_test.go Go 桩）。
package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/events"
)

// envDSN 取 FIN_PG_DSN（env 优先；回退 REPORT_ENV_FILE/默认 env 件——同 eventbus 形）。
func envDSN(key, file string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if file == "" {
		file = os.Getenv("REPORT_ENV_FILE")
	}
	if file == "" {
		file = "/opt/o-research/env"
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, key+"=") {
			return strings.Trim(strings.TrimPrefix(ln, key+"="), `"`)
		}
	}
	return ""
}

// kick 踹件：WAKEBUS_KICK_CMD 桩测径收单元名；正径 systemctl --user
// start --no-block（踹=跑一次拍幂等自闸兜底；踹败=timer 兜底接管，fail-open）。
func kick(ctx context.Context, unit string) {
	if cmd := os.Getenv("WAKEBUS_KICK_CMD"); cmd != "" {
		if err := exec.CommandContext(ctx, cmd, unit).Run(); err != nil {
			log.Printf("[wakebus] 桩踹 %s 落败: %v", unit, err)
		}
		return
	}
	if out, err := exec.CommandContext(ctx, "systemctl", "--user", "start", "--no-block", unit).CombinedOutput(); err != nil {
		log.Printf("[wakebus] 踹 %s 落败（行已处理不补踹——补扫只兜漏读）: %v %.200s", unit, err, out)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fileMode := os.Getenv("WAKEBUS_ROWS_FILE") != ""
	dsn := envDSN("FIN_PG_DSN", "")
	var pool *pgxpool.Pool
	if !fileMode {
		if dsn == "" {
			log.Printf("[wakebus] FIN_PG_DSN 读不出——fail-closed不猜")
			os.Exit(1)
		}
		var err error
		pool, err = pgxpool.New(ctx, dsn)
		if err != nil {
			log.Printf("[wakebus] pgxpool 起败: %v", err)
			os.Exit(1)
		}
		defer pool.Close()
	}

	// 水位：内存态不回放史（WAKEBUS_WM 覆写；file 面缺省 0=全量行集）。
	wm := int64(-1)
	if v := os.Getenv("WAKEBUS_WM"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			wm = n
		}
	}
	if wm < 0 {
		if fileMode {
			wm = 0
		} else if err := pool.QueryRow(ctx, "SELECT COALESCE(max(id),0) FROM ops_event").Scan(&wm); err != nil {
			log.Printf("[wakebus] 水位初读败: %v", err)
			os.Exit(1)
		}
	}

	st := &batchState{wm: wm, seen: map[int64]struct{}{}}
	if os.Getenv("WAKEBUS_ONCE") == "1" {
		if err := processBatch(ctx, pool, st, kick); err != nil {
			log.Printf("[wakebus] 单批次败: %v", err)
			os.Exit(1)
		}
		return
	}

	batch := 6 * time.Second // 攒批窗——feed_push 聚合窗同例（WAKEBUS_BATCH_SEC 桩测覆写）
	if v := os.Getenv("WAKEBUS_BATCH_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			batch = time.Duration(n) * time.Second
		}
	}
	sweep := 10 * time.Second // O-34618 补扫兜底——最坏踹时=lag15+sweep10≈T+20s < lag+batch+5=26s 钦定界
	if v := os.Getenv("WAKEBUS_SWEEP_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			sweep = time.Duration(n) * time.Second
		}
	}
	var mu sync.Mutex // NOTIFY 径与补扫径共享 processBatch（seen/wm 非并发安全）
	runBatch := func() error {
		mu.Lock()
		defer mu.Unlock()
		return processBatch(ctx, pool, st, kick)
	}
	go sweepLoop(ctx, sweep, runBatch)
	log.Printf("[wakebus] ev_work 监听起——batch=%s sweep=%s lag=%ds lookback=%d wm=%d",
		batch, sweep, lagSec(), lookbackID(), st.wm)
	events.ListenLoopChan(ctx, pool, "wakebus", []string{"ev_work"},
		func(ctx context.Context, _ string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(batch): // sched-exempt-ok 攒批窗——事件驱短窗合流防连珠踹（feed_push 聚合窗同例）
			}
			return runBatch()
		})
}

// sweepLoop 周期补扫（O-34618 裁案 a——漏唤兜底唯一件：NOTIFY 单发静默期
// 水位后未处理行不再无限搁浅；timer 保持 disabled 三处一致）。fn=互斥后
// 的批次径，败=记日志下拍续（fail-open 同踹件语义）。
func sweepLoop(ctx context.Context, every time.Duration, fn func() error) {
	t := time.NewTicker(every) // sched-exempt-ok 周期补扫=漏唤兜底本体（spec 钦定 a 案——事件间隔不定无事件可驱）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := fn(); err != nil {
				log.Printf("[wakebus] 补扫批败: %v", err)
			}
		}
	}
}
