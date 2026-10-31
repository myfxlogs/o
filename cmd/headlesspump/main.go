// cmd/headlesspump —— 无头单状态面事件泵（T-0263 业主令：信号径全推送制；
// 秒级化钉案 2026-10-31：5s 拍 tick——dir 对账/在飞 /proc 实证/30s 心跳三门控，
// kill 突殁 ≤10s 上板不等 30min 静默判；写败退避 30s 自愈）。
// mbusd 同款骨架：inotify 感知（~/devin-headless-*.log/devin-audit-*.log/.done
// + mail .hb 心跳面 + devin-relay 自报行）→ 票级状态机 → ops_headless upsert
// （行版 seq+ev_headless NOTIFY 由 DB 触发器派发）→o-core Watch→前端瓦片。
// 死亡/挂起=泵侧静默超时判定（单发期界定时器事件驱动）+5s 拍 /proc 实证兜底；
// 进程生死=/proc cmdline 实证。daemon 互斥单飞（supervisor 同形 flock）。
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sys/unix"
)

const (
	lockPath       = "/tmp/o-headlesspump.lock"
	hbDir          = "/opt/meta/mail/.hb"
	silenceFor     = 30 * time.Minute // 日志静默阈（supervisor STALL 同口径）
	overdueFor     = 8 * time.Hour    // 在飞逾期阈（supervisor MAXRUN 同口径）
	emitMinGap     = time.Second      // 活动帧节流（一跳 ≤2s 锚内——满窗尾帧补新态）
	beatEvery      = 5 * time.Second  // 秒级化钉案①：泵周期拍（对账/proc 实证/心跳三门控）
	heartbeatEvery = 30 * time.Second // 秒级化钉案②：liveness 心跳拍（在飞行 updated_at 保鲜）
	backoffFor     = 30 * time.Second // 秒级化钉案④：写败退避拍（emit 成功自愈回 5s）
	relayLog       = "devin-relay.log"
	doneSuffix     = ".done"
	logSuffix      = ".log"
	prefixRun      = "devin-headless-"
	prefixAudit    = "devin-audit-"
)

func main() {
	log.SetFlags(log.LstdFlags)
	l, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		log.Fatalf("lock 开败: %v", err)
	}
	if err := unix.Flock(int(l.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		log.Fatalf("同泵在飞（flock %s 被持）——副本让位", lockPath)
	}

	dsn := os.Getenv("ROSTER_PG_DSN") // 白名单件（fin 册隧道径——report.sh 同源）
	if dsn == "" {
		log.Fatal("ROSTER_PG_DSN 缺——ops_headless 写径无目的地 fail-closed")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("roster 池连接败: %v", err)
	}
	defer pool.Close()

	home, _ := os.UserHomeDir()
	dir := flag.String("dir", home, "无头日志目录（wrapper HEADLESS_LOGDIR 缺省同=$HOME——env 白名单禁增故走旗参）")
	flag.Parse()
	p := newPump(pool, *dir) // pool 满足 writer 缝（Exec 同签名）
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("watcher 建败: %v", err)
	}
	defer watcher.Close()
	for _, d := range []string{*dir, hbDir} {
		if err := watcher.Add(d); err != nil {
			log.Printf("WARN watch %s 缺席（该面失感）: %v", d, err)
		}
	}
	p.sweep(ctx) // 启动补扫（一次性初始化——mbusd sweepStale 同形，非轮询）
	log.Printf("o-headlesspump up watch=%s hb=%s", *dir, hbDir)
	p.loop(ctx, watcher)
}
