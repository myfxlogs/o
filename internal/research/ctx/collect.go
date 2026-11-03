package ctx

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"strings"
	"time"
)

// —— 经济日历（返修③换源：mql5 宿主对机房 ASN 整站 403——POST XHR/六头
// 全失效死源弃；改 ForexFactory 周历 XML nfs.faireconomy.media
// /ff_calendar_thisweek.xml 免凭据双盒实测 200——GET 直 unmarshal；
// High→md_macro_event source='ffcal'（优先级 WHERE+种子兜底不动）；
// 余→md_ctx_fact dedupe_key=url slug）——

type ffWeek struct {
	Events []ffEvent `xml:"event"`
}
type ffEvent struct {
	Title    string `xml:"title"`
	CCY      string `xml:"country"`
	Date     string `xml:"date"`   // "11-03-2026" MM-DD-YYYY
	T        string `xml:"time"`   // "3:30am" ET | "All Day"
	Impact   string `xml:"impact"` // High|Medium|Low|Holiday
	URL      string `xml:"url"`
	Forecast string `xml:"forecast"`
	Previous string `xml:"previous"`
}

func (r *Runner) collectCalendar(c context.Context) error {
	b, err := r.fetch(c, "https://nfs.faireconomy.media/ff_calendar_thisweek.xml")
	if err != nil {
		return err
	}
	var wk ffWeek
	if err := xml.Unmarshal(b, &wk); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	for _, e := range wk.Events {
		if strings.EqualFold(e.Impact, "High") {
			if kind := macroKind(e.Title); kind != "" {
				dt, ok := ffDate(e.Date, e.T)
				if !ok {
					r.rej.Inc("parse_err")
					continue // 解析败=行跳不拖拍
				}
				if _, err := r.pool.Exec(c, `INSERT INTO md_macro_event(event_date,kind,title,source)
					VALUES($1,$2,$3,'ffcal')
					ON CONFLICT (event_date) DO UPDATE SET kind=EXCLUDED.kind,
						title=EXCLUDED.title,source=EXCLUDED.source
					WHERE md_macro_event.source='seed'
					   OR EXCLUDED.kind='fomc'
					   OR (EXCLUDED.kind='cpi' AND md_macro_event.kind<>'fomc')`,
					dt, kind, e.Title); err != nil {
					r.rej.Inc("write_err")
					return err
				}
				continue
			}
		}
		key := e.URL
		if i := strings.LastIndex(key, "/"); i >= 0 {
			key = key[i+1:] // url slug=dedupe 唯一键
		}
		if err := r.ins(c, "ffcal", "macro", key, e); err != nil {
			return err
		}
	}
	return nil
}

// ffDate "11-03-2026"+"3:30am" ET→UTC；"All Day"/空=仅日期落；败=false。
func ffDate(d, t string) (time.Time, bool) {
	base, err := time.Parse("01-02-2006", d)
	if err != nil {
		return time.Time{}, false
	}
	t = strings.TrimSpace(strings.ToLower(t))
	if t == "" || t == "all day" || t == "tentative" {
		return base.UTC(), true
	}
	et, err := time.LoadLocation("America/New_York")
	if err != nil {
		et = time.FixedZone("ET", -5*3600)
	}
	tm, err := time.Parse("3:04pm", t)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(base.Year(), base.Month(), base.Day(),
		tm.Hour(), tm.Minute(), 0, 0, et).UTC(), true
}

func macroKind(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "fomc") || strings.Contains(n, "fed interest rate") ||
		strings.Contains(n, "federal funds") || strings.Contains(n, "fed funds"):
		return "fomc"
	case strings.Contains(n, "cpi") || strings.Contains(n, "consumer price"):
		return "cpi"
	case strings.Contains(n, "nonfarm") || strings.Contains(n, "nfp") ||
		strings.Contains(n, "employment change") || strings.Contains(n, "payroll"):
		return "nfp"
	}
	return ""
}

// —— FRED（api_key 在才采——键缺=整源跳过如实；凭据归 pg cfg.Secret 注入）——

var fredSeries = []string{"DGS10", "DGS2", "UNRATE", "CPIAUCSL"}

func (r *Runner) collectFred(c context.Context) error {
	if r.fredKey == "" {
		return nil
	}
	for _, s := range fredSeries {
		b, err := r.fetch(c, "https://api.stlouisfed.org/fred/series/observations?series_id="+s+
			"&api_key="+r.fredKey+"&sort_order=desc&limit=2&file_type=json")
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
		if err := r.ins(c, "fred", "macro", s, map[string]any{"series": s, "obs": resp.Obs}); err != nil {
			return err
		}
	}
	return nil
}

// —— COT（CFTC Socrata 公共面免凭据——周频）——
func (r *Runner) collectCot(c context.Context) error {
	// 端点 ark config.go:120 钉（6dca-aqww——TFF 报告面；$order 报告日倒序
	// $limit=50；dedupe_key=行 id 字段幂等）。
	b, err := r.fetch(c, "https://publicreporting.cftc.gov/resource/6dca-aqww.json"+
		"?$order=report_date_as_yyyy_mm_dd DESC&$limit=50")
	if err != nil {
		return err
	}
	var rows []map[string]any
	if err := json.Unmarshal(b, &rows); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	for _, row := range rows {
		key, _ := row["id"].(string) // 行 id 字段=dedupe 键（缺=NULL 不入键）
		if err := r.ins(c, "cftc", "cot", key, row); err != nil {
			return err
		}
	}
	return nil
}

// collectOI OKX 公开 OI（/api/v5/public/open-interest?instType=SWAP 全 USDT
// 永续一拍 kind='oi' source='okx_public'；dedupe_key=日戳幂等）。
func (r *Runner) collectOI(c context.Context) error {
	b, err := r.fetch(c, "https://www.okx.com/api/v5/public/open-interest?instType=SWAP")
	if err != nil {
		return err
	}
	var resp struct {
		Data []struct {
			InstID string `json:"instId"`
			Oi     string `json:"oi"`
			OiCcy  string `json:"oiCcy"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	return r.ins(c, "okx_public", "oi", r.now().UTC().Format("2006-01-02"),
		map[string]any{"rows": resp.Data})
}
