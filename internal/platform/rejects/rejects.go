// Package rejects —— 判定面 rejects 计数门（D-652-A：面×门预注册键集、原子加、
// 零业务副作用；return nil/continue/fail-closed 判定门拒因不可见=静默杀，计数门留痕）。
package rejects

import (
	"log"
	"sync/atomic"
)

type Door struct {
	face string
	keys map[string]*atomic.Int64
}

// New 预注册键集（门=拒因分类键；运行期未注册键拒入并留痕——防计数门名漂移）。
func New(face string, keys ...string) *Door {
	d := &Door{face: face, keys: map[string]*atomic.Int64{}}
	for _, k := range keys {
		d.keys[k] = &atomic.Int64{}
	}
	return d
}

// Inc 拒因 +1（预注册键内原子加；未注册键落 warn——计数门自身可审）。
func (d *Door) Inc(key string) {
	if c, ok := d.keys[key]; ok {
		c.Add(1)
		return
	}
	log.Printf("WARN rejects[%s]: 未注册门键 %q（预注册外拒因——审计数门完整性）", d.face, key)
}

// Snapshot 当前计数快照（观测/status 面只读；键带面名前缀）。
func (d *Door) Snapshot() map[string]int64 {
	out := map[string]int64{}
	for k, c := range d.keys {
		out[d.face+"/"+k] = c.Load()
	}
	return out
}
