package ctx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// —— mql5 经济日历（vendored 真件重写——源 ark-intelligent
// internal/service/news/fetcher.go：POST /en/economic-calendar/content
// L243 端点+form L236-241（date_mode/from/to/importance=15 L57-58/currencies
// =131071 L61）+六头 L248-253（Content-Type/UA/Referer/X-Requested-With/
// Accept/Origin 缺→403）+字段 Id/EventName/Importance(string)/FullDate
// L38-48（FullDate 以 UTC 解析——ark L66-70 注释实证 UTC 非 ET，以
// time.ParseInLocation UTC 钉））——

const mql5Endpoint = "https://www.mql5.com/en/economic-calendar/content"

type mql5Event struct {
	ID         int    `json:"Id"`
	EventName  string `json:"EventName"`
	Importance string `json:"Importance"`
	CCY        string `json:"CurrencyCode"`
	FullDate   string `json:"FullDate"`
	Actual     string `json:"ActualValue"`
	Forecast   string `json:"ForecastValue"`
	Previous   string `json:"PreviousValue"`
}

// collectMql5 本周窗拉页（date_mode=1 from/to 本周 UTC——ark getWeekRange
// L82-97 同款 WIB 周界换算；High→md_macro_event upsert 同日优先级防互踩；
// 中低/无 kind→md_ctx_fact dedupe_key=eventId）。
func (r *Runner) collectMql5(c context.Context) error {
	now := r.now().UTC()
	for now.Weekday() != time.Monday {
		now = now.AddDate(0, 0, -1)
	}
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 7)
	form := url.Values{}
	form.Set("date_mode", "1")
	form.Set("from", from.Format("2006-01-02T15:04:05"))
	form.Set("to", to.Format("2006-01-02T15:04:05"))
	form.Set("importance", "15")
	form.Set("currencies", "131071")
	raw, err := r.fetchForm(c, mql5Endpoint, form)
	if err != nil {
		return err
	}
	var evs []mql5Event
	if err := json.Unmarshal(raw, &evs); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	for _, e := range evs {
		if strings.EqualFold(e.Importance, "high") {
			if kind := macroKind(e.EventName); kind != "" {
				dt, _ := time.ParseInLocation("2006-01-02T15:04:05", e.FullDate, time.UTC)
				if _, err := r.pool.Exec(c, `INSERT INTO md_macro_event(event_date,kind,title,source)
					VALUES($1,$2,$3,'mql5')
					ON CONFLICT (event_date) DO UPDATE SET kind=EXCLUDED.kind,
						title=EXCLUDED.title,source=EXCLUDED.source
					WHERE md_macro_event.source='seed'
					   OR EXCLUDED.kind='fomc'
					   OR (EXCLUDED.kind='cpi' AND md_macro_event.kind<>'fomc')`,
					dt, kind, e.EventName); err != nil {
					r.rej.Inc("write_err")
					return err
				}
				continue
			}
		}
		if err := r.ins(c, "mql5", "macro", strconv.Itoa(e.ID), e); err != nil {
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

// fetchForm POST 表单取件（六头——ark L248-253：免凭据但缺头=403；Fetcher
// 缝仅 GET 故本径内联 http——测试经 stubClient 同形可桩）。
func (r *Runner) fetchForm(c context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(c, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.mql5.com/en/economic-calendar")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Origin", "https://www.mql5.com")
	if r.post != nil {
		return r.post(c, endpoint, form.Encode())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &fetchErr{resp.StatusCode}
	}
	b := make([]byte, 0)
	buf := make([]byte, 64<<10)
	for {
		n, e := resp.Body.Read(buf)
		b = append(b, buf[:n]...)
		if e != nil {
			break
		}
	}
	return b, nil
}

type fetchErr struct{ code int }

func (e *fetchErr) Error() string { return "status " + strconv.Itoa(e.code) }

// —— FRED（vendored fetcher.go——api_key 在才采（键缺=整源跳过如实；
// env FRED_API_KEY 读件）。series 闭集钉）——

var fredSeries = []string{"DGS10", "DGS2", "UNRATE", "CPIAUCSL"}

func (r *Runner) collectFred(c context.Context) error {
	key := r.fredKey
	if key == "" {
		return nil // 键缺=整源跳过如实（免配公共档不冒限速）
	}
	for _, s := range fredSeries {
		b, err := r.fetch(c, "https://api.stlouisfed.org/fred/series/observations?series_id="+s+
			"&api_key="+key+"&sort_order=desc&limit=2&file_type=json")
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
	b, err := r.fetch(c, "https://publicreporting.cftc.gov/resource/jun7-fme8.json?$limit=50")
	if err != nil {
		return err
	}
	var rows []map[string]any
	if err := json.Unmarshal(b, &rows); err != nil {
		r.rej.Inc("parse_err")
		return nil
	}
	return r.ins(c, "cftc", "cot", "", map[string]any{"rows": rows})
}

// —— 新闻流（mql5 /en/news 页头题段 kind='news'——粗解析零行如实空拍）——
func (r *Runner) collectNews(c context.Context) error {
	b, err := r.fetch(c, "https://www.mql5.com/en/news")
	if err != nil {
		return err
	}
	txt := string(b)
	var titles []string
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
	return r.ins(c, "mql5", "news", r.now().UTC().Format("2006-01-02"),
		map[string]any{"titles": titles, "ts": r.now().Format(time.RFC3339)})
}
