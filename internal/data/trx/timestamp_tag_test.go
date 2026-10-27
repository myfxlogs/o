package trx

import (
	"encoding/json"
	"testing"
)

// T-0192 根修锚：蛇形 block_timestamp 解出真时间（驼峰旧标签恒 0=全量跳过病灶）。
func TestTxJSONTimestampTag(t *testing.T) {
	sample := `{"txID":"abc","block_timestamp":1793099220000,"raw_data":{"contract":[{"type":"TransferContract"}]}}`
	var tx txJSON
	if err := json.Unmarshal([]byte(sample), &tx); err != nil {
		t.Fatal(err)
	}
	if tx.BlockTimestamp != 1793099220000 {
		t.Fatalf("block_timestamp 应解出: %d", tx.BlockTimestamp)
	}
	if tx.TxID != "abc" {
		t.Fatalf("txID 应仍解驼峰: %q", tx.TxID)
	}
}
