// Package testutil —— 测试基建（独立测试实例 D-021；缺 DSN=skip 语义）。
// T-0021 隔离硬化：每测试包独占 schema（test_<pkg>_<rand>+search_path）——
// 共享库并发下 app_config 键/快照行/策略行互不干扰（guard 翻 kill_switch
// 不再误伤 risk 包，他包 fixture 行不顶阈值）。Migrate 非限定名随 schema 落位。
package testutil

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/pgguard"
)

// T-0468 族内分库：O_TEST_SHARDS>1 时按「调用方包目录」确定性散列分库
// （o_test_s1..sK）——pg_advisory_lock 锁域按库隔离（pg_locks 锁标含
// databaseOid），跨库并发 Migrate 不再互抢 860223 全库锁=herd 消。
// 语义钉死：包间零共享态由包级 schema 隔离结构性保证（每测试独 schema），
// 任意确定性划分不拆语义耦；cluster 级对象（o_fin_ro 角色等）在租约实例
// 上已存在/幂等，跨库同库名对象互不干扰。K<=1 或分库确保败=落基库
// o_test=现行语义（fail-open 退化不语义错）。
const shardDBPrefix = "o_test_s"

var shardDBRe = regexp.MustCompile(`^o_test_s[0-9]+$`)

// shardCount 读 O_TEST_SHARDS（非法/缺省=1=现行单库语义）。
func shardCount() int {
	k, err := strconv.Atoi(strings.TrimSpace(os.Getenv("O_TEST_SHARDS")))
	if err != nil || k < 1 {
		return 1
	}
	return k
}

// callerFileDir 取 skip 层外调用方文件所在目录（=包目录）。
// 公共面（Pool/ShardDatabase）各自以 skip=1 直取测试文件——helper
// 不内嵌取帧（深度漂移坑：多一层包装取帧即错位）。
func callerFileDir(skip int) string {
	_, file, _, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	return path.Dir(file)
}

// shardForDir 包目录→库名：fnv32a 散列 mod K（K=1 恒基库 o_test；
// K>1 → o_test_s<1..K>）。同目录必同库（确定性），异目录散列分布。
func shardForDir(dir string) string {
	k := shardCount()
	if k <= 1 || dir == "" {
		return "o_test"
	}
	h := fnv.New32a()
	h.Write([]byte(dir))
	return fmt.Sprintf("%s%d", shardDBPrefix, int(h.Sum32()%uint32(k))+1)
}

// ShardDatabase 返回「本调用方包」测试解析到的库名（o_test 或
// o_test_s<N>）。裸 DSN 消费面（第二池/异用户连）与 testutil.Pool 配对
// 时必须把 cfg.ConnConfig.Database 置为本值——否则池落基库而 schema
// 在分库=断言必败（writefail/pipeline_face T-0468 根修点）。
// 契约：须由测试文件直接调用（Caller 取帧=测试文件目录）；须先有
// testutil.Pool 或等价确保（分库不存在时裸连败——正常序 fixture 先行）。
func ShardDatabase() string {
	return shardForDir(callerFileDir(1))
}

var (
	shardEnsureMu sync.Mutex
	shardEnsured  = map[string]bool{}
)

// ensureShardDB 确保分库存在（每测试二进制至多一次/库——成败同记，
// 败不重试风暴；跨包并发 CREATE 撞 duplicate_database 时复查存在性
// 放行）。败=返回 false，调用方落基库（语义不变只慢）。
func ensureShardDB(ctx context.Context, base *pgx.ConnConfig, dbname string) bool {
	if !shardDBRe.MatchString(dbname) {
		return dbname == "o_test"
	}
	shardEnsureMu.Lock()
	defer shardEnsureMu.Unlock()
	if ok, seen := shardEnsured[dbname]; seen {
		return ok
	}
	ok := false
	defer func() { shardEnsured[dbname] = ok }()
	cfg := base.Copy()
	cfg.Database = "postgres" // 维护库——CREATE DATABASE 须在库外交互
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return false
	}
	defer conn.Close(ctx)
	var exists bool
	if err := conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", dbname,
	).Scan(&exists); err != nil {
		return false
	}
	if !exists {
		// 库名由 shardForDir 内部生成（o_test_s+数字）——注入面零外部输入
		if _, err := conn.Exec(ctx, "CREATE DATABASE "+dbname); err != nil {
			// 并发首建撞车：复查存在性——在则视为成
			if err2 := conn.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", dbname,
			).Scan(&exists); err2 != nil || !exists {
				return false
			}
		}
	}
	ok = true
	return true
}

// Pool 接 O_TEST_PG_DSN 独立测试实例+包级 schema 隔离；
// 未设 DSN=Skip（CI 5434/本机覆写端口同一口径）。
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("O_TEST_PG_DSN 未设——独立测试实例缺席，skip（不打生产实例）")
	}
	// 包级 schema 名（caller 包尾段+随机尾——同包多测共享 schema 无碍，包间零干扰）
	pkg := "x"
	if _, file, _, ok := runtime.Caller(1); ok {
		parts := strings.Split(file, "/")
		if len(parts) >= 2 {
			pkg = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
					return r
				}
				return '_'
			}, parts[len(parts)-2])
		}
	}
	schema := fmt.Sprintf("t_%s_%d", pkg, time.Now().UnixNano()%1e8)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("dsn parse: %v", err)
	}
	// T-0468 分库：包目录散列→o_test_s<N>（K=1 恒 o_test 现行语义）；
	// 分库确保败落基库=只慢不语义错。Database 置位先于 GuardPool/
	// NewWithConfig——ConnConfig.Database 即连接目标库。
	dbname := shardForDir(callerFileDir(1))
	if dbname != "o_test" {
		ctx0 := context.Background()
		if ensureShardDB(ctx0, cfg.ConnConfig, dbname) {
			cfg.ConnConfig.Database = dbname
		} else {
			t.Logf("testutil: 分库 %s 确保败——落基库 o_test（语义不变只慢）", dbname)
		}
	}
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET search_path TO "+schema+",public")
		return err
	}
	pgguard.GuardPool(cfg) // T-0154 事务残留守卫（测试面同守——桩锚同池验）
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("test pool: %v", err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS "+schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	// T-0435：GC 出生标——gc_test_schemas.sh 扫②按 _gc_born.ts 判孤儿
	// （标新鲜 <24h=在租豁免；无标历史件走 T-0186 原语义日清）。
	if _, err := pool.Exec(ctx, "CREATE TABLE IF NOT EXISTS "+schema+"._gc_born(ts timestamptz NOT NULL)"); err != nil {
		t.Fatalf("gc born mark: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO "+schema+"._gc_born(ts) VALUES (now())"); err != nil {
		t.Fatalf("gc born mark: %v", err)
	}
	t.Cleanup(func() { // T-0186：schema GC——用后清（实例窒息根治）
		ctx2, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx2, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		pool.Close()
	})
	return pool
}
