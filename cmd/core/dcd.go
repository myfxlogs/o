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
	"time"

	"o/internal/research/dcd"
	"o/internal/research/gapscan"
	"o/internal/trading/okxexec"
)

// dcdFeedFor 双币赢枚举 REST 面（<endpoint_family>/currency-pair → products?uly=
// 全对在架；信封数字 code 由 okxexec flexCode 兼容——坑条 okx-dcd-endpoint-dialect）。
func dcdFeedFor(okcli *okxexec.Client) dcd.Feed {
	return func(ctx context.Context, s dcd.Surface) ([]dcd.Product, error) {
		if okcli == nil {
			return nil, fmt.Errorf("dcd feed 缺席（凭据缺）")
		}
		raw, err := okcli.Get(ctx, s.EndpointFamily+"/currency-pair")
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
			raw2, err := okcli.Get(ctx, fmt.Sprintf("%s/products?baseCcy=%s&quoteCcy=%s&optType=%s",
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
		raw, err := okcli.Get(ctx, f.Endpoint)
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
