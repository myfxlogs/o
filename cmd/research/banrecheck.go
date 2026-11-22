// banrecheck.go —— 禁前提复核拍核径装配（O-33393③——research 域零 venue
// 依赖红线：venue 调用闭包在 cmd 落地，kind 派核；前提证伪升件/仍立
// 复核行/不可核不产行——复核拍独立提案面，眼死了探照灯也得亮）。
package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/research/banrecheck"
	"o/internal/research/dcd"
	"o/internal/trading/okxexec"
)

// banrecheckCheckFor 禁前提重核缝（kind 派核：
//
//	venue_capability=能力面端点族实探（dual-investment=DCD currency-pair
//	  目录；余族=endpoint_family 根径通形——通=前提证伪）；
//	apr_regime=产品面重取 APR 带对拍判据阈（仅 dual-investment 有判据件，
//	  余族 unknown 不产行）；
//	owner_order/未识类=unknown 不产行（owner_order 另在臂内先拦——业主线
//	  不归机器翻案仅龄报）。
func banrecheckCheckFor(okcli authGetter, cfg *config.Client, rej *rejects.Door) banrecheck.CheckFunc {
	return func(ctx context.Context, r banrecheck.Row) (banrecheck.Result, error) {
		if okcli == nil {
			return banrecheck.Result{Verdict: "unknown", Detail: "凭据缺——okcli 缺席"}, nil
		}
		switch r.BanPremiseKind {
		case "venue_capability":
			return probeVenueCap(ctx, okcli, r)
		case "apr_regime":
			return probeAprRegime(ctx, okcli, cfg, rej, r)
		default:
			return banrecheck.Result{Verdict: "unknown",
				Detail: "kind=" + r.BanPremiseKind + " 无派核径"}, nil
		}
	}
}

// probeVenueCap venue_capability 前提核——「面缺类/不可用」命题重探：
// dual-investment 走 DCD 目录实形（currency-pair 对在架=前提证伪）；
// 余族走 endpoint_family 根径通形（通=证伪）。业务拒/面缺席=前提仍立；
// 瞬态错上抛下拍重核（fail-closed 不误裁）。
func probeVenueCap(ctx context.Context, okcli authGetter, r banrecheck.Row) (banrecheck.Result, error) {
	var path string
	if r.ProductFamily == "dual-investment" {
		path = r.EndpointFamily + "/currency-pair"
	} else if strings.HasPrefix(r.EndpointFamily, "/") {
		path = r.EndpointFamily
	} else {
		return banrecheck.Result{Verdict: "unknown",
			Detail: "endpoint_family 非 REST 径形（" + r.EndpointFamily + "）"}, nil
	}
	raw, err := get429(ctx, okcli, path)
	if err != nil {
		var ae *okxexec.APIError
		if errors.As(err, &ae) {
			return banrecheck.Result{Verdict: "upheld",
				Detail: fmt.Sprintf("%s 业务拒 code=%s msg=%s——能力面仍拒", path, ae.Code, ae.Msg)}, nil
		}
		if m := httpStatusRe.FindStringSubmatch(err.Error()); m != nil {
			st, _ := strconv.Atoi(m[1])
			if st == 404 || st == 405 {
				return banrecheck.Result{Verdict: "upheld",
					Detail: fmt.Sprintf("%s http %d——面缺席仍立", path, st)}, nil
			}
		}
		return banrecheck.Result{}, err // 瞬态/认证败上抛下拍重核
	}
	if r.ProductFamily == "dual-investment" {
		pairs, perr := dcd.ParseCurrencyPairs(raw)
		if perr != nil {
			return banrecheck.Result{Verdict: "unknown",
				Detail: "currency-pair 解析败: " + perr.Error()}, nil
		}
		if len(pairs) == 0 {
			return banrecheck.Result{Verdict: "upheld",
				Detail: path + " 在架对=0——目录面空仍立"}, nil
		}
		return banrecheck.Result{Verdict: "falsified",
			Detail: fmt.Sprintf("%s 在架对=%d（base×quote×optType 三元组实供）——缺类前提证伪", path, len(pairs))}, nil
	}
	if len(strings.TrimSpace(string(raw))) == 0 || string(raw) == "[]" {
		return banrecheck.Result{Verdict: "upheld",
			Detail: path + " 空载——面通而供给空仍立"}, nil
	}
	return banrecheck.Result{Verdict: "falsified",
		Detail: fmt.Sprintf("%s 通且载非空（%dB）——缺类前提证伪", path, len(raw))}, nil
}

// probeAprRegime apr_regime 前提核——「收益恒拒」命题重核：产品面重取
// APR 带对拍判据阈（research.dcd.min_apr——判据档沿用枚举臂同一配置
// 键不另立判据）。仅 dual-investment 有判据件；余族 unknown。
func probeAprRegime(ctx context.Context, okcli authGetter, cfg *config.Client,
	rej *rejects.Door, r banrecheck.Row) (banrecheck.Result, error) {
	if r.ProductFamily != "dual-investment" || r.EndpointFamily == "" {
		return banrecheck.Result{Verdict: "unknown",
			Detail: "apr_regime 判据件仅 dual-investment 面在册"}, nil
	}
	feed := dcdFeedFor(okcli, rej)
	ps, err := feed(ctx, dcd.Surface{Venue: r.Venue, Surface: r.Surface,
		EndpointFamily: r.EndpointFamily, ProductFamily: r.ProductFamily, Enabled: true})
	if err != nil {
		return banrecheck.Result{}, err // 断供上抛下拍重核（fail-closed）
	}
	if len(ps) == 0 {
		return banrecheck.Result{Verdict: "upheld",
			Detail: "产品面在架=0——收益恒拒前提仍立（目录空）"}, nil
	}
	minApr := 0.04
	if v, e := cfg.GetFloat("research.dcd.min_apr"); e == nil && v > 0 {
		minApr = v
	}
	var maxApr float64
	var maxID string
	var maxExp time.Time
	for _, p := range ps {
		if p.AnnualizedYield > maxApr {
			maxApr, maxID, maxExp = p.AnnualizedYield, p.ProductID, p.Expiry
		}
	}
	if maxApr >= minApr {
		return banrecheck.Result{Verdict: "falsified",
			Detail: fmt.Sprintf("产品面在架=%d 最高APR=%.4f（%s 到期 %s）≥判据 %.4f——恒拒前提证伪",
				len(ps), maxApr, maxID, maxExp.Format("2006-01-02"), minApr)}, nil
	}
	return banrecheck.Result{Verdict: "upheld",
		Detail: fmt.Sprintf("产品面在架=%d 最高APR=%.4f<判据 %.4f——恒拒前提仍立",
			len(ps), maxApr, minApr)}, nil
}
