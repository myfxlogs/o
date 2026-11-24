// organ.go —— 外部器官产出账（O-38640 案③）：cron/systemd 器官
// 「最后产出时刻」巡检——arch_daily.sh 类宿主脚本死 26 天无感的
// 对治面。名单=app_config `ops.cron_manifest`（JSON 串数组
// ["名:定位形",…]，投影面 md app_config 随 SG 正本热更）；
// 阈=`ops.organ_max_age_h`（缺省 36h——日器官漏一日+半日余量）。
// 名单空/缺键=不探（防误报——未登记器官永不入册）。
//
// 定位形二式：
//   - 含 / 或 *：文件径/glob——匹配件最大 mtime=末次产出
//     （零匹配=nil→零行史升件=面未产）；
//   - "systemd"：systemctl --user show <名>.timer LastTriggerUSec
//     =末次触发（从未触发=nil→升件；命令败=读败上抛）。
//
// 只报不修——器官本体修归运维径/常票。
package repair

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"o/internal/platform/selfprobe"
)

const (
	// organManifestKey 器官产出账名单键（app_config——SG 正本投影 md 面）。
	organManifestKey = "ops.cron_manifest"
	// organMaxAgeKey 器官陈龄阈键（小时——缺省 organMaxAgeDef）。
	organMaxAgeKey = "ops.organ_max_age_h"
	// organMaxAgeDef 缺省阈（日器官 36h——漏一日+半日余量）。
	organMaxAgeDef = 36.0
)

// organEntry 名单行解析件（"名:定位形"——名限 [A-Za-z0-9_-]）。
func parseOrganEntry(raw string) (name, locator string, err error) {
	i := strings.Index(raw, ":")
	if i <= 0 {
		return "", "", fmt.Errorf("名单行缺冒号界: %q", raw)
	}
	name, locator = strings.TrimSpace(raw[:i]), strings.TrimSpace(raw[i+1:])
	if name == "" || locator == "" {
		return "", "", fmt.Errorf("名单行空段: %q", raw)
	}
	return name, locator, nil
}

// organSpec 单器官升件规格（scope=ORGAN-<名大写>——面独立不撞窗）。
func organSpec(name string, maxAge time.Duration, thresh string) selfprobe.Spec {
	return selfprobe.Spec{
		Site:     "sweeper",
		Ticket:   "O-38640",
		Scope:    "ORGAN-" + strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(name)),
		Label:    "器官 " + name,
		Verb:     "停产",
		TsCol:    "末次产出",
		MaxAge:   maxAge,
		Thresh:   thresh,
		ZeroWord: "零产出史（清单定位面无匹配/从未触发）",
		Ditto:    "器官本体死/调度失——对拍归宿主 cron/systemctl 面",
	}
}

// organFace 名+定位形→末次产出读缝（systemd 形名段=单元基名）。
func organFace(ctx context.Context, name, locator string, sysLast func(context.Context, string) (*time.Time, error)) selfprobe.Face {
	if strings.ContainsAny(locator, "/*") {
		return func(ctx context.Context) (*time.Time, error) {
			return newestMtime(locator)
		}
	}
	if locator == "systemd" {
		return func(ctx context.Context) (*time.Time, error) {
			return sysLast(ctx, name)
		}
	}
	return func(ctx context.Context) (*time.Time, error) {
		return nil, fmt.Errorf("未知定位形 %q（文件径/glob 或 systemd）", locator)
	}
}

// newestMtime glob/径匹配件最大 mtime（零匹配=nil→零产出史升件）。
func newestMtime(pattern string) (*time.Time, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob 形坏 %q: %w", pattern, err)
	}
	var latest *time.Time
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil {
			continue // 竞态消失件跳过不误伤
		}
		mt := fi.ModTime()
		if latest == nil || mt.After(*latest) {
			latest = &mt
		}
	}
	return latest, nil
}

// sysLastTrigger systemctl LastTriggerUSec 读缝——user 总线先探，
// LoadState=not-found 回落 system 总线（o-backup-* 在 /etc/systemd/
// system 域，user 域查无此件——双域并存是 OCI 现状）。值形双解：
// µs epoch 整数（新 systemd）或人读日期串（本机 systemd 255 实测）。
func sysLastTrigger(ctx context.Context, unit string) (*time.Time, error) {
	for _, scope := range [][]string{{"--user"}, {}} {
		args := append(scope, "show", unit+".timer",
			"-p", "LoadState", "-p", "LastTriggerUSec")
		out, err := exec.CommandContext(ctx, "systemctl", args...).Output()
		if err != nil {
			continue
		}
		var load, trig string
		for _, ln := range strings.Split(string(out), "\n") {
			k, v, _ := strings.Cut(strings.TrimSpace(ln), "=")
			switch k {
			case "LoadState":
				load = v
			case "LastTriggerUSec":
				trig = v
			}
		}
		if load == "not-found" || load == "" {
			continue // 本域无此件——试下一域
		}
		if trig == "" || trig == "n/a" {
			return nil, nil // 从未触发=零产出史升件
		}
		if us, err := strconv.ParseInt(trig, 10, 64); err == nil && us > 0 {
			t := time.UnixMicro(us)
			return &t, nil
		}
		if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", trig); err == nil {
			return &t, nil
		}
		return nil, fmt.Errorf("LastTriggerUSec 形不可解 %q", trig)
	}
	return nil, fmt.Errorf("systemctl %s.timer 双域皆读败", unit)
}

// probeOrgans 器官死活账一拍：manifest 空=不探（防误报钦定）。
func (s *Sentinel) probeOrgans(ctx context.Context) {
	raw, err := s.cfg.Get(organManifestKey)
	if err != nil {
		return // 键缺=未登记——不探
	}
	var entries []string
	if err := json.Unmarshal(raw, &entries); err != nil || len(entries) == 0 {
		return // 坏件/空单=不探
	}
	maxAge := time.Duration(organMaxAgeDef * float64(time.Hour))
	if h, err := s.cfg.GetFloat(organMaxAgeKey); err == nil && h > 0 {
		maxAge = time.Duration(h * float64(time.Hour))
	}
	thresh := fmt.Sprintf("%gh", maxAge.Hours())
	for _, e := range entries {
		name, locator, err := parseOrganEntry(e)
		if err != nil {
			log.Printf("WARN [sentinel] 器官名单行坏跳过 %q: %v", e, err)
			continue
		}
		face := organFace(ctx, name, locator, s.sysLast)
		if _, err := selfprobe.Probe(ctx, s.esc, face,
			organSpec(name, maxAge, thresh), s.now); err != nil {
			log.Printf("WARN [sentinel] 器官 %s 读败: %v", name, err)
		}
	}
}
