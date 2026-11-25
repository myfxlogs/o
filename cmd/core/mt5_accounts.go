package main

// mt5_accounts.go —— O-39440 MT5 逐户实读/NAV 件（venue_readers.go
// 棘轮拆出）：fin_account×fin_venue_roster 解户、cred_gap 升件、
// 逐户活读封装。

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

// mt5AcctRef fin_account×roster 解出的 mt5 户实读参照（O-39440——
// account_id=对拍/簿记域键；tok=roster venue token 会话寻址）。
type mt5AcctRef struct {
	accountID string
	tok       string
}

// mt5RosterAccounts MT5-proto 逐户实读清单（O-39440 通用件——proto
// 驱动非 venue 硬编码：今 Exness(mt5)/XM(xmglobal)，未来 FXPro/ICMarkets
// 等 MT5 经纪商 roster+fin_account 插行即自动入面零码改）：
// fin_account active main 行 ×fin_venue_roster proto='mt5' venue+login
// 双匹配→roster venue token；roster 缺行/disabled/removed=该户不入
// 实读面（cred_gap 由调用侧升件——凭据缺席不猜数）。读败=空集。
func mt5RosterAccounts(ctx context.Context, pool *pgxpool.Pool) []mt5AcctRef {
	rows, err := pool.Query(ctx, `SELECT a.account_id, r.venue
		FROM fin_account a JOIN fin_venue_roster r
		  ON r.proto='mt5' AND r.venue=a.venue AND r.login=a.account_no
		WHERE a.sub_type='main' AND a.state='active'
		  AND r.state NOT IN ('disabled','removed')
		ORDER BY a.account_id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []mt5AcctRef
	for rows.Next() {
		var r mt5AcctRef
		if rows.Scan(&r.accountID, &r.tok) == nil {
			out = append(out, r)
		}
	}
	return out
}

// credGapEsc 账户凭据缺升件（O-39440——spec 钦定：凭据缺席=升件不猜
// 数；esc_class='凭据缺' valence 70 已在 fin_0256 登账；日限 1/
// account/日 NOT EXISTS 原子去重——venue_fail_gate 同构）。
func credGapEsc(ctx context.Context, pool *pgxpool.Pool, accountID, why string) {
	note := fmt.Sprintf("账户实读缺凭据/会话：%s——%s（O-39440：不猜数，凭据供物径归业主 condreq）",
		accountID, why)
	_, _ = pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT 'o-core','升件','', $1,'凭据缺',$2
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE seat='o-core' AND event='升件' AND esc_class='凭据缺' AND esc_scope=$2
			  AND ts >= date_trunc('day', now()))`, note, accountID)
}

// mt5LiveFor MT5 户活读件（OpenSession 逐会话现读——AccountFacts 出 login、
// AccountSummary 出净值；凭据缺席/读败=err 上抛，卡面显「读件缺」不假零。
// T-0101：走单飞缓存——与对拍/风控同一会话窗，防并发会话互踩返零）。
func mt5LiveFor(pool *pgxpool.Pool, cfg *config.Client) func(context.Context) ([]mt5Acct, error) {
	return func(ctx context.Context) ([]mt5Acct, error) {
		// O-39440：逐户活读——fin_account active main×roster token；
		// 单户读败=该户缺席（cred_gap 由对账面升件径覆盖），不拖全表。
		accts := mt5RosterAccounts(ctx, pool)
		if len(accts) == 0 {
			return nil, fmt.Errorf("mt5 账户册空")
		}
		var out []mt5Acct
		var lastErr error
		for _, ra := range accts {
			a, err := mt5SummaryOnce(ctx, pool, cfg, ra.tok, nil)
			if err != nil {
				lastErr = err
				continue
			}
			out = append(out, a)
		}
		if len(out) == 0 && lastErr != nil {
			return nil, lastErr
		}
		return out, nil
	}
}
