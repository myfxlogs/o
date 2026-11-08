package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/hunger"
	"o/internal/store/pgstore"
)

// stubUniverseGet 宇宙目录桩读件（SWAP 锚族活行+SPOT/FUTURES 族败降级
// ——Symbols 面照供 derivedSpot 兜底径，同产线族级 fail-closed 语义）。
func stubUniverseGet(ctx context.Context, path string) (json.RawMessage, error) {
	switch {
	case strings.Contains(path, "instType=SWAP"):
		return json.RawMessage(`[{"instId":"BTC-USDT-SWAP","state":"live"},{"instId":"ETH-USDT-SWAP","state":"live"}]`), nil
	case strings.Contains(path, "instType=SPOT"):
		return nil, errors.New("stub spot 族败")
	case strings.Contains(path, "instType=FUTURES"):
		return nil, errors.New("stub fut 族败")
	}
	return nil, errors.New("stub 未识路径 " + path)
}

// TestShellUniverseAttach T-0443 先红桩（T-0388 修复类律——病灶存在断言
// 修前红+同断言修后绿）：拍壳 sensor 未挂宇宙扩件+okx.symbols=["*"]
// →ProbeDeps/ProbeLegCoverage fail-closed「宇宙扩件探针未挂」（病灶
// 复刻=生产升件 #2630/#2631 同签名根域——o-core 侧 T-0416 已修，本件
// 钉拍壳侧同根残余挂接面）；newSensor 同装配挂线→同判据复电零未挂错。
func TestShellUniverseAttach(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE app_config SET value='["*"]' WHERE key='okx.symbols'`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 红点（病灶存在断言）：裸 sensor 未挂扩件——修前拍壳形 fail-closed。
	bare := hunger.New(pool, cfg)
	for name, probe := range map[string]func(context.Context) error{
		"ProbeDeps":        bare.ProbeDeps,
		"ProbeLegCoverage": bare.ProbeLegCoverage,
	} {
		if err := probe(ctx); err == nil ||
			!strings.Contains(err.Error(), "宇宙扩件探针未挂") {
			t.Fatalf("%s 未挂扩件应 fail-closed 未挂错: %v", name, err)
		}
	}
	// 绿点：拍壳同装配挂线→双探针复电（修前 newSensor 缺席=编译红）。
	s := newSensor(pool, cfg, stubUniverseGet)
	if err := s.ProbeDeps(ctx); err != nil {
		t.Fatalf("挂线后 ProbeDeps 应复电: %v", err)
	}
	if err := s.ProbeLegCoverage(ctx); err != nil {
		t.Fatalf("挂线后 ProbeLegCoverage 应复电: %v", err)
	}
}
