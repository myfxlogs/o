package eventbus

import (
	"testing"
	"time"

	"o/internal/gen/eventv1"
)

// 锚：resume 补帧/增扇出/水位回退不帧/多 watcher。
func TestHub(t *testing.T) {
	h := NewHub()
	h.Advance(&eventv1.DataAdvanced{Source: "t1", Kind: "ctx", MaxId: 5})
	h.Advance(&eventv1.DataAdvanced{Source: "t1", Kind: "ctx", MaxId: 3}) // 回退不帧
	ch, _ := h.Watch(map[string]int64{"t1": 2})
	fr := <-ch // resume=2<5 补帧
	if fr.MaxId != 5 {
		t.Fatalf("补帧 max=%d", fr.MaxId)
	}
	ch2, _ := h.Watch(map[string]int64{})
	h.Advance(&eventv1.DataAdvanced{Source: "t1", Kind: "ctx", MaxId: 7})
	// resume 补帧(5)与活帧(7)同管无序同入 ch2 缓冲——消费侧按 MaxId
	// 去重语义容忍先收 5，逐帧读到 7 即证扇出（限时不死等）。
	if (<-ch).MaxId != 7 {
		t.Fatal("活 watcher 扇出缺")
	}
	deadline := time.After(2 * time.Second) // sched-exempt-ok: 测试限时不死等——期界等待非周期调度
	for {
		select {
		case fr := <-ch2:
			if fr.MaxId == 7 {
				return
			}
		case <-deadline:
			t.Fatal("新 watcher 扇出缺——max=7 未达")
		}
	}
}
