// recon.go —— 重连纪律件（升件#18354/mt5-session-conflict-churn 坑根修面）：
// 流死≠会话死——断线先 CheckConnect 复挂（零新 login 避开券商冲突掐）；
// 会话冲突指纹错串照 mtapi 内嵌冷却睡，勿 2s 节奏硬撞垂死会话。
package mt5

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// SetToken 复挂注凭——断线重连前把上会话 token 回灌新 Client（同网关域有效；
// 同时作 sessID 复挂锚：Connect 兜底径以同键再登录，session 冲突签名下
// 仍可能复用网关侧存活会话而非强开新 login）。
func (c *Client) SetToken(tok string) { c.token, c.sessID = tok, tok }

// CheckAlive 会话活性探（CheckConnect——旧 token 活着=流死非会话死，
// 复挂零新 login；死=调用侧落 Connect 新 login 径）。
func (c *Client) CheckAlive(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("会话凭空")
	}
	if c.connM == nil {
		return fmt.Errorf("conn 未立")
	}
	r, err := c.connM.CheckConnect(c.authed(ctx), &mt5pb.CheckConnectRequest{Id: c.token})
	if err != nil {
		return err
	}
	if r.Error != nil {
		return fmt.Errorf("check: %s", r.Error.Message)
	}
	return nil
}

// pauseHintRe mtapi 冷却自白值——「paused for 8285 more milliseconds」/
// 「wait 5000 milliseconds」形内嵌毫秒数。
var pauseHintRe = regexp.MustCompile(`(\d+)\s*(?:more\s+)?millisecond`)

// SessionBackoff 会话冲突指纹分类——垂死会话上强开新 login 被券商掐
// （「closed the socket」）或撞 mtapi 节流（「refusing logins / refused a
// login」）。true=调用侧退避量应抬到返回值（内嵌冷却值+1s 余量；无内嵌
// 值 12s——实测窗 5~8.3s 上界）；false=非常规冲突走普通退避。
func SessionBackoff(err error) (time.Duration, bool) {
	if err == nil {
		return 0, false
	}
	s := err.Error()
	if !strings.Contains(s, "closed the socket") &&
		!strings.Contains(s, "refusing logins") &&
		!strings.Contains(s, "refused a login") &&
		!strings.Contains(s, "paused for") {
		return 0, false
	}
	if m := pauseHintRe.FindStringSubmatch(s); m != nil {
		if ms, e := strconv.Atoi(m[1]); e == nil && ms > 0 {
			return time.Duration(ms)*time.Millisecond + time.Second, true
		}
	}
	return 12 * time.Second, true
}
