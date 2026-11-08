package seatcard

// T-0458 锚 a：渲染哈希同态——同输入两渲哈希同（哈希闸正确性根基）；
// 附：分钟截断（渲染墙钟不击穿哈希闸）+末动 note 截 40 字+班态模板色。

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
	"time"
)

// re 秒级时间字面（HH:MM:SS）——截分律断言面。
var re = regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)

func sampleState() CardState {
	return CardState{
		Seat:          "CTO席",
		Status:        StatusOnDuty,
		Latest:        &EventRow{Seat: "CTO", Ts: base, Event: "裁决", Ticket: "T-0458", Note: "通过——simvenue 矩阵闸齐"},
		Working:       &EventRow{Seat: "CTO", Ts: base.Add(-5 * time.Minute), Event: "领单", Ticket: "T-0458"},
		QueueReview:   3,
		QueueApproved: 5,
		Now:           base,
	}
}

func cardHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestRenderDeterministic(t *testing.T) {
	cs := sampleState()
	c1 := Render(cs)
	c2 := Render(cs)
	if c1 != c2 {
		t.Fatalf("同输入两渲不同体——哈希闸必失效:\n%s\n%s", c1, c2)
	}
	if cardHash(c1) != cardHash(c2) {
		t.Fatal("同输入哈希不同（锚a 破）")
	}

	// 输入变（末动行变）→渲染体与哈希必变。
	cs2 := sampleState()
	cs2.Latest = &EventRow{Seat: "CTO", Ts: base.Add(time.Minute), Event: "收口", Ticket: "T-0458", Note: "移交"}
	if Render(cs2) == c1 {
		t.Fatal("输入变渲染体未变——末动段断链")
	}
	if cardHash(Render(cs2)) == cardHash(c1) {
		t.Fatal("输入变哈希未变（锚a 对面断言）")
	}
}

func TestRenderMinuteTruncation(t *testing.T) {
	cs := sampleState()
	// 同分窗内 Now 漂移（秒级）——卡面逐字节同（哈希闸不击穿）。
	csA := sampleState()
	csA.Now = base.Add(20 * time.Second)
	csB := sampleState()
	csB.Now = base.Add(50 * time.Second)
	if Render(csA) != Render(csB) {
		t.Fatalf("同分窗秒级漂移击穿哈希闸:\n%s\n%s", Render(csA), Render(csB))
	}
	if re.MatchString(Render(cs)) {
		// 秒级时间字面（HH:MM:SS）禁入卡面（截分律）——末动/在办 ts 均分钟粒度。
		t.Fatalf("卡面含秒级时间（截分律破）: %s", Render(cs))
	}
}

func TestRenderNoteClip(t *testing.T) {
	long := strings.Repeat("长", 50)
	cs := sampleState()
	cs.Latest.Note = long
	out := Render(cs)
	if strings.Count(out, "长") > 41 { // 40 字+省略号
		t.Fatalf("note 未截 40 字: %d", strings.Count(out, "长"))
	}
	if !strings.Contains(out, "…") {
		t.Fatal("截断无省略号标")
	}
	// rune 截防中文半字（40 rune 恰截）。
	if got := clipNote(strings.Repeat("字", 40)); got != strings.Repeat("字", 40) {
		t.Fatalf("恰 40 字被误截: %d", len([]rune(got)))
	}
}

func TestRenderSegments(t *testing.T) {
	cs := sampleState()
	cs.Workers = []Worker{{Unit: "o-worker-T-0001.service", TID: "T-0001", Model: "swe-2-high", Since: base.Add(-12 * time.Minute)}}
	out := Render(cs)
	for _, want := range []string{"席位活卡 · CTO席", "开班在岗", "T-0458 领单", "并行活件", "T-0001 · swe-2-high · 已 12 分", "待裁 3 · 待派 5", "末动"} {
		if !strings.Contains(out, want) {
			t.Fatalf("卡面缺段 %q:\n%s", want, out)
		}
	}
	// 无活件=「待派」。
	cs2 := sampleState()
	cs2.Working = nil
	if !strings.Contains(Render(cs2), "在办**：待派") {
		t.Fatalf("无活件应显待派:\n%s", Render(cs2))
	}
}

func TestStatusTemplate(t *testing.T) {
	cases := map[string]string{
		StatusOnDuty: "green", StatusWorking: "blue", StatusBlocked: "orange",
		StatusError: "red", StatusOff: "grey", "未知态": "blue",
	}
	for st, want := range cases {
		if got := statusTemplate(st); got != want {
			t.Fatalf("班态色 %q=%s 期 %s", st, got, want)
		}
	}
}
