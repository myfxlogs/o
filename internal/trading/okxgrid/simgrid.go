// simgrid.go —— OKX tradingBot/grid venue 仿真（O-41370 grid_spot 演习径）：
// bot 创建→挂单→行情喂入→kill 全径仿真，Doer 面=okxgrid.Adapter 产线
// 同径注入缝（Get/Post 返 data 段 JSON——okxexec.Client 信封解同约）。
// 网格引擎=最小诚实模型：档位价网格+买低卖高挂单对称+成交翻转补位
// （买成→上档补卖/卖成→下档补买）——只证「bot 拉起→挂单在册→喂价
// 成交→停 bot」执行径产能，venue 侧网格引擎正确性归 OKX 实证面。
package okxgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SimGrid tradingBot/grid 全端点仿真 venue（纸面钱域——零真账）。
type SimGrid struct {
	mu   sync.Mutex
	bots map[string]*simBot
	seq  int
	px   float64 // 当前喂入价（Feed 写——缺省取带中值）
}

// NewSimGrid 仿真 venue 构造（px=起始行情价——0=首个 Launch 用带中值）。
func NewSimGrid(px float64) *SimGrid {
	return &SimGrid{bots: map[string]*simBot{}, px: px}
}

// Feed 行情喂入（演习径钉价——网格成交触发源）。
func (s *SimGrid) Feed(px float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.px = px
	for _, b := range s.bots {
		if b.state == "running" {
			b.tickLocked(px)
		}
	}
}

// RunningCount 在飞 bot 数（对账锚——kill 后应归零）。
func (s *SimGrid) RunningCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, b := range s.bots {
		if b.state == "running" {
			n++
		}
	}
	return n
}

// Get 读面（path 含 query——okxgrid.Adapter get 传全串）。
func (s *SimGrid) Get(_ context.Context, pathWithQuery string) (json.RawMessage, error) {
	u, err := url.Parse("https://sim.local" + pathWithQuery)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case strings.HasSuffix(u.Path, "/orders-algo-details"):
		b := s.bots[q.Get("algoId")]
		if b == nil {
			return json.Marshal([]any{})
		}
		return json.Marshal([]any{b.detailsJSON()})
	case strings.HasSuffix(u.Path, "/orders-algo-pending"):
		var out []any
		for _, b := range s.bots {
			if b.state == "running" {
				out = append(out, b.detailsJSON())
			}
		}
		return json.Marshal(out)
	case strings.HasSuffix(u.Path, "/sub-orders"):
		b := s.bots[q.Get("algoId")]
		if b == nil {
			return json.Marshal([]any{})
		}
		typ := q.Get("type")
		var out []any
		for _, o := range b.orders {
			if typ == "live" && o.State == "live" || typ == "filled" && o.State == "filled" || typ == "" {
				out = append(out, map[string]any{
					"ordId": o.OrdID, "instId": b.instID, "state": o.State,
					"px": numStr(o.Px), "sz": numStr(o.Sz), "avgPx": numStr(o.AvgPx),
					"accFillSz": numStr(o.AccSz), "fee": numStr(o.Fee), "feeCcy": o.FeeCcy,
					"pnl": numStr(o.Pnl), "side": o.Side, "cTime": o.CTime,
				})
			}
		}
		return json.Marshal(out)
	}
	return nil, fmt.Errorf("simgrid: 未识路 GET %s", u.Path)
}

// Post 写面（order-algo 单对象/stop-order-algo 数组/withdraw-income 对象）。
func (s *SimGrid) Post(_ context.Context, path string, body []byte) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case strings.HasSuffix(path, "/order-algo") && !strings.HasSuffix(path, "/stop-order-algo"):
		var r struct {
			InstID   string `json:"instId"`
			MaxPx    string `json:"maxPx"`
			MinPx    string `json:"minPx"`
			GridNum  string `json:"gridNum"`
			QuoteSz  string `json:"quoteSz"`
			RunType  string `json:"runType"`
			AlgoClID string `json:"algoClOrdId"`
			Tag      string `json:"tag"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, err
		}
		maxPx, _ := strconv.ParseFloat(r.MaxPx, 64)
		minPx, _ := strconv.ParseFloat(r.MinPx, 64)
		quoteSz, _ := strconv.ParseFloat(r.QuoteSz, 64)
		gridNum, _ := strconv.Atoi(r.GridNum)
		if r.InstID == "" || !(maxPx > minPx && minPx > 0) || gridNum < 2 || quoteSz <= 0 {
			return json.Marshal([]any{map[string]string{"sCode": "51000", "sMsg": "simgrid: 参数非法"}})
		}
		s.seq++
		b := &simBot{algoID: fmt.Sprintf("SIMGRID-%d", s.seq), instID: r.InstID,
			state: "running", maxPx: maxPx, minPx: minPx, gridNum: gridNum,
			quoteSz: quoteSz, runType: r.RunType, clOrdID: r.AlgoClID, tag: r.Tag,
			ct: time.Now()}
		b.runPx = s.px
		if b.runPx <= 0 || b.runPx < minPx || b.runPx > maxPx {
			b.runPx = (minPx + maxPx) / 2
		}
		b.placeInitial()
		s.bots[b.algoID] = b
		return json.Marshal([]any{map[string]string{"algoId": b.algoID, "sCode": "0"}})
	case strings.HasSuffix(path, "/stop-order-algo"):
		var rs []struct {
			AlgoID   string `json:"algoId"`
			InstID   string `json:"instId"`
			StopType string `json:"stopType"`
		}
		if err := json.Unmarshal(body, &rs); err != nil {
			return nil, err
		}
		var out []any
		for _, r := range rs {
			b := s.bots[r.AlgoID]
			if b == nil {
				out = append(out, map[string]string{"algoId": r.AlgoID, "sCode": "51603", "sMsg": "simgrid: 单不在册"})
				continue
			}
			b.state = "stopped"
			for _, o := range b.orders {
				if o.State == "live" {
					o.State = "cancelled"
				}
			}
			out = append(out, map[string]string{"algoId": r.AlgoID, "sCode": "0"})
		}
		return json.Marshal(out)
	case strings.HasSuffix(path, "/withdraw-income"):
		var r struct {
			AlgoID string `json:"algoId"`
		}
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, err
		}
		b := s.bots[r.AlgoID]
		if b == nil {
			return json.Marshal([]any{map[string]string{"sCode": "51603", "sMsg": "simgrid: 单不在册"}})
		}
		p := b.gridProfit
		b.gridProfit = 0
		return json.Marshal([]any{map[string]string{"profit": numStr(p), "sCode": "0"}})
	}
	return nil, fmt.Errorf("simgrid: 未识路 POST %s", path)
}
