package cmc

import (
	"testing"
	"time"
)

// reauthWait 时序闸——auth-JWT ttl=300s 实采：到期前 reauthMargin 秒续期，
// 贴线 exp 钳 reauthMinWait 防热环，解析败回退 reauthDefault。
func TestReauthWait(t *testing.T) {
	cases := []struct {
		name string
		jwt  string
		min  time.Duration
		max  time.Duration
	}{
		{"300s新鲜JWT→255s档", fakeJWT(time.Now().Add(300 * time.Second)), 250 * time.Second, 260 * time.Second},
		{"剩40s→钳下限30s", fakeJWT(time.Now().Add(40 * time.Second)), 29 * time.Second, 31 * time.Second},
		{"已过期→下限", fakeJWT(time.Now().Add(-10 * time.Second)), 29 * time.Second, 31 * time.Second},
		{"非法串→默认240s", "not-a-jwt", 239 * time.Second, 241 * time.Second},
	}
	for _, c := range cases {
		got := reauthWait(c.jwt)
		if got < c.min || got > c.max {
			t.Errorf("%s: reauthWait=%v 不在 [%v,%v]", c.name, got, c.min, c.max)
		}
	}
}
