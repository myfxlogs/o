package trx

import (
	"encoding/json"
	"testing"
)

// T-0108 桩测锚：TronGrid 响应解析——accountJSON/txJSON 方言字段名对拍
// （txID 驼峰、block_timestamp 蛇形——T-0192 SG 200 笔分段探针实拍：同结构大小写混杂）。
func TestTronGridDialectParse(t *testing.T) {
	var a accountJSON
	if err := json.Unmarshal([]byte(`{"address":"TX","balance":1000000,"allowance":500,
		"votes":[{"vote_address":"SR1","vote_count":100}],
		"frozenV2":[{"amount":2000000},{"type":"ENERGY"}],
		"account_resource":{"delegated_frozenV2_balance_for_energy":3000000}}`), &a); err != nil {
		t.Fatal(err)
	}
	if a.Balance != 1000000 || len(a.Votes) != 1 || a.FrozenV2[0].Amount == nil ||
		a.FrozenV2[1].Type == nil || a.AccountResource.DelegatedFrozenV2BalanceForEnergy != 3000000 {
		t.Fatalf("account 解析: %+v", a)
	}
	var txr txsResponseJSON
	if err := json.Unmarshal([]byte(`{"data":[{"block_timestamp":1700000000000,"txID":"abc123",
		"raw_data":{"contract":[{"type":"TransferContract"}]},"ret":[{"contractRet":"SUCCESS"}]}]}`), &txr); err != nil {
		t.Fatal(err)
	}
	if len(txr.Data) != 1 || txr.Data[0].TxID != "abc123" || txr.Data[0].BlockTimestamp != 1700000000000 {
		t.Fatalf("tx 解析: %+v", txr)
	}
}
