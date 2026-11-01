// proc.go —— 进程生死实证（/proc 直读非轮询调用点即查——supervisor pgrep
// 锚同语义：cmdline 含 headless-<tid>-prompt 或 headless-audit-<tid>-prompt；
// DEVIN_MODEL 自目标进程 environ 实读，读件缺=空不猜）。
package main

import (
	"os"
	"strconv"
	"strings"
)

func envOf(raw []byte, key string) string {
	for _, kv := range strings.Split(string(raw), "\x00") {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}

// findProc 票号→在飞 devin 进程实证（pid/底模/在否——run 与 audit 双锚）。
func findProc(tid string) (pid int, model string, ok bool) {
	anchors := []string{"headless-" + tid + "-prompt", "headless-audit-" + tid + "-prompt"}
	es, err := os.ReadDir("/proc")
	if err != nil {
		return 0, "", false
	}
	for _, e := range es {
		p, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		b, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err != nil {
			continue
		}
		cmd := strings.ReplaceAll(string(b), "\x00", " ")
		for _, a := range anchors {
			if strings.Contains(cmd, a) {
				env, _ := os.ReadFile("/proc/" + e.Name() + "/environ")
				return p, envOf(env, "DEVIN_MODEL"), true
			}
		}
	}
	return 0, "", false
}
