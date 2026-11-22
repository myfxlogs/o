// wakebus —— 工作面事件唤醒总线（工单O-32300——拍边界等待病灶根修：
// ops_event 关键行落库即 NOTIFY 唤醒对应 dispatch，15min timer 降级为
// 漏唤看门狗双保险；「滞留警报→等拍」一跳消除）。
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
// 派发器自带（本件不造第二把锁）。水位=内存态不回放史（停机期漏唤归
// timer 兜底）。桩测面：WAKEBUS_ONCE/ROWS_FILE/KICK_CMD/WM/BATCH_SEC/
// LAG_SEC（spec 四案=行集文件→踹件桩断言）。
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/events"
)

const (
	unitCognition = "o-cognition-dispatch.service"
	unitResearch  = "o-research-dispatch.service"
	unitWorker    = "o-worker-dispatch.service"
	unitLandq     = "o-land-queue.service"
)

var (
	// workEvents/ notePat = fin_0231 ev_work_trg WHEN 与 pending() 读行
	// 谓词同文（双写一致件——改须同改；file 桩测面 matchRow 同第三份）。
	workEvents = map[string]bool{
		"工单": true, "升件": true, "完工回执": true, "候审回执": true, "复审过": true,
	}
	notePat   = regexp.MustCompile(`完工回执|完工推支候审|候审回执|复审过\s+sha=`)
	reviewSHA = regexp.MustCompile(`复审过\s+sha=`)
)

type wakeRow struct {
	id    int64
	event string
	seat  string
	note  string
}

// routeRow 路由表——event/note 词面→唤起单元集（派发器幂等自闸，
// 总线只踹不判；未知行类返回 nil=不踹）。
func routeRow(event, note string) []string {
	switch event {
	case "工单":
		if strings.HasPrefix(note, "求知:") || strings.HasPrefix(note, "求知：") ||
			strings.HasPrefix(note, "研究席:") || strings.HasPrefix(note, "研究席：") {
			return []string{unitResearch}
		}
		return []string{unitWorker, unitCognition}
	case "完工回执", "候审回执":
		return []string{unitCognition}
	case "复审过":
		return []string{unitLandq}
	case "升件":
		return []string{unitCognition}
	default:
		switch {
		case strings.Contains(note, "完工回执") || strings.Contains(note, "完工推支候审") ||
			strings.Contains(note, "候审回执"):
			return []string{unitCognition}
		case reviewSHA.MatchString(note):
			return []string{unitLandq}
		}
		return nil
	}
}

// matchRow = 读行谓词镜像（file 桩测面预筛——与 WHEN/pending() 同文三处一致）。
func matchRow(event, note string) bool {
	return workEvents[event] || notePat.MatchString(note)
}

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

func lagSec() int {
	if v := os.Getenv("WAKEBUS_LAG_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 15
}

// pending 读行：谓词与 fin_0231 ev_work_trg WHEN 同文（双写一致件）。
func pending(ctx context.Context, pool *pgxpool.Pool, wm int64) ([]wakeRow, error) {
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, event, seat, note FROM ops_event
WHERE id > $1 AND ts < now() - interval '%d seconds'
  AND (event IN ('工单','升件','完工回执','候审回执','复审过')
       OR note ~ '完工回执|完工推支候审|候审回执|复审过[[:space:]]+sha=')
ORDER BY id`, lagSec()), wm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wakeRow
	for rows.Next() {
		var r wakeRow
		if err := rows.Scan(&r.id, &r.event, &r.seat, &r.note); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// fileRows 桩测面：行集 id|event|seat|note——matchRow 同文预筛+id>wm。
func fileRows(path string, wm int64) ([]wakeRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []wakeRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || id <= wm {
			continue
		}
		if !matchRow(parts[1], parts[3]) {
			continue
		}
		out = append(out, wakeRow{id: id, event: parts[1], seat: parts[2], note: parts[3]})
	}
	return out, sc.Err()
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
		log.Printf("[wakebus] 踹 %s 落败（timer 兜底仍在）: %v %.200s", unit, err, out)
	}
}

// processBatch：读行→路由→单元去重→逐踹→水位推进。
func processBatch(ctx context.Context, pool *pgxpool.Pool, wm *int64, kickFn func(context.Context, string)) error {
	var rows []wakeRow
	var err error
	if path := os.Getenv("WAKEBUS_ROWS_FILE"); path != "" {
		rows, err = fileRows(path, *wm)
	} else {
		rows, err = pending(ctx, pool, *wm)
	}
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	units := map[string]bool{}
	maxID := *wm
	for _, r := range rows {
		if r.id > maxID {
			maxID = r.id
		}
		for _, u := range routeRow(r.event, r.note) {
			units[u] = true
		}
	}
	for u := range units {
		kickFn(ctx, u)
	}
	*wm = maxID
	log.Printf("[wakebus] 踹 %d 单元（行 %d）", len(units), len(rows))
	return nil
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

	if os.Getenv("WAKEBUS_ONCE") == "1" {
		if err := processBatch(ctx, pool, &wm, kick); err != nil {
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
	log.Printf("[wakebus] ev_work 监听起——batch=%s lag=%ds wm=%d", batch, lagSec(), wm)
	events.ListenLoopChan(ctx, pool, "wakebus", []string{"ev_work"},
		func(ctx context.Context, _ string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(batch): // sched-exempt-ok 攒批窗——事件驱短窗合流防连珠踹（feed_push 聚合窗同例）
			}
			return processBatch(ctx, pool, &wm, kick)
		})
}
