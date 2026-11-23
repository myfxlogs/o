package main

// 批读/批处理核（O-34618 自 main.go 拆出——300 行棘轮）：读行谓词
// （pending/fileRows）+路由表+水位/seen 集+processBatch。滞后窗
// ts<now-lag 与回看窗 id>wm-LOOKBACK 双窗语义注记在各件头。

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
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

// lookbackID 补扫回看窗（O-34618：晚提交小 id 行兜底——id 序≈落库序，
// 滞后窗保「不读早」而 seen 集+回看窗保「不读漏」；5000 id≈日级覆盖，
// seen 集 8192 上限同步盖）。
func lookbackID() int64 {
	if v := os.Getenv("WAKEBUS_LOOKBACK_ID"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return 5000
}

func lagSec() int {
	if v := os.Getenv("WAKEBUS_LAG_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 15
}

// pending 读行：谓词与 fin_0231 ev_work_trg WHEN 同文（双写一致件）；
// id>wm-LOOKBACK 兼收补扫窗内晚提交小 id 行（O-34618——id≤wm 行由
// processBatch seen 集去重判新否，滞后窗 ts<now-lag 语义原样保留）。
func pending(ctx context.Context, pool *pgxpool.Pool, wm int64) ([]wakeRow, error) {
	rows, err := pool.Query(ctx, fmt.Sprintf(`
SELECT id, event, seat, note FROM ops_event
WHERE id > $1 AND ts < now() - interval '%d seconds'
  AND (event IN ('工单','升件','完工回执','候审回执','复审过')
       OR note ~ '完工回执|完工推支候审|候审回执|复审过[[:space:]]+sha=')
ORDER BY id`, lagSec()), wm-lookbackID())
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

// batchState 水位+seen 补扫集（O-34618：seen=已处理行 id 集——补扫窗
// 内 id≤wm 行重见时判新否，防 10s 补扫对同一批历史关键行永踹）。
type batchState struct {
	wm    int64
	seen  map[int64]struct{}
	seenQ []int64
}

const seenCap = 8192 // seen FIFO 上限——盖过 lookbackID 窗内关键行量级

func (st *batchState) markSeen(id int64) {
	if _, ok := st.seen[id]; ok {
		return
	}
	st.seen[id] = struct{}{}
	st.seenQ = append(st.seenQ, id)
	for len(st.seenQ) > seenCap {
		delete(st.seen, st.seenQ[0])
		st.seenQ = st.seenQ[1:]
	}
}

// processBatch：读行→seen 去重（id≤wm 补扫行）→路由→单元去重→逐踹→
// 水位只推进到已处理行（id>wmOld 的处理行 max；补扫行不动水位）。
func processBatch(ctx context.Context, pool *pgxpool.Pool, st *batchState, kickFn func(context.Context, string)) error {
	var rows []wakeRow
	var err error
	if path := os.Getenv("WAKEBUS_ROWS_FILE"); path != "" {
		rows, err = fileRows(path, st.wm)
	} else {
		rows, err = pending(ctx, pool, st.wm)
	}
	if err != nil {
		return err
	}
	units := map[string]bool{}
	maxNew := st.wm
	fresh := 0
	for _, r := range rows {
		if r.id <= st.wm { // 补扫窗内小 id 行：seen 在册=已处理跳过
			if _, ok := st.seen[r.id]; ok {
				continue
			}
		}
		st.markSeen(r.id)
		fresh++
		if r.id > maxNew {
			maxNew = r.id
		}
		for _, u := range routeRow(r.event, r.note) {
			units[u] = true
		}
	}
	if fresh == 0 {
		return nil
	}
	for u := range units {
		kickFn(ctx, u)
	}
	st.wm = maxNew
	log.Printf("[wakebus] 踹 %d 单元（新行 %d/读 %d）", len(units), fresh, len(rows))
	return nil
}
