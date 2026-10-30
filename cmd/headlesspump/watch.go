// watch.go —— fs 事件分派+relay 尾扫+启动补扫+主环（泵的感知面；态机本体在
// pump.go）。补扫=一次性初始化非轮询（mbusd sweepStale 同形）。
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func (p *pump) onFs(ctx context.Context, ev fsnotify.Event) {
	base := filepath.Base(ev.Name)
	switch {
	case ev.Name == filepath.Join(p.dir, relayLog):
		p.tailReport(ev.Name)
	case strings.HasPrefix(ev.Name, hbDir+"/"):
		p.onHb(base)
	case strings.HasPrefix(base, prefixRun) && strings.HasSuffix(base, doneSuffix):
		tid := strings.TrimSuffix(strings.TrimPrefix(base, prefixRun), doneSuffix)
		if ev.Op&fsnotify.Create != 0 {
			p.onDone(p.ensure(tid, "", ""))
		} else if ev.Op&fsnotify.Remove != 0 {
			if t := p.tks[tid]; t != nil {
				t.doneF = false // 重派前清完工旗（新 spawn 锚即复活在飞）
			}
		}
	case strings.HasPrefix(base, prefixRun) && strings.HasSuffix(base, logSuffix):
		tid := strings.TrimSuffix(strings.TrimPrefix(base, prefixRun), logSuffix)
		p.onLog(p.ensure(tid, "Dev", ev.Name))
	case strings.HasPrefix(base, prefixAudit) && strings.HasSuffix(base, logSuffix):
		tid := strings.TrimSuffix(strings.TrimPrefix(base, prefixAudit), logSuffix)
		p.onLog(p.ensure(tid, "DM", ev.Name))
	}
}

var reportOff int64

// tailReport relay 自报行尾扫（boot 起只读尾——历史自报非现态活证）。
func (p *pump) tailReport(path string) {
	st, err := os.Stat(path)
	if err != nil || st.Size() < reportOff {
		if err == nil {
			reportOff = 0
		}
		return
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Seek(reportOff, 0)
	buf := make([]byte, st.Size()-reportOff)
	n, _ := f.Read(buf)
	reportOff += int64(n)
	for _, ln := range strings.Split(string(buf[:n]), "\n") {
		if m := reReport.FindStringSubmatch(ln); m != nil {
			p.onReport(m[1])
		}
	}
}

// sweep 启动补扫：在册票全量建档派初态（.done/proc 实证双锚），后续纯事件驱动。
func (p *pump) sweep(ctx context.Context) {
	for _, pat := range []string{prefixRun + "*" + logSuffix, prefixAudit + "*" + logSuffix} {
		fs, _ := filepath.Glob(filepath.Join(p.dir, pat))
		for _, f := range fs {
			base := filepath.Base(f)
			seat, pre := "Dev", prefixRun
			if strings.HasPrefix(base, prefixAudit) {
				seat, pre = "DM", prefixAudit
			}
			tid := strings.TrimSuffix(strings.TrimPrefix(base, pre), logSuffix)
			t := p.ensure(tid, seat, f)
			b, _ := os.ReadFile(f) // 全件扫一次定型（锚行时序——spawn/exit/done 末锚赢）
			t.offset = int64(len(b))
			if st, err := os.Stat(f); err == nil {
				t.lastAct = st.ModTime()
			}
			for _, ln := range strings.Split(string(b), "\n") {
				switch {
				case reSpawn.MatchString(ln):
					t.procAlive, t.stalled, t.overdue, t.doneF = true, false, false, false
				case reDone.MatchString(ln), reExitAudit.MatchString(ln):
					t.doneF = true
				case reNoDone.MatchString(ln), reYield.MatchString(ln):
					t.procAlive = false
				}
			}
			if _, err := os.Stat(strings.TrimSuffix(f, logSuffix) + doneSuffix); err == nil {
				t.doneF = true
			}
			if _, m, alive := findProc(tid); alive {
				t.procAlive, t.model = true, m
				p.arm(t) // 在飞票武装双判（期界顺延自 lastAct 见 arm 注）
			} else {
				t.procAlive = t.doneF // 无进程且未完工=死（完工票不翻死）
			}
			p.setState(t)
		}
	}
	if st, err := os.Stat(filepath.Join(p.dir, relayLog)); err == nil {
		reportOff = st.Size()
	}
}

func (p *pump) loop(ctx context.Context, w *fsnotify.Watcher) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			p.onFs(ctx, ev)
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("watcher: %v", err)
		case msg := <-p.in:
			p.onTimer(msg)
		}
	}
}
