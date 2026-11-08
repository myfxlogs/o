package testutil

// shard_test.go —— T-0468 族内分库映射桩测（纯函数零 DB——映射确定性/
// 命名域/散列分布断言；建库/连接面归 testdb_shard_test.sh 集成桩）。

import (
	"fmt"
	"os"
	"regexp"
	"testing"
)

func TestShardCountDefault(t *testing.T) {
	os.Unsetenv("O_TEST_SHARDS")
	if got := shardCount(); got != 1 {
		t.Fatalf("缺省 shardCount=%d，want 1（现行单库语义）", got)
	}
	for _, v := range []string{"0", "-3", "abc", ""} {
		t.Setenv("O_TEST_SHARDS", v)
		if got := shardCount(); got != 1 {
			t.Fatalf("O_TEST_SHARDS=%q shardCount=%d，want 1（非法值落单库）", v, got)
		}
	}
}

func TestShardForDirSingle(t *testing.T) {
	t.Setenv("O_TEST_SHARDS", "1")
	for _, d := range []string{"", "/opt/o-wt/t0468/internal/risk", "/x"} {
		if got := shardForDir(d); got != "o_test" {
			t.Fatalf("K=1 dir=%q → %q，want o_test", d, got)
		}
	}
}

func TestShardForDirMapping(t *testing.T) {
	t.Setenv("O_TEST_SHARDS", "8")
	re := regexp.MustCompile(`^o_test_s[1-8]$`)
	// 确定性：同目录两调同库
	d := "/opt/o-wt/t0468/internal/store/pgstore"
	if a, b := shardForDir(d), shardForDir(d); a != b {
		t.Fatalf("同目录非确定性映射 %q vs %q", a, b)
	}
	// 命名域+分布：50 个异目录须全落 s1..s8 且至少 3 个不同库
	// （fnv 确定性——断言为结构性钉死非统计抽样）
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		name := shardForDir(fmt.Sprintf("/opt/o-wt/t0468/internal/pkg%02d", i))
		if !re.MatchString(name) {
			t.Fatalf("dir pkg%02d → %q 出 o_test_s1..8 命名域", i, name)
		}
		seen[name] = true
	}
	if len(seen) < 3 {
		t.Fatalf("50 目录仅散到 %d 库——散列退化", len(seen))
	}
}

func TestShardForDirKBounds(t *testing.T) {
	t.Setenv("O_TEST_SHARDS", "3")
	re := regexp.MustCompile(`^o_test_s[1-3]$`)
	for i := 0; i < 30; i++ {
		if name := shardForDir(fmt.Sprintf("/d/%d", i)); !re.MatchString(name) {
			t.Fatalf("K=3 → %q 越界（mod 边界漏）", name)
		}
	}
}

// ShardDatabase 须由测试文件直调——本文件即 testutil 包目录，
// 断言其返回值命名域合法且 K=1 时恒基库。
func TestShardDatabaseNaming(t *testing.T) {
	os.Unsetenv("O_TEST_SHARDS")
	if got := ShardDatabase(); got != "o_test" {
		t.Fatalf("K 缺省 ShardDatabase=%q，want o_test", got)
	}
	t.Setenv("O_TEST_SHARDS", "8")
	if got := ShardDatabase(); !shardDBRe.MatchString(got) {
		t.Fatalf("K=8 ShardDatabase=%q 出 o_test_s<N> 命名域", got)
	}
}
