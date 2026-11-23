// oi.go —— md_open_interest OI 状态拍臂（工单O-35587②/#35490）：OKX
// /public/open-interest?instType=SWAP 全 SWAP 宇宙快照→md_open_interest
// upsert 最新态（state 表同 md_ticks 律——OI 慢变量低频拍，ctx 臂
// md_ctx_fact 日戳 JSON 桶非关系面，本件=研究/eval 可查正本）。
// 公开零凭据 REST 读件同 ScanGet 径——行情轮询退役令不涉本件：OI 无
// WS 频道（公开面枚举实证），REST 快照=唯一取数道。
package okxws

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// oiPollS OI 快照节拍——慢变量（日内漂移）300s 足供 eval 面新鲜度。
const oiPollS = 300.0

// OIPoller OI 快照采集臂（cmd/collector 常驻件内细胞——同 collector
// 单元登记不新增 resident_units 行）。
type OIPoller struct {
	pool *pgxpool.Pool
	get  ScanGet
}

func NewOIPoller(pool *pgxpool.Pool, get ScanGet) *OIPoller {
	return &OIPoller{pool: pool, get: get}
}

// Run 节拍环（errc 只报终身错——单拍败 WARN 让位下拍，同 ScanPoller 律）。
func (p *OIPoller) Run(ctx context.Context, errc chan<- error) {
	t := time.NewTicker(time.Duration(oiPollS * float64(time.Second))) // sched-exempt-ok：慢变量低频拍
	defer t.Stop()
	p.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.poll(ctx)
		}
	}
}

// poll 单拍：全 SWAP OI 快照→逐 inst upsert（批内单事务——半途败
// 下拍幂等覆写无害）。
func (p *OIPoller) poll(ctx context.Context) {
	raw, err := p.get(ctx, "/api/v5/public/open-interest?instType=SWAP")
	if err != nil {
		log.Printf("WARN [okxoi] OI 快照读败——本拍让位: %v", err)
		return
	}
	var resp struct {
		Data []struct {
			InstID string `json:"instId"`
			Oi     string `json:"oi"`
			OiCcy  string `json:"oiCcy"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		log.Printf("WARN [okxoi] OI 快照解析败——本拍让位: %v", err)
		return
	}
	n := 0
	for _, r := range resp.Data {
		if r.InstID == "" {
			continue
		}
		oi, err1 := strconv.ParseFloat(r.Oi, 64)
		oiccy, err2 := strconv.ParseFloat(r.OiCcy, 64)
		if err1 != nil || err2 != nil {
			continue // 缺席≠真值0——不可解析行跳过不落
		}
		if _, err := p.pool.Exec(ctx, `INSERT INTO md_open_interest(ts,inst_id,oi,oi_ccy)
			VALUES(now(),$1,$2,$3)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
			oi=EXCLUDED.oi,oi_ccy=EXCLUDED.oi_ccy,recv_ts=now()`,
			r.InstID, oi, oiccy); err != nil {
			log.Printf("WARN [okxoi] OI 落行败 %s: %v", r.InstID, err)
			continue
		}
		n++
	}
	log.Printf("[okxoi] OI 拍：%d/%d inst 落行", n, len(resp.Data))
}
