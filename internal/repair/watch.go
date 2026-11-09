// Package repair —— 采集臂修复环（业主令「RSI=自我发现然后自我修复」：
// 无状态采集臂静默死自动重启）。白名单=collecto/projector/research
// 三件——o-core/钱径件/清道夫自身永不在册。判据=注册面鲜度超 dead_sec ∧
// 冷却/配额闸过 → systemctl --user restart <unit>；全动作落 md_repair_log
// （后端账非日志——SG 处置席可投影核证）。注册面/阈/白名单闸门参数全归
// app_config repair.* 族热更——env 零涉。
package repair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/config"
)

// unitWhitelist 可重启单元钦定白名单（业主令面——册外单永不触；
// 修本册=改码走闸,非配置面——防配置注入越权）。
var unitWhitelist = map[string]bool{
	"o-collecto":  true,
	"o-projector": true,
	"o-research":  true,
}

// feedReg 注册面行（app_config repair.feeds JSON 元素——table/where 拼
// max(ts) 鲜度查询,dead_sec=死阈秒）。
type feedReg struct {
	Name    string `json:"name"`
	Unit    string `json:"unit"`
	Table   string `json:"table"`
	Where   string `json:"where"`
	DeadSec int    `json:"dead_sec"`
}

// guardCfg 闸参数（repair.guard JSON——缺件走默认）。
type guardCfg struct {
	RestartGapSec int `json:"restart_gap_sec"` // 同单元重启最小间隔
	GraceSec      int `json:"grace_sec"`       // 重启后该单元面豁免窗
	MaxPerDay     int `json:"max_per_day"`     // 单单元日配额
}

var defGuard = guardCfg{RestartGapSec: 1800, GraceSec: 600, MaxPerDay: 3}

// unitState 单元态（重启时刻+当日计数——内存面,进程重启归零;
// 24h 配额以 md_repair_log 同窗补查兜底）。
type unitState struct {
	lastRestart time.Time
}

// Watch 常驻修复环：tick 节拍下逐注册面查鲜度→死→闸过→重启+落账。
// cfg=md 侧 config 件（repair.* 族——SG 正本投影径热更）。
func Watch(ctx context.Context, md *pgxpool.Pool, cfg *config.Client) {
	t := time.NewTicker(60 * time.Second) // sched-exempt-ok：修复判定期界非业务轮询
	defer t.Stop()
	st := map[string]*unitState{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		feeds, guard := loadCfg(cfg)
		for _, f := range feeds {
			check(ctx, md, f, guard, st)
		}
	}
}

func loadCfg(cfg *config.Client) ([]feedReg, guardCfg) {
	g := defGuard
	if raw, err := cfg.Get("repair.guard"); err == nil {
		_ = json.Unmarshal(raw, &g) // 坏件走默认不炸
		if g.RestartGapSec <= 0 {
			g.RestartGapSec = defGuard.RestartGapSec
		}
	}
	var fs []feedReg
	if raw, err := cfg.Get("repair.feeds"); err == nil {
		_ = json.Unmarshal(raw, &fs)
	}
	return fs, g
}

// check 单面判：鲜度死 ∧ 闸过 → restart+md_repair_log。
func check(ctx context.Context, md *pgxpool.Pool, f feedReg, g guardCfg, st map[string]*unitState) {
	if f.Table == "" || f.Unit == "" || f.DeadSec <= 0 {
		return
	}
	if !unitWhitelist[f.Unit] {
		log.Printf("[repair] 册外单元拒 %s——白名单三件外永不触", f.Unit)
		return
	}
	// 重启豁免窗：本单元刚被拉起,面陈属预热非死。
	if s := st[f.Unit]; s != nil && time.Since(s.lastRestart) < time.Duration(g.GraceSec)*time.Second {
		return
	}
	var ts *time.Time
	q := fmt.Sprintf(`SELECT max(ts) FROM %s WHERE %s`, f.Table, coalesceWhere(f.Where))
	if err := md.QueryRow(ctx, q).Scan(&ts); err != nil {
		return
	}
	if ts == nil || time.Since(*ts) <= time.Duration(f.DeadSec)*time.Second {
		return
	}
	// 闸①冷却 ②日配额（md_repair_log 同窗查——进程重启不丢忆）。
	if s := st[f.Unit]; s != nil && time.Since(s.lastRestart) < time.Duration(g.RestartGapSec)*time.Second {
		return
	}
	var today int
	_ = md.QueryRow(ctx, `SELECT count(*) FROM md_repair_log
		WHERE unit=$1 AND action='restart' AND ts>now()-interval '24 hours'`, f.Unit).Scan(&today)
	if today >= g.MaxPerDay {
		log.Printf("[repair] %s 日配额 %d 尽——面 %s 仍死,转告警不再重启（处置席接）",
			f.Unit, today, f.Name)
		audit(ctx, md, f.Unit, f.Name, "quota_burned", fmt.Sprintf("面死陈 %s 配额尽", time.Since(*ts).Round(time.Second)))
		return
	}
	active, _ := exec.CommandContext(ctx, "systemctl", "--user", "is-active", f.Unit).Output()
	if err := exec.CommandContext(ctx, "systemctl", "--user", "restart", f.Unit).Run(); err != nil {
		log.Printf("[repair] restart %s 败: %v", f.Unit, err)
		audit(ctx, md, f.Unit, f.Name, "restart_fail", fmt.Sprintf("is_active=%s err=%v", string(active), err))
		return
	}
	if st[f.Unit] == nil {
		st[f.Unit] = &unitState{}
	}
	st[f.Unit].lastRestart = time.Now()
	log.Printf("[repair] restart %s——面 %s 陈 %s（前置态 %s）",
		f.Unit, f.Name, time.Since(*ts).Round(time.Second), string(active))
	audit(ctx, md, f.Unit, f.Name, "restart",
		fmt.Sprintf("面陈 %s 超阈 %ds 前置态=%s", time.Since(*ts).Round(time.Second), f.DeadSec, string(active)))
}

func coalesceWhere(w string) string {
	if w == "" {
		return "true"
	}
	return w
}

// audit 全动作落账——md_repair_log（md_1093：后端账非日志,处置席/复盘核证面）。
func audit(ctx context.Context, md *pgxpool.Pool, unit, feed, action, note string) {
	if _, err := md.Exec(ctx, `INSERT INTO md_repair_log(unit,feed,action,note)
		VALUES($1,$2,$3,$4)`, unit, feed, action, note); err != nil {
		log.Printf("[repair] 落账败 %s/%s: %v", unit, action, err)
	}
}
