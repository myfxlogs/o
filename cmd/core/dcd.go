// dcd.go —— dcd 臂+差扫岗装配（T-0102——research 域零 venue 依赖红线：
// feed/probe 闭包在此以 okcli 落地，解析/分类逻辑在域内纯函数）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/research/dcd"
	"o/internal/research/funding"
	"o/internal/research/gapscan"
	"o/internal/research/grammar"
	"o/internal/research/hypothesis"
	"o/internal/trading/okxexec"
)

// dcdFeedFor 双币赢枚举 REST 面（<endpoint_family>/currency-pair → products?uly=
// 全对在架；信封数字 code 由 okxexec flexCode 兼容——坑条 okx-dcd-endpoint-dialect）。
// get429 限频退避重读件（T-0117 根修：boot 两臂同打 DCD 产品端点撞 429
// fail-closed 让位成静默死亡——429/5xx ≤3 次指数退避重试再让位；4xx 不重试）。
func get429(ctx context.Context, okcli *okxexec.Client, path string) ([]byte, error) {
	var raw []byte
	var err error
	for try := 0; try < 4; try++ {
		raw, err = okcli.Get(ctx, path)
		if err == nil {
			return raw, nil
		}
		msg := err.Error()
		if !strings.Contains(msg, "429") && !strings.Contains(msg, "http 5") {
			return nil, err // 非限频/服务端错不重试
		}
		if try == 3 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, err
		case <-time.After(time.Duration(1<<try) * time.Second): // sched-exempt-ok：429 退避
		}
	}
	return nil, err
}

func dcdFeedFor(okcli *okxexec.Client) dcd.Feed {
	return func(ctx context.Context, s dcd.Surface) ([]dcd.Product, error) {
		if okcli == nil {
			return nil, fmt.Errorf("dcd feed 缺席（凭据缺）")
		}
		raw, err := get429(ctx, okcli, s.EndpointFamily+"/currency-pair")
		if err != nil {
			return nil, err
		}
		pairs, err := dcd.ParseCurrencyPairs(raw)
		if err != nil {
			return nil, err
		}
		var out []dcd.Product
		for i, pr := range pairs {
			if i > 0 {
				time.Sleep(300 * time.Millisecond) // sched-exempt-ok：限流件
				// 退避节拍（429 实证——逐对连发风暴；sfp 族限速紧，非轮询）
			}
			raw2, err := get429(ctx, okcli, fmt.Sprintf("%s/products?baseCcy=%s&quoteCcy=%s&optType=%s",
				s.EndpointFamily, pr.BaseCcy, pr.QuoteCcy, pr.OptType))
			if err != nil {
				return nil, err
			}
			ps, err := dcd.ParseProducts(raw2)
			if err != nil {
				return nil, err
			}
			out = append(out, ps...)
		}
		return out, nil
	}
}

var httpStatusRe = regexp.MustCompile(`^http (\d{3}):`)

// gapProbeFor 差扫面探针（签名 GET+响应形通用提取：JSON 数组长=products 计数、
// avgRate/annualizedYield 正值=收益带位；错误分型=面缺席〔404/405〕/业务拒
// 〔APIError〕/瞬态〔其余上抛——fail-closed 不裁死〕）。
func gapProbeFor(okcli *okxexec.Client) gapscan.Probe {
	return func(ctx context.Context, f gapscan.ProbeFace) (gapscan.ProbeResult, error) {
		if okcli == nil {
			return gapscan.ProbeResult{}, fmt.Errorf("probe 缺席（凭据缺）")
		}
		raw, err := get429(ctx, okcli, f.Endpoint)
		if err != nil {
			var ae *okxexec.APIError
			if errors.As(err, &ae) {
				return gapscan.ProbeResult{}, &gapscan.FaceBizError{Code: ae.Code, Msg: ae.Msg}
			}
			if m := httpStatusRe.FindStringSubmatch(err.Error()); m != nil {
				st, _ := strconv.Atoi(m[1])
				if st == 404 || st == 405 {
					return gapscan.ProbeResult{}, &gapscan.FaceAbsentError{Status: st}
				}
			}
			return gapscan.ProbeResult{}, err
		}
		res := gapscan.ProbeResult{}
		var rows []map[string]any
		if json.Unmarshal(raw, &rows) == nil {
			res.Products = len(rows)
			for _, r := range rows {
				for _, k := range []string{"avgRate", "annualizedYield"} {
					if v, ok := r[k].(string); ok {
						if fv, e2 := strconv.ParseFloat(v, 64); e2 == nil && fv > 0 {
							res.YieldBearing = true
						}
					}
				}
			}
		} else {
			res.Products = 1 // 非数组形（单对象）——面通
		}
		return res, nil
	}
}

// researchArms 研究域四臂发射（T-0117：funding/dcd/差扫/假想官静默告警同挂
// 业主通道——读件缺席=让位 WARN 不炸 boot）。
func researchArms(ctx context.Context, errc chan<- error, pool *pgxpool.Pool,
	cfg *config.Client, okcli *okxexec.Client, alerter *notify.Alerter) {
	go funding.NewRunner(pool, cfg).AttachAlerter(alerter).Run(ctx, errc)
	go dcd.NewRunner(pool, cfg).AttachFeed(dcdFeedFor(okcli)).
		AttachCoins(okxCoinsFor(pool, okcli)).AttachAlerter(alerter).Run(ctx, errc)
	go gapscan.NewRunner(pool, cfg).AttachProbe(gapProbeFor(okcli)).AttachAlerter(alerter).Run(ctx, errc)
	go hypothesis.NewRunner(pool, cfg).AttachAlerter(alerter).Run(ctx, errc)
	go grammar.NewRunner(pool, cfg).Run(ctx, errc) // T-0120 文法枚举臂
}
