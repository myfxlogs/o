package main

// 逐段取证（T-0192 流水零行——复刻 Poll 内部四段，每段打印）。
import (
	"encoding/json"
	"fmt"
	"net/http"
)

type stagedRow struct {
	TxID string
	Kind string
}

func stageProbe(key, addr string) {
	base := "https://api.trongrid.io"
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions?limit=200&order_by=block_timestamp,desc&visible=true", base, addr)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("TRON-PRO-API-KEY", key)
	cli := &http.Client{Timeout: 30 * timeSeconds()}
	resp, err := cli.Do(req)
	if err != nil {
		fmt.Println("段① fetch err:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("段① fetch status:", resp.StatusCode)
	var parsed struct {
		Data []struct {
			TxID           string `json:"txID"`
			BlockTimestamp int64  `json:"blockTimestamp"`
			RawData        struct {
				Contract []struct {
					Type string `json:"type"`
				} `json:"contract"`
			} `json:"raw_data"`
		} `json:"data"`
	}
	body, _ := respBody(resp)
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println("段② unmarshal err:", err)
		return
	}
	fmt.Println("段② unmarshal data len:", len(parsed.Data))
	types := map[string]int{}
	for _, tx := range parsed.Data {
		if len(tx.RawData.Contract) > 0 {
			types[tx.RawData.Contract[0].Type]++
		} else {
			types["<空 raw_data>"]++
		}
	}
	for k, v := range types {
		fmt.Printf("段③ 类型 %s: %d\n", k, v)
	}
	skipped := 0
	for _, tx := range parsed.Data {
		if tx.BlockTimestamp <= 0 {
			skipped++
		}
	}
	fmt.Println("段④ 零时间戳（会被 lastTs 跳过口径吞掉）:", skipped)
}

func respBody(r *http.Response) ([]byte, error) {
	b := make([]byte, 0, 1<<20)
	buf := make([]byte, 65536)
	for {
		n, err := r.Body.Read(buf)
		b = append(b, buf[:n]...)
		if err != nil {
			break
		}
	}
	return b, nil
}
