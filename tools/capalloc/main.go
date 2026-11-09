// capalloc —— 资金闸值窄门（业主令 2026-11-09「资金分配决策权整体下放
// RSI/system」的执行手）：唤知会话调本件写闸值白名单三键，码面钦定
// 边界+同事务落 ops_event 参数调整行（自主≠无账）。
// 用法：FIN_PG_DSN=... capalloc set <key> <value> --evidence "<证据锚>"
// 白名单：risk.max_exposure_usd / risk.max_notional_usd /
//
//	invest.paper.notional_usd（其余键=越界硬拒）
//
// 边界（业主令包络内自决；净值=md_equity_curve ccy=NAV 末帧）：
//
//	exposure  ∈ [100, 0.50×equity]      （总敞口永远 ≤ 净值半）
//	notional  ∈ [10,  min(0.10×eq,exp)] （单笔永远 ≤ 净值 1/10 且 ≤敞口）
//	paper_not ∈ [10,  min(0.05×eq,not)] （纸面永远 ≤ 净值 1/20 且 ≤单笔）
//
// 净值读不出=拒（fail-closed——禁无锚调闸）；--evidence 缺=拒（无账=违例）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/store/pgstore"
)

// keyRule 白名单键规则（lo/hi 由净值动态算——比例界在码面钦定）。
var keyRules = map[string]struct {
	hiPct float64 // 占净值比例上界
	loAbs float64 // 绝对下界
}{
	"risk.max_exposure_usd":     {hiPct: 0.50, loAbs: 100},
	"risk.max_notional_usd":     {hiPct: 0.10, loAbs: 10},
	"invest.paper.notional_usd": {hiPct: 0.05, loAbs: 10},
}

// allocStore 可测缝——run 依赖的最小库面。
type allocStore interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgxRow
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
}

// pgxRow/poolStore——pgxpool 往 allocStore 缝的适配（可测面窄化）。
type pgxRow interface{ Scan(dest ...any) error }

type poolStore struct{ p *pgxpool.Pool }

func (s poolStore) QueryRow(ctx context.Context, sql string, args ...any) pgxRow {
	return s.p.QueryRow(ctx, sql, args...)
}

func (s poolStore) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := s.p.Exec(ctx, sql, args...)
	return tag.RowsAffected(), err
}

// run 判界→写键→落账（同连接串行——闸值写低频,事务面窄化无妨）。
func run(ctx context.Context, s allocStore, key string, val float64, evidence string) error {
	r, ok := keyRules[key]
	if !ok {
		return fmt.Errorf("册外键硬拒 %s——白名单三件外永不触", key)
	}
	if evidence == "" || val <= 0 {
		return errors.New("证据锚缺/值非正=拒")
	}
	var eq float64
	if err := s.QueryRow(ctx, `SELECT equity FROM md_equity_curve
		WHERE ccy='NAV' ORDER BY ts DESC LIMIT 1`).Scan(&eq); err != nil || eq <= 0 {
		return fmt.Errorf("净值读件缺=拒（fail-closed 禁无锚调闸）: %w", err)
	}
	cur := map[string]float64{}
	for k := range keyRules {
		var v float64
		if err := s.QueryRow(ctx, `SELECT coalesce((value#>>'{}')::float8,0)
			FROM app_config WHERE key=$1`, k).Scan(&v); err == nil {
			cur[k] = v
		}
	}
	cur[key] = val
	if cur["risk.max_notional_usd"] > cur["risk.max_exposure_usd"] {
		return fmt.Errorf("序闸拒: notional %.0f > exposure %.0f",
			cur["risk.max_notional_usd"], cur["risk.max_exposure_usd"])
	}
	if cur["invest.paper.notional_usd"] > cur["risk.max_notional_usd"] {
		return fmt.Errorf("序闸拒: paper %.0f > notional %.0f",
			cur["invest.paper.notional_usd"], cur["risk.max_notional_usd"])
	}
	hi := r.hiPct * eq
	if val < r.loAbs || val > hi {
		return fmt.Errorf("界外拒: %s=%.0f 出包络 [%.0f, %.0f]（净值=%.0f 界=码面钦定非配置）",
			key, val, r.loAbs, hi, eq)
	}
	var old string
	_ = s.QueryRow(ctx, `SELECT value#>>'{}' FROM app_config WHERE key=$1`, key).Scan(&old)
	if _, err := s.Exec(ctx, `UPDATE app_config SET value=$2, updated_by='capalloc'
		WHERE key=$1`, key, strconv.FormatFloat(val, 'f', -1, 64)); err != nil {
		return fmt.Errorf("写键败: %w", err)
	}
	if _, err := s.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('处置席','参数调整','',$1)`,
		fmt.Sprintf("%s %s→%.0f（净值=%.0f 包络上界=%.0f）证据:%s",
			key, old, val, eq, hi, evidence)); err != nil {
		return fmt.Errorf("落账败: %w", err)
	}
	return nil
}

func main() {
	// 手工取参——flag.Parse 遇首个位置参即停（set <k> <v> --evidence
	// 自然序会漏读旗标）,窄门参数面小,扫串自取。
	var evidence string
	var pos []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		if a == "--evidence" && i+1 < len(os.Args) {
			i++
			evidence = os.Args[i]
		} else if v, ok := strings.CutPrefix(a, "--evidence="); ok {
			evidence = v
		} else {
			pos = append(pos, a)
		}
	}
	if len(pos) != 3 || pos[0] != "set" {
		fmt.Fprintf(os.Stderr, "用法: capalloc set <key> <value> --evidence \"<锚>\"\n白名单: risk.max_exposure_usd|risk.max_notional_usd|invest.paper.notional_usd\n")
		os.Exit(2)
	}
	key, raw := pos[1], pos[2]
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil || val <= 0 {
		log.Fatalf("值形败 %q", raw)
	}
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		log.Fatal("FIN_PG_DSN 缺——窄门无账面 fail-closed")
	}
	ctx := context.Background()
	pool, err := pgstore.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("连库败: %v", err)
	}
	defer pool.Close()
	if err := run(ctx, poolStore{pool}, key, val, evidence); err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("[capalloc] %s→%.0f", key, val)
}
