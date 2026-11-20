package testutil

// shared_reset.go —— 共享 schema 种子快照与 per-call 定向清表
// （applyReset 两相位：A 全 DELETE 子→父 / B 种子 INSERT 父→子；
// NoTrigger 表 INSERT 包 DISABLE/ENABLE TRIGGER USER——ops_event
// 触发器群重放副作用实证 guard 首迁）。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// snapshotSeed <表> → _testutil_seed_<表>（CREATE TABLE AS 全列快照）+
// 非生成列清单登记（复位 INSERT 用——GENERATED/IDENTITY 列禁显式写）。
func (s *sharedPkg) snapshotSeed(ctx context.Context, pool *pgxpool.Pool, table string) error {
	seed := "_testutil_seed_" + table
	if _, err := pool.Exec(ctx, fmt.Sprintf(
		"DROP TABLE IF EXISTS %s CASCADE", seed)); err != nil {
		return fmt.Errorf("seed drop %s: %w", table, err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf(
		"CREATE TABLE %s AS SELECT * FROM %s", seed, table)); err != nil {
		return fmt.Errorf("seed snap %s: %w", table, err)
	}
	rows, err := pool.Query(ctx, `SELECT column_name FROM information_schema.columns
		WHERE table_schema=$1 AND table_name=$2
		  AND is_generated='NEVER' AND is_identity='NO'
		ORDER BY ordinal_position`, s.schema, table)
	if err != nil {
		return fmt.Errorf("seed cols %s: %w", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return err
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(cols) == 0 {
		return fmt.Errorf("seed cols %s: 空列集（表缺席?）", table)
	}
	s.seedCols[table] = cols
	return nil
}

// teardown 包尾清场——DROP SCHEMA CASCADE（schema GC 日清兜底双保险）。
func (s *sharedPkg) applyReset(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	// A 相：app_config + SeedRestore + Reset 全删（子→父序）。
	if _, err := pool.Exec(ctx, "DELETE FROM app_config"); err != nil {
		t.Fatalf("shared reset app_config del: %v", err)
	}
	for _, tb := range s.seedTbls {
		if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", tb)); err != nil {
			t.Fatalf("shared reset %s del: %v", tb, err)
		}
	}
	for _, tb := range s.reset {
		if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", tb)); err != nil {
			t.Fatalf("shared reset %s: %v", tb, err)
		}
	}
	// B 相：种子还原（逆序=父→子——ops_precedent→ops_event 类种子
	// FK 在此相才有父行可指）。NoTrigger 表 INSERT 包
	// DISABLE/ENABLE TRIGGER USER（enable 必达——Fatal 前先复位，
	// 防半禁态漏给后续测试）。
	ac := strings.Join(s.seedCols["app_config"], ",")
	s.restoreSeed(t, pool, ctx, "app_config", "_testutil_seed_app_config", ac)
	for i := len(s.seedTbls) - 1; i >= 0; i-- {
		tb := s.seedTbls[i]
		s.restoreSeed(t, pool, ctx, tb, "_testutil_seed_"+tb,
			strings.Join(s.seedCols[tb], ","))
	}
}

// restoreSeed 单表种子还原——NoTrigger 表先 DISABLE TRIGGER USER，
// INSERT 成败均 ENABLE 复位后才报错（半禁态=后续测试静默失写面）。
func (s *sharedPkg) restoreSeed(t *testing.T, pool *pgxpool.Pool, ctx context.Context,
	table, seedTable, cols string) {
	t.Helper()
	noTrg := s.noTrigger[table]
	if noTrg {
		if _, err := pool.Exec(ctx, "ALTER TABLE "+table+" DISABLE TRIGGER USER"); err != nil {
			t.Fatalf("shared reset %s disable trg: %v", table, err)
		}
	}
	_, err := pool.Exec(ctx, fmt.Sprintf(
		"INSERT INTO %s (%s) SELECT %s FROM %s", table, cols, cols, seedTable))
	if noTrg {
		if _, e := pool.Exec(ctx, "ALTER TABLE "+table+" ENABLE TRIGGER USER"); e != nil {
			t.Fatalf("shared reset %s enable trg: %v", table, e)
		}
	}
	if err != nil {
		t.Fatalf("shared reset %s restore: %v", table, err)
	}
}

// sharedForDir Pool 共享径判定——caller 目录已注册=共享包。
