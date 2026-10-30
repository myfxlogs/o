// pump.go —— 票级状态机+事件环（单 goroutine 串行——定时器回调只往环内投信
// 不直改态，无锁）。信号源三：①日志写/.done（inotify——干活必写=活证据）；
// ②.hb 心跳面触碰（席级活证——只刷 lastAct 显面，停滞判定仍钉票级日志静默，
// 防席它务掩盖本票真停）；③relay 自报行（[自报] 携票号=票级活证）。
// 五态派生=标志位函数（终态不猜）：done>dead>stalled>overdue>alive。
package main

import (
	"context"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// wrapper 行锚（票面机读协议同 supervisor/relay——时间戳行首+命名词）。
var (
	reSpawn     = regexp.MustCompile(`无头(?:审计)? (T-\d+) spawn`)
	reExitRun   = regexp.MustCompile(`无头 (T-\d+) 进程退出 rc=`)
	reExitAudit = regexp.MustCompile(`无头审计 (T-\d+) 退出 rc=`)
	reDone      = regexp.MustCompile(`(T-\d+) 收口待审命中`)
	reNoDone    = regexp.MustCompile(`(T-\d+) 退出但全树无收口待审标记`)
	reYield     = regexp.MustCompile(`无头 (T-\d+) 让位`)
	reReport    = regexp.MustCompile(`\[自报\]\[[^\]]+\] \S+ (T-\d+)`)
)

// tk 票级态（标志位——derive() 派生五态串，事件只改旗不直写态）。
type tk struct {
	tid, seat                                     string
	logPath                                       string
	offset                                        int64 // tail 解析偏移（文件截短=复位）
	model                                         string
	state                                         string // 派生快照（变才发）
	lastAct                                       time.Time
	lastEmit                                      time.Time
	procAlive, stalled, overdue, doneF, emitDirty bool
	emitT, silenceT, overdueT                     *time.Timer
}

func (t *tk) derive() string {
	switch {
	case t.doneF:
		return "done"
	case !t.procAlive:
		return "dead"
	case t.stalled:
		return "stalled"
	case t.overdue:
		return "overdue"
	default:
		return "alive"
	}
}

type pump struct {
	w   writer
	dir string
	tks map[string]*tk
	in  chan string // 定时器投信："silence|overdue|emit <TID>"
}

// writer 写缝（pgxpool 实装——测试注假件验状态机不连库）。
type writer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func newPump(w writer, dir string) *pump {
	return &pump{w: w, dir: dir, tks: map[string]*tk{}, in: make(chan string, 64)}
}

func (p *pump) ensure(tid, seat, logPath string) *tk {
	t := p.tks[tid]
	if t == nil {
		t = &tk{tid: tid, seat: seat, logPath: logPath, procAlive: true}
		p.tks[tid] = t
	}
	if seat != "" {
		t.seat = seat
	}
	if logPath != "" {
		t.logPath = logPath
	}
	return t
}

// arm 两单发期界闸重臂自 lastAct（写/自报即续命——期界锚=末次活动非判时点，
// 补扫在飞票的存量静默不白赦；sched-exempt 单发期界非周期）。
func (p *pump) arm(t *tk) {
	if t.silenceT != nil {
		t.silenceT.Stop()
	}
	if t.overdueT != nil {
		t.overdueT.Stop()
	}
	silenceDue := silenceFor - time.Since(t.lastAct)
	overdueDue := overdueFor - time.Since(t.lastAct)
	t.silenceT = time.AfterFunc(max(silenceDue, 0), func() { p.in <- "silence " + t.tid }) // sched-exempt-ok 单发静默判
	t.overdueT = time.AfterFunc(max(overdueDue, 0), func() { p.in <- "overdue " + t.tid }) // sched-exempt-ok 单发逾期判
}

func (p *pump) disarm(t *tk) {
	for _, tm := range []*time.Timer{t.silenceT, t.overdueT} {
		if tm != nil {
			tm.Stop()
		}
	}
}

// setState 派生变才发（幂等——同态重复事件零帧）。
func (p *pump) setState(t *tk) {
	if s := t.derive(); s != t.state {
		t.state = s
		p.emit(t)
	}
}

// emitSoon 活动帧节流（满窗尾帧补新态——突发写不砸库）。
func (p *pump) emitSoon(t *tk) {
	if t.emitDirty {
		return
	}
	if d := emitMinGap - time.Since(t.lastEmit); d > 0 {
		t.emitDirty = true
		t.emitT = time.AfterFunc(d, func() { p.in <- "emit " + t.tid }) // sched-exempt-ok 单发节流尾帧
		return
	}
	p.emit(t)
}

func (p *pump) emit(t *tk) {
	t.emitDirty = false
	t.lastEmit = time.Now()
	var la *time.Time
	if !t.lastAct.IsZero() {
		la = &t.lastAct
	}
	_, err := p.w.Exec(context.Background(),
		`INSERT INTO ops_headless(ticket,seat,model,state,last_activity)
		 VALUES($1,$2,$3,$4,$5)
		 ON CONFLICT(ticket) DO UPDATE SET
		   seat=CASE WHEN EXCLUDED.seat<>'' THEN EXCLUDED.seat ELSE ops_headless.seat END,
		   model=CASE WHEN EXCLUDED.model<>'' THEN EXCLUDED.model ELSE ops_headless.model END,
		   state=EXCLUDED.state,
		   last_activity=GREATEST(EXCLUDED.last_activity,ops_headless.last_activity),
		   updated_at=now()`,
		t.tid, t.seat, t.model, t.state, la)
	if err != nil {
		log.Printf("WARN %s upsert 败（下事件自愈——有界态全行重写）: %v", t.tid, err)
		return
	}
	log.Printf("emit %s seat=%s state=%s model=%s", t.tid, t.seat, t.state, t.model)
}

// onLog 日志写帧：tail 增量扫行锚 + 活动记账。行锚序=wrapper 时序（exit 行先、
// 命中/无收口行随至——exit 自身不判态，后继锚裁决；wrapper 殁于 exit~裁决间
// 的残案由静默判 30min 兜底）。
func (p *pump) onLog(t *tk) {
	st, err := os.Stat(t.logPath)
	if err != nil {
		return
	}
	if st.Size() < t.offset {
		t.offset = 0 // 截短复位
	}
	f, err := os.Open(t.logPath)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(t.offset, 0); err != nil {
		return
	}
	buf := make([]byte, st.Size()-t.offset)
	n, _ := f.Read(buf)
	t.offset += int64(n)
	t.lastAct = st.ModTime()
	t.stalled = false // 活动即醒（停滞旗单代幂等——新写帧清旗可再报）
	p.arm(t)
	for _, ln := range strings.Split(string(buf[:n]), "\n") {
		switch {
		case reSpawn.MatchString(ln):
			_, m, _ := findProc(t.tid)
			t.model = m
			t.procAlive, t.stalled, t.overdue, t.doneF = true, false, false, false
		case reDone.MatchString(ln):
			t.doneF = true
			p.disarm(t)
		case reNoDone.MatchString(ln):
			t.procAlive = false
			p.disarm(t)
		case reExitAudit.MatchString(ln):
			t.doneF = true // 审计退出=本代审计 pass 完工（判决相位走 ops_event）
			p.disarm(t)
		case reYield.MatchString(ln):
			_, _, alive := findProc(t.tid)
			t.procAlive = alive // 让位=它代在飞或全殁——proc 实证
		case reExitRun.MatchString(ln):
			// 判态待后继锚（收口命中/无收口行毫秒内随至——见文件头注）
		}
	}
	p.setState(t)
	p.emitSoon(t)
}

// onDone/.done 落件=完工铁证（wrapper 侦听锚同件）。
func (p *pump) onDone(t *tk) {
	t.doneF = true
	p.disarm(t)
	p.setState(t)
}

// onHb .hb 触碰=席级活证——只刷显面 lastAct（不动停滞判锚，席它务不掩本票停）。
func (p *pump) onHb(seat string) {
	for _, t := range p.tks {
		if t.seat == seat && t.procAlive && !t.doneF {
			t.lastAct = time.Now()
			p.emitSoon(t)
		}
	}
}

// onReport relay 自报行携票号=票级活证（report.sh 落行——自报即心跳同源）。
func (p *pump) onReport(tid string) {
	if t := p.tks[tid]; t != nil && t.procAlive && !t.doneF {
		t.lastAct = time.Now()
		t.stalled = false
		p.arm(t)
		p.emitSoon(t)
	}
}

func (p *pump) onTimer(msg string) {
	what, tid, _ := strings.Cut(msg, " ")
	t := p.tks[tid]
	if t == nil {
		return
	}
	switch what {
	case "emit":
		if t.emitDirty {
			p.emit(t)
		}
	case "silence":
		if t.doneF {
			return
		}
		_, m, alive := findProc(t.tid) // 静默到点=/proc 实证：在=停滞 殁=死
		if alive && m != "" {
			t.model = m
		}
		t.procAlive = alive
		if alive {
			t.stalled = true
			p.arm(t) // 持续静默=链式再判（死转迟捕——殁于停滞中的进程无人写帧）
		}
		p.setState(t)
	case "overdue":
		if !t.doneF && t.procAlive {
			t.overdue = true
			p.setState(t)
		}
	}
}
