package okxpriv

import (
	"context"
	"fmt"

	"o/internal/platform/signutil"
)

// cexvenue.go —— CexVenue 接口缝（O-37450——设计 SSOT
// /opt/o-research/designs/venue-multi-account.md D5「CEX 同形」）。
// 窄面两职责：verify=验连摘要（SubmitBrokerCred verify-then-write 同源
// 消费——册内 venue fail-closed 不猜端点）；collect=本包 Supervisor
// roster 迭代私有采集（proto='cex' 活跃户每户一 WS）。exec=执行面
// O-37449 落地即兑现（本票立缝不实现——interface 只载已落地面）。
// okx=首实现，币安=后续票第二实现——新所=新 adapter 登记+归族前缀
// 约定，存量户零码。
type CexVenue interface {
	// Name 交易所族名（归族判据——venue token 前缀约定同 venueMatchOKX）。
	Name() string
	// Verify 验连摘要（凭据三件→venue 只读端实测；err=验连败拒写）。
	Verify(ctx context.Context, creds signutil.Credentials) (string, error)
}

// cexAdapter CEX adapter 注册形（verify 由装配层注入——数据域不反向
// 依赖交易域，cmd/core 接 okxexec 实现；nil=未接线 fail-closed）。
type cexAdapter struct {
	name   string
	verify func(ctx context.Context, creds signutil.Credentials) (string, error)
}

func (a cexAdapter) Name() string { return a.name }
func (a cexAdapter) Verify(ctx context.Context, creds signutil.Credentials) (string, error) {
	if a.verify == nil {
		return "", fmt.Errorf("cex adapter %s verify 未接线——装配层须 RegisterCexVerify 注入", a.name)
	}
	return a.verify(ctx, creds)
}

// cexVenues venue 族→adapter 注册表（首档 okx——他所进册须先落
// Verify 实现+归族前缀登记，fail-closed 不猜端点同 cexVenueSet 律）。
var cexVenues = []CexVenue{cexAdapter{name: "okx"}}

// CexVenueFor venue token→adapter（归族=token 前缀匹配注册表 Name——
// 'okx'/'okx_read'/'okx2' 同族命中 okx adapter；册外=ok=false 调用方
// fail-closed 拒写，同 SubmitBrokerCred cexVenueSet 闸位）。
func CexVenueFor(venue string) (CexVenue, bool) {
	for _, v := range cexVenues {
		if len(venue) >= len(v.Name()) && venue[:len(v.Name())] == v.Name() {
			return v, true
		}
	}
	return nil, false
}

// RegisterCexVerify 装配层注入 venue 族 verify 实现（cmd/core 接
// okxexec.BalanceEquity——幂等覆写=测试重装配）。
func RegisterCexVerify(name string, fn func(ctx context.Context, creds signutil.Credentials) (string, error)) {
	for i, v := range cexVenues {
		if v.Name() == name {
			cexVenues[i] = cexAdapter{name: name, verify: fn}
			return
		}
	}
	cexVenues = append(cexVenues, cexAdapter{name: name, verify: fn})
}
