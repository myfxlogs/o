package ctx

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// —— mql5 经济日历（vendored ark internal/service/news/fetcher.go——XHR 端点
// 择优钉：POST /en/economic-calendar/content 免凭据 JSON 直出，优于 HTML 拉页；
// High=3→md_macro_event upsert source='mql5'（替种子不删——macroKind 词集闭集
// 域外归 ctx）；中低→md_ctx_fact kind='macro'）——

type mql5Resp []struct {
	EventID int    `json:"eventId"`
	Event   string `json:"event"`
	CCY     string `json:"currency"`
	Imp     int    `json:"importance"`
	T       string `json:"time"`
}

func (r *Runner) collectMql5(c context.Context) error {
	// XHR 择优径：/content POST date=本周（免凭据服务端渲染同源数据直出——
	// ark fetcher.go 同款径；HTML 拉页兜底弃）。
	d := r.now().UTC().Format("2006-01-02")
	b, err := r.fetch(c,
		"https://www.mql5.com/en/economic-calendar/content?date="+d)
	if err != nil {
		return err
	}
	var evs mql5Resp
	if err := json.Unmarshal(b, &evs); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	for _, e := range evs {
		if e.Imp == 3 {
			if kind := macroKind(e.Event); kind != "" {
				if _, err := r.pool.Exec(c, `INSERT INTO md_macro_event(event_date,kind,title,source)
					VALUES($1::timestamptz::date,$2,$3,'mql5')
					ON CONFLICT (event_date) DO UPDATE SET kind=EXCLUDED.kind,
						title=EXCLUDED.title,source=EXCLUDED.source`,
					e.T, kind, e.Event); err != nil {
					r.rej.Inc("write_err")
					return err
				}
				continue
			}
		}
		if err := r.ins(c, "mql5", "macro", e); err != nil {
			return err
		}
	}
	return nil
}

func macroKind(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "fomc") || strings.Contains(n, "fed interest rate") ||
		strings.Contains(n, "federal funds"):
		return "fomc"
	case strings.Contains(n, "cpi") || strings.Contains(n, "consumer price"):
		return "cpi"
	case strings.Contains(n, "nonfarm") || strings.Contains(n, "nfp") ||
		strings.Contains(n, "employment change"):
		return "nfp"
	}
	return ""
}

// —— FRED 宏观序列（vendored ark internal/service/fred/fetcher.go——
// 公共 REST api_key=fred_api_key env 免配则公共档；series 闭集钉）——

var fredSeries = []string{"DGS10", "DGS2", "UNRATE", "CPIAUCSL"}

func (r *Runner) collectFred(c context.Context) error {
	for _, s := range fredSeries {
		b, err := r.fetch(c, "https://api.stlouisfed.org/fred/series/observations?series_id="+s+
			"&sort_order=desc&limit=2&file_type=json")
		if err != nil {
			return err
		}
		var resp struct {
			Obs []struct {
				Date  string `json:"date"`
				Value string `json:"value"`
			} `json:"observations"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			r.rej.Inc("parse_err")
			continue
		}
		if err := r.ins(c, "fred", "macro", map[string]any{"series": s, "obs": resp.Obs}); err != nil {
			return err
		}
	}
	return nil
}

// —— COT 持仓报告（vendored ark internal/service/cot/fetcher.go——CFTC Socrata
// 公共面 https://publicreporting.cftc.gov/resource/jun7-fme8.json——免凭据；周频）——

func (r *Runner) collectCot(c context.Context) error {
	b, err := r.fetch(c, "https://publicreporting.cftc.gov/resource/jun7-fme8.json?$limit=50")
	if err != nil {
		return err
	}
	var rows []map[string]any
	if err := json.Unmarshal(b, &rows); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	return r.ins(c, "cftc", "cot", map[string]any{"rows": rows})
}

// —— 新闻流（vendored ark news 面——mql5 /en/news 页头题段采集 kind='news'；
// 拉页解析粗取 title 行）——

func (r *Runner) collectNews(c context.Context) error {
	b, err := r.fetch(c, "https://www.mql5.com/en/news")
	if err != nil {
		return err
	}
	txt := string(b)
	var titles []string
	// 粗解析 title 标签（页面改版脆——零行如实空拍）
	for _, seg := range strings.Split(txt, "news-item__title") {
		if i := strings.Index(seg, ">"); i >= 0 {
			t := strings.TrimSpace(seg[i+1:])
			if j := strings.Index(t, "<"); j > 0 && len(t[:j]) > 10 && len(titles) < 20 {
				titles = append(titles, t[:j])
			}
		}
	}
	if len(titles) == 0 {
		return nil
	}
	return r.ins(c, "mql5", "news", map[string]any{"titles": titles, "ts": r.now().Format(time.RFC3339)})
}
