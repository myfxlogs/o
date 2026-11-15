package main

import "testing"

// depSegSha 指纹件判据——只吐 sha 不吐龄（A1 实证:「sha(龄)」拼指纹=
// 分钟变=gaugeDue 6h 闸永火,self_state 4522帧/30d）。表驱钉死:
// 合法 sha 形出 12 位;非标/短 sha/空行/缺标记一律 "?"——
// 指纹件宁标 ? 不含时漂串,闸语义靠「结构变才重报」保。
func TestDepSegSha(t *testing.T) {
	for _, c := range []struct{ note, want string }{
		{"deploy_image o-core:de8ba7650d80 rc=0", "de8ba7650d80"},
		{"o-core:abbbf192d031 healthy", "abbbf192d031"},
		{"x o-core:54c01f3be11aa9f0c1d2e3 rc=0 ok", "54c01f3be11a"},
		{"deploy rc=0 no-sha-line", "?"},
		{"o-core:abc123 rc=0", "?"}, // sha<12 位=截断不安全
		{"", "?"},
		{"o-core:ZZZZZZZZZZZZ", "?"}, // 非 hex 不命中 regex
	} {
		if got := depSegSha(c.note); got != c.want {
			t.Errorf("depSegSha(%q)=%q want %q", c.note, got, c.want)
		}
	}
	// 判据钉死:同 note 重取恒同——时漂件（age）无进入径。
	n := "deploy_image o-core:de8ba7650d80 rc=0"
	if depSegSha(n) != depSegSha(n) {
		t.Error("depSegSha 非纯函数——指纹件含时漂=闸报废")
	}
}
