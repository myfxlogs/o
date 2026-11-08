package ctx

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"log"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
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
		log.Printf("[ctx] ffcal 拉取败: %v", err) // 静默吞错坑定位灯（T-0311 返修⑤同形）
		return err
	}
	var wk ffWeek
	// windows-1252 声明 charset 病灶根修——Go xml 遇非 UTF-8 声明且
	// CharsetReader=nil 直接报错（T-0429 ffcal 零行史根因）。同坑双面案：
	// T-0183 修 notify/calendar.go 推送面时研究 ctx 采集臂漏扫未修——
	// 接入件照 T-0183 同款（charset.NewReaderLabel），不另造轮。
	dec := xml.NewDecoder(bytes.NewReader(b))
	dec.CharsetReader = charset.NewReaderLabel
	if err := dec.Decode(&wk); err != nil {
		log.Printf("[ctx] ffcal 解析败: %v", err)
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
			key = key[i+1:] // url slug=事件身份
		}
		// T-0444 版本位律（T-0429 fred/frankfurter 同族根修——KB
		// dedupe-key-missing-version-bit-silent-stall）：FF url slug 无日期位，
		// 周/月循环事件（Unemployment Claims/CPI/PPI 类）次周同键撞
		// ON CONFLICT DO NOTHING 恒静默吞新发生——生产实证
		// 11-us-unemployment-claims 冻结 11-05-2026 行。键=slug:事件日期，
		// 新日期=新键=新行；同日重拍仍幂等。slug 空=键空=NULL 不入键
		// （ins 既有径——urlless 事件不互撞）。
		if key != "" {
			key += ":" + e.Date
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

// fredSeries 采集列（T-0429 扩面：macro 四列+equity/commodity 补维——判词
// 六维饥荒四维进水；stooq/Yahoo 对机房 ASN 全拒实证后 equity/commodity 走
// 既有 FRED 臂同列扩面）。kind 跟序列族走——分析席 dim 判词读
// [source][kind]payload 头，kind=维度自述。
var fredSeries = []struct {
	id   string
	kind string
}{
	{"DGS10", "macro"}, {"DGS2", "macro"}, {"UNRATE", "macro"}, {"CPIAUCSL", "macro"},
	{"SP500", "equity"}, {"NASDAQCOM", "equity"},
	{"DCOILWTI", "commodity"}, {"DHHNGSP", "commodity"},
}

func (r *Runner) collectFred(c context.Context) error {
	if r.fredFn != nil {
		if k, err := r.fredFn(c); err == nil {
			r.fredKey = k
		} else {
			log.Printf("[ctx] fred 键取败: %v", err) // T-0311 返修⑤——静默吞错坑定位灯
			r.rej.Inc("fetch_err")
		}
	}
	if r.fredKey == "" {
		return nil
	}
	for _, s := range fredSeries {
		b, err := r.fetch(c, "https://api.stlouisfed.org/fred/series/observations?series_id="+s.id+
			"&api_key="+r.fredKey+"&sort_order=desc&limit=2&file_type=json")
		if err != nil {
			log.Printf("[ctx] fred %s 拉取败: %v", s.id, err) // T-0311 返修⑤——静默吞错坑定位灯
			r.rej.Inc("fetch_err")
			continue // 逐列 fail-soft（T-0429——首列败拖杀余列=断供扩面形，单列败不拖拍）
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
		if len(resp.Obs) == 0 {
			continue // 空观察集无新件可断言——不落行（dedupe 键无日期位）
		}
		// dedupe_key=序列:最新观察日（T-0429 根修——原键=序列名，首拍成功后
		// ON CONFLICT DO NOTHING 恒静默，成功也零新行=「断供」主病灶实锤：
		// 生产止 id976 而键/端点/transport 今日全通）。新观察日=新键=新行，
		// 分析席 id 水印增量可见。
		if err := r.ins(c, "fred", s.kind, s.id+":"+resp.Obs[0].Date,
			map[string]any{"series": s.id, "obs": resp.Obs}); err != nil {
			return err
		}
	}
	return nil
}

// —— frankfurter FX（ECB 日率公共面免凭据——T-0429 fx 补维新源）——
// api.frankfurter.dev/v1/latest：{amount,base,date,rates{}}；date=率日
// （ECB ~16:00 CET 营业日更新）；dedupe_key=base:date——日一行幂等。
// 独立上游（FRED 断供不连坐 fx 维——上游分家=补维本意）。
func (r *Runner) collectFx(c context.Context) error {
	b, err := r.fetch(c, "https://api.frankfurter.dev/v1/latest?base=USD&symbols=EUR,JPY,GBP,CNY")
	if err != nil {
		log.Printf("[ctx] fx 拉取败: %v", err) // 静默吞错坑定位灯（T-0311 返修⑤同形）
		return err
	}
	var resp struct {
		Date  string             `json:"date"`
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		log.Printf("[ctx] fx 解析败: %v", err)
		r.rej.Inc("parse_err")
		return nil
	}
	if resp.Date == "" || len(resp.Rates) == 0 {
		log.Printf("[ctx] fx 空率集——date=%q rates=%d", resp.Date, len(resp.Rates))
		r.rej.Inc("parse_err")
		return nil
	}
	return r.ins(c, "frankfurter", "fx", resp.Base+":"+resp.Date,
		map[string]any{"base": resp.Base, "date": resp.Date, "rates": resp.Rates})
}

// —— COT（CFTC Socrata 公共面免凭据——周频）——
func (r *Runner) collectCot(c context.Context) error {
	// 端点 ark config.go:120 钉（6dca-aqww——TFF 报告面；$order 报告日倒序
	// $limit=50；dedupe_key=行 id 字段幂等）。
	b, err := r.fetch(c, "https://publicreporting.cftc.gov/resource/6dca-aqww.json"+
		"?$order=report_date_as_yyyy_mm_dd%20DESC&$limit=50")
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
