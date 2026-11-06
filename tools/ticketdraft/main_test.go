package main

import (
	"strings"
	"testing"
)

// TestRender 模板槽填充断言（确定性——{{KEY}} 字面全替换零残留；
// 候选态/证据引/红线三件票面钦定形态在件）。
func TestRender(t *testing.T) {
	tpl := "# {{TID}} {{TITLE}}\n签名={{SIGNATURE}} 键={{LESION_KEY}}\n证={{EVIDENCE_IDS}}\n" +
		"> {{EVIDENCE_NOTE}}\n{{EVIDENCE_QUERY}}\n域={{DOMAIN_HINT}} 类={{KIND}} 案={{DRAFT_ID}}"
	p := payload{
		Slug: "guance-xvenue-carry", Title: "探针升件自动起草：观测盲@xvenue-carry",
		Signature: "观测盲:xvenue-carry", Kind: "观测盲", LesionKey: "xvenue-carry",
		EvidenceIDs:   []int64{7, 9},
		EvidenceNote:  "族沉默 xvenue-carry：信号活而评估零",
		EvidenceQuery: "SELECT id FROM ops_event WHERE 1=1",
		DomainHint:    "评估×信号链面",
	}
	out := render(tpl, p, "T-9001", "3")
	for _, frag := range []string{"# T-9001 探针升件自动起草：观测盲@xvenue-carry",
		"签名=观测盲:xvenue-carry", "键=xvenue-carry", "证=#7,#9",
		"SELECT id FROM ops_event", "域=评估×信号链面", "类=观测盲", "案=3"} {
		if !strings.Contains(out, frag) {
			t.Fatalf("渲染缺 %q:\n%s", frag, out)
		}
	}
	if strings.Contains(out, "{{") {
		t.Fatalf("槽位残留未填:\n%s", out)
	}
}

// TestRenderFullTemplate 真模板件全槽填充（docs/templates/ticket-draft.md
// 契面——候选态标+证据引节+红线继承模板齐=票面钦②形态）。
func TestRenderFullTemplate(t *testing.T) {
	tplb, err := readTemplate()
	if err != nil {
		t.Skipf("模板件相对径不可达（非仓根跑）: %v", err)
	}
	tpl := string(tplb)
	p := payload{
		Slug: "x", Title: "探针升件自动起草：写败@LIFECYCLE-HUNGER",
		Signature: "写败:LIFECYCLE-HUNGER", Kind: "写败", LesionKey: "LIFECYCLE-HUNGER",
		EvidenceIDs: []int64{42}, EvidenceNote: "hungertick respond 败: 桩",
		EvidenceQuery: "SELECT id,ts,seat,note FROM ops_event WHERE event='升件' AND esc_class='写败' AND esc_scope='LIFECYCLE-HUNGER' ORDER BY id DESC LIMIT 20",
		DomainHint:    "lifecycle 饥饿拍写径",
	}
	out := render(tpl, p, "T-9002", "5")
	for _, frag := range []string{"状态：候选", "ops_event 行 id）：#42",
		"疑似根域", "验收锚", "红线", "结果回填", "T-9002"} {
		if !strings.Contains(out, frag) {
			t.Fatalf("票件形态缺 %q", frag)
		}
	}
	if strings.Contains(out, "{{") {
		t.Fatalf("槽位残留未填:\n%s", out)
	}
}
