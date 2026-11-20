package testutil

// shared.go —— 包级共享 schema 径（工单#26108 per-test setup 税根治；
// 画像实证 ops_event#26104：每测随机 schema+fin 全链 216 迁移重施
// ≈包时长 70-80%，联合谱争用乘数 ~3x）。
//
// 形态：包 TestMain 调一次 ShareMain——建随机 schema+Migrate 各域+
// 种子快照基线+注册 caller dir；此后包内 Pool(t) 走共享径：每调新
// pool（search_path=包 schema——连接级隔离同旧径）+每测首次调用
// 定向清表（app_config 快照还原+SeedRestore 快照还原+Reset DELETE）。
// 未注册包 Pool 走旧径（随机 schema+全链迁移+DROP CASCADE）——
// 兼容壳，迁移渐进不回杀。
//
// 用法（每包一 TestMain）：
//
//	func TestMain(m *testing.M) {
//		testutil.ShareMain(m, testutil.ShareSpec{
//			Domains: []string{"fin"},
//			Migrate: pgstore.Migrate,
//			Reset:   []string{"fin_guard_close", "fin_position", ...}, // 子→父序
//			SeedRestore: []string{"fin_cashflow", "ops_event"},          // 有迁移种子的写表
//			NoTrigger:   []string{"ops_event"},                          // 还原须旁路触发器之表
//		})
//	}
//
// 隔离语义钉死：每次 Pool(t) 调用=一次定向清表（对齐旧径「每调新
// schema」语义——同 t 二次调 Pool 亦得清面；清表在入口非出口，残留
// 不读）。app_config 键册恒复位迁移后基线（覆写/新增键不跨测渗漏——
// ddbar 参数覆写实证面）。schema_migrations 永不入清单——迁移态=
// 共享资产。代价=Reset 清单必须覆盖包写面全表（漏列=跨测渗漏）。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/pgguard"
)

// ShareSpec 共享 schema 声明面。
type ShareSpec struct {
	// Domains 迁移链并集（"fin"/"md"——须覆盖包内测试 Migrate 面）。
	Domains []string
	// Migrate 迁移执行件——调用方传 pgstore.Migrate（testutil 不引
	// pgstore：pgstore 测试件已引 testutil，引即成测试依赖环）。
	Migrate func(ctx context.Context, pool *pgxpool.Pool, domain string) error
	// Reset per-test 定向清表（DELETE FROM——子→父序自排；
	// schema_migrations/种子表禁列——种子表走 SeedRestore）。
	Reset []string
	// SeedRestore 有迁移种子且测试会写的表——快照基线逐测还原
	// （DELETE+INSERT 快照列——种子保、测试行清）。
	SeedRestore []string
	// NoTrigger SeedRestore 中还原 INSERT 须旁路触发器的表——
	// 触发器密集表（ops_event：sem_gate 拒写/act pkey/precedent/
	// chain 改写）INSERT 会重放副作用，DISABLE/ENABLE TRIGGER USER
	// 包复位窗（FK 约束触发器不受 USER 域影响仍生效）。
	NoTrigger []string
	// SeedSQL 建面迁移后、种子快照前执行的种子 SQL——包级测试基线
	// 事实行（如本金曲线 NAV 采点：形态闸等本金读面 fail-closed 判据
	// 需 ambient 净值在册）；快照随 SeedRestore 逐测还原保基线。
	SeedSQL []string
}

// sharedPkg 包级共享 schema 登记项（caller dir → 实体）。
type sharedPkg struct {
	dir       string
	schema    string
	dbname    string
	reset     []string
	seedTbls  []string // "<表>" → 快照表 _testutil_seed_<表>
	seedCols  map[string][]string
	noTrigger map[string]bool
	err       error // ShareMain 建面败——Pool(t) t.Fatal 透传
}

var sharedPkgs sync.Map // callerDir → *sharedPkg

// callerDirShare ShareMain 用 caller 目录——TestMain→ShareMain→
// callerDirShare→callerFileDir(2)：帧0=callerFileDir/1=callerDirShare/
// 2=ShareMain/3=TestMain 所在 _test.go（callerFileDir 内 +1）。
func callerDirShare() string { return callerFileDir(2) }

// ShareMain TestMain 入口——建面+迁移+快照+注册+跑+清场+退出。
// 须由包 TestMain 直接调用（caller 目录=注册键）；DSN 缺席=登记
// dsnOK=false，包内 Pool(t) 逐测 Skip（同旧径语义不炸）。
func ShareMain(m *testing.M, spec ShareSpec) {
	dir := callerDirShare()
	sp := &sharedPkg{
		dir:       dir,
		schema:    sharedSchemaName(dir),
		reset:     spec.Reset,
		seedTbls:  spec.SeedRestore,
		seedCols:  map[string][]string{},
		noTrigger: map[string]bool{},
	}
	for _, tb := range spec.NoTrigger {
		sp.noTrigger[tb] = true
	}
	sharedPkgs.Store(dir, sp)
	code := sp.run(m, spec)
	os.Exit(code)
}

// sharedSchemaName 与旧径同形（t_<pkg尾段>_<rand>）——GC 扫描面同构。
func sharedSchemaName(dir string) string {
	pkg := "x"
	parts := strings.Split(dir, "/")
	if len(parts) >= 2 {
		pkg = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return '_'
		}, parts[len(parts)-1])
	}
	return fmt.Sprintf("t_%s_%d", pkg, time.Now().UnixNano()%1e8)
}

// run 建面全程——败不炸 TestMain（错存 sp.err 由 Pool(t) 首调 t.Fatal 透传，
// 同旧径「建面败=测试败」语义；m.Run 照跑=各测各自 Skip/Fatal）。
func (s *sharedPkg) run(m *testing.M, spec ShareSpec) int {
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" {
		return m.Run() // Pool(t) 共享径见 dsn 空=Skip（旧径同判）
	}
	if err := s.setup(dsn, spec); err != nil {
		s.err = err
		return m.Run()
	}
	code := m.Run()
	s.teardown()
	return code
}

// setup 建 schema+迁移各域+种子快照（admin 池无 search_path 定向 DDL；
// 业务池 AfterConnect 落本 schema——同 testutil.Pool 形）。
func (s *sharedPkg) setup(dsn string, spec ShareSpec) error {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("dsn parse: %w", err)
	}
	// 分库同租约（shardForDir 同径——caller 目录散列）。
	s.dbname = shardForDir(s.dir)
	if s.dbname != "o_test" {
		if ensureShardDB(ctx, cfg.ConnConfig, s.dbname) {
			cfg.ConnConfig.Database = s.dbname
		} else {
			s.dbname = "o_test"
		}
	}
	// admin 池：建/删 schema+快照 DDL（search_path 不涉——全限定名）。
	admin, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("admin pool: %w", err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+s.schema+" CASCADE"); err != nil {
		return fmt.Errorf("schema drop: %w", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+s.schema); err != nil {
		return fmt.Errorf("schema create: %w", err)
	}
	// _gc_born 标（同旧径——GC 在租豁免+日清兜底双保险）
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"CREATE TABLE %s._gc_born(ts timestamptz NOT NULL)", s.schema)); err != nil {
		return fmt.Errorf("gc born: %w", err)
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"INSERT INTO %s._gc_born(ts) VALUES (now())", s.schema)); err != nil {
		return fmt.Errorf("gc born mark: %w", err)
	}
	// 业务池：search_path=包 schema（Migrate/快照全落本 schema）。
	wcfg := cfg.Copy()
	wcfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET search_path TO "+s.schema+",public")
		return err
	}
	pgguard.GuardPool(wcfg)
	work, err := pgxpool.NewWithConfig(ctx, wcfg)
	if err != nil {
		return fmt.Errorf("work pool: %w", err)
	}
	defer work.Close()
	if len(spec.Domains) > 0 && spec.Migrate == nil {
		return fmt.Errorf("ShareSpec.Domains 非空而 Migrate 未供——传 pgstore.Migrate")
	}
	for _, d := range spec.Domains {
		if err := spec.Migrate(ctx, work, d); err != nil {
			return fmt.Errorf("migrate %s: %w", d, err)
		}
	}
	for _, q := range spec.SeedSQL {
		if _, err := work.Exec(ctx, q); err != nil {
			return fmt.Errorf("seed sql: %w", err)
		}
	}
	// app_config 键册快照（无条件——覆写/新增键恒复位迁移后基线）。
	if err := s.snapshotSeed(ctx, work, "app_config"); err != nil {
		return err
	}
	// SeedRestore 表快照+列集（非生成列——INSERT 复位用）。
	for _, tb := range s.seedTbls {
		if err := s.snapshotSeed(ctx, work, tb); err != nil {
			return err
		}
	}
	return nil
}

func (s *sharedPkg) teardown() {
	dsn := os.Getenv("O_TEST_PG_DSN")
	if dsn == "" || s.err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return
	}
	cfg.ConnConfig.Database = s.dbname
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return
	}
	defer pool.Close()
	_, _ = pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+s.schema+" CASCADE")
}

// poolFor Pool(t) 共享径——建 per-call pool（search_path=包 schema）+
// 每测首调定向清表。连接级隔离同旧径（每测新池新连——会话态零渗漏）。
func (s *sharedPkg) poolFor(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	if s.err != nil {
		t.Fatalf("shared schema init: %v", s.err)
	}
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("dsn parse: %v", err)
	}
	cfg.ConnConfig.Database = s.dbname
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET search_path TO "+s.schema+",public")
		return err
	}
	pgguard.GuardPool(cfg)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("test pool: %v", err)
	}
	t.Cleanup(pool.Close)
	s.applyReset(t, pool)
	return pool
}

// applyReset 每次 Pool 调用定向清表——两相位：A 相全 DELETE（种子表
// +Reset 表，注册序=调用方给定子→父序——残行 FK 不阻父删）；B 相全
// INSERT（种子表逆序=父→子——种子 FK 指回父种子行恒在）。app_config
// 键册恒复位迁移后基线（覆写/新增键不跨测渗漏）。跨列 FK 边（Reset
// 表被 SeedRestore 表引用等）=spec 作者自裁——列表内序各自成立。
func sharedForDir(dir string) (*sharedPkg, bool) {
	v, ok := sharedPkgs.Load(dir)
	if !ok {
		return nil, false
	}
	return v.(*sharedPkg), true
}
