// universe.go —— 宇宙扩件装配适配（T-0416 归一：目录实读/TTL 缓存归
// okxws.UniverseCatalog 单一供给件——cmd/collector 复制件同归，禁第二套
// 归一链）。本件只剩两适配：懒客取件（凭据缺席期公开面无凭据 PublicGet
// 直读降级——T-0122 原语义）+venue 黑名单滤（T-0142——venue_untradeable
// venue 拒过的 inst 不进订阅面）。
package main

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/okxws"
	"o/internal/trading/okxexec"
)

// universeWire 宇宙扩件接线（T-0416——供给件单源+订阅扩件闭包同出：
// uni 供 hunger 探针/他消费面同读，expand 供 collector.ExpandSymbols
// 内嵌 fin_venue_untradeable 黑名单滤——T-0142 语义随件）。
func universeWire(getCli func() *okxexec.Client, pool *pgxpool.Pool) (*okxws.UniverseCatalog, func(ctx context.Context) ([]string, error)) {
	uni := okxws.NewUniverseCatalog(universeGet(getCli), okxws.UniverseTTL)
	return uni, func(ctx context.Context) ([]string, error) {
		syms, err := uni.Symbols(ctx)
		if err != nil {
			return nil, err
		}
		return dropUntradeable(ctx, pool, syms), nil
	}
}

// universeGet 懒客 REST 公开读件（okcli 凭据件后至——每拍自取；nil=无凭据
// 公开面直读降级）。
func universeGet(getCli func() *okxexec.Client) okxws.ScanGet {
	return func(ctx context.Context, pathWithQuery string) (json.RawMessage, error) {
		cli := getCli()
		if cli == nil {
			cli = okxexec.NewClient(okxexec.Credentials{}, false)
		}
		return cli.PublicGet(ctx, pathWithQuery)
	}
}

// dropUntradeable venue 黑名单滤（T-0142——venue_untradeable 行=venue 拒过
// 的 inst 不进订阅/评估；表读败=原集放行不连坐——滤是增益面非阻断面）。
func dropUntradeable(ctx context.Context, pool *pgxpool.Pool, syms []string) []string {
	bl, err := pool.Query(ctx, `SELECT inst_id FROM fin_venue_untradeable`)
	if err != nil {
		return syms
	}
	defer bl.Close()
	banned := map[string]bool{}
	for bl.Next() {
		var b string
		if bl.Scan(&b) == nil {
			banned[b] = true
		}
	}
	var kept []string
	for _, sy := range syms {
		if !banned[sy] {
			kept = append(kept, sy)
		}
	}
	return kept
}
