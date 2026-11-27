// registry.go —— fin_canonical_registry 官名登记表读面（O-41378 件1：
// 席位每族登记一次、臂逐 venue 复用；机器只读——写径=迁移种子+运维行，
// 本件零写）。aliases=配对键集（census base/suffix 重组候选对拍）。
package mt5

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoCanonicalRegistry 官名未登记（fail-closed——无别名命中不猜名）。
var ErrNoCanonicalRegistry = errors.New("canonical 未登记 fin_canonical_registry")

// RegistryRow 官名登记行（ct_val=1手 base币量语义锚；quote_ccy=计价币族锚
// 信息位——判据只用 ct_val/dollarPegged 族，不以 quote_ccy 字面等值卡）。
// O-41379：canonical=概念名非落行键——dialect 行 canonical=venue_symbol
// 恒等式（策略 inst_id=venue 原符唯一活命名空间）；registry 只供别名配对
// +参数对拍验证规范，命中族名留 canonical_ref/审计 note 证物位。
type RegistryRow struct {
	Canonical string
	Family    string
	CtVal     float64
	QuoteCcy  string
	Aliases   []string
	Enabled   bool
	Note      string
}

// RegistryStore fin_canonical_registry 读仓（fin 池——verifySymbolMapping
// 与 onboard mint 拍共用单源）。
type RegistryStore struct{ Pool *pgxpool.Pool }

func NewRegistryStore(pool *pgxpool.Pool) *RegistryStore { return &RegistryStore{Pool: pool} }

// Row 单官名登记行（未登记/disabled=ErrNoCanonicalRegistry——调用方按
// 「registry 无锚→走 OKX instruments 参源或 flag」分支）。
func (s *RegistryStore) Row(ctx context.Context, canonical string) (RegistryRow, error) {
	var r RegistryRow
	err := s.Pool.QueryRow(ctx, `SELECT canonical,family,ct_val,quote_ccy,aliases,enabled,note
		FROM fin_canonical_registry WHERE canonical=$1`, canonical).
		Scan(&r.Canonical, &r.Family, &r.CtVal, &r.QuoteCcy, &r.Aliases, &r.Enabled, &r.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return RegistryRow{}, fmt.Errorf("%w: %s", ErrNoCanonicalRegistry, canonical)
	}
	if err != nil {
		return RegistryRow{}, err
	}
	if !r.Enabled {
		return RegistryRow{}, fmt.Errorf("%w: %s（disabled）", ErrNoCanonicalRegistry, canonical)
	}
	return r, nil
}

// Enabled 全量 enabled 登记行（mint 拍别名索引建料——表小全载内存对拍，
// 配对=表上纯函数禁边拉边配）。
func (s *RegistryStore) Enabled(ctx context.Context) ([]RegistryRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT canonical,family,ct_val,quote_ccy,aliases,note
		FROM fin_canonical_registry WHERE enabled ORDER BY canonical`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegistryRow
	for rows.Next() {
		var r RegistryRow
		if err := rows.Scan(&r.Canonical, &r.Family, &r.CtVal, &r.QuoteCcy, &r.Aliases, &r.Note); err != nil {
			return nil, err
		}
		r.Enabled = true
		out = append(out, r)
	}
	return out, rows.Err()
}
