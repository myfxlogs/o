// replica.go —— 第九桥副本面（T-0169）：app_config 正本 SG 全量 diff 副本+
// md 侧 config_changed 回灌（热启链路保全）；宏观日历 diff 搬运。
package projector

import (
	"context"
	"fmt"
	"time"
)

// syncConfig app_config 第九桥副本同步（正本 SG——全量 diff 副本+md 侧
// config_changed 回灌，热启链路保全；app_secrets 不副本不出 SG）。
func (r *Runner) syncConfig(ctx context.Context) error {
	sgRows, err := r.sg.Query(ctx, qConfigAll)
	if err != nil {
		return fmt.Errorf("config 正本读: %w", err)
	}
	type kv struct {
		key        string
		value      string
		generation int64
		updated    time.Time
		updatedBy  string // T-0178 审计列随正本副本同步
	}
	want := map[string]kv{}
	defer sgRows.Close()
	for sgRows.Next() {
		var c kv
		if err := sgRows.Scan(&c.key, &c.value, &c.generation, &c.updated, &c.updatedBy); err != nil {
			return err
		}
		want[c.key] = c
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	have := map[string]kv{}
	mdRows, err := r.md.Query(ctx, qConfigAll)
	if err != nil {
		return fmt.Errorf("config 副本读: %w", err)
	}
	defer mdRows.Close()
	for mdRows.Next() {
		var c kv
		if err := mdRows.Scan(&c.key, &c.value, &c.generation, &c.updated, &c.updatedBy); err != nil {
			return err
		}
		have[c.key] = c
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	var gone []string
	for k, c := range want {
		if h, ok := have[k]; !ok || h != c {
			batch = append(batch, []any{c.key, c.value, c.generation, c.updated, c.updatedBy})
		}
	}
	for k := range have {
		if _, ok := want[k]; !ok {
			gone = append(gone, k)
		}
	}
	if len(gone) > 0 {
		if _, err := r.md.Exec(ctx, dConfigGone, gone); err != nil {
			return fmt.Errorf("config 副本删陈: %w", err)
		}
	}
	if err := batchExec(ctx, r.md, uConfig, batch); err != nil {
		return fmt.Errorf("config 副本投影: %w", err)
	}
	if len(batch) > 0 || len(gone) > 0 { // 副本变了才回灌热启帧（payload=表名惯例）
		if _, err := r.md.Exec(ctx, "SELECT pg_notify('config_changed','app_config')"); err != nil {
			return fmt.Errorf("config_changed 回灌: %w", err)
		}
	}
	return nil
}

// projectMacro 宏观日历全表 diff（~30 行 event_date 键——无 ts 全列覆写语义）。
func (r *Runner) projectMacro(ctx context.Context) error {
	mdRows, err := r.md.Query(ctx, qMacroAll)
	if err != nil {
		return fmt.Errorf("macro md: %w", err)
	}
	type ev struct {
		date   time.Time
		kind   string
		brief  string
		source string
	}
	want := map[time.Time]ev{}
	defer mdRows.Close()
	for mdRows.Next() {
		var e ev
		if err := mdRows.Scan(&e.date, &e.kind, &e.brief, &e.source); err != nil {
			return err
		}
		want[e.date] = e
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	have := map[time.Time]ev{}
	sgRows, err := r.sg.Query(ctx, `SELECT event_date,kind,title,source FROM md_macro_event`)
	if err != nil {
		return fmt.Errorf("macro sg: %w", err)
	}
	defer sgRows.Close()
	for sgRows.Next() {
		var e ev
		if err := sgRows.Scan(&e.date, &e.kind, &e.brief, &e.source); err != nil {
			return err
		}
		have[e.date] = e
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for d, e := range want {
		if h, ok := have[d]; !ok || h != e {
			batch = append(batch, []any{e.date, e.kind, e.brief, e.source})
		}
	}
	return batchExec(ctx, r.sg, uMacro, batch)
}
