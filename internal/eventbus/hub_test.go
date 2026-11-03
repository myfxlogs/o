package eventbus

import (
	"testing"

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
	if (<-ch).MaxId != 7 || (<-ch2).MaxId != 7 {
		t.Fatal("多 watcher 扇出缺")
	}
	select {
	case <-ch: // ch 已收 max=7 补帧
	default:
	}
}
