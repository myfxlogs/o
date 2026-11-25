package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// brokerCredNS cred_ns 推导（O-37450 roster upsert 同源——roster 行在=
// cred_ns 真源沿用；裸键族在=首户兼容 cred_ns=venue 名〔存量无中缀形〕；
// 否则 venue+'.'+段 新户键前缀——与凭据键写 `<cred_ns>.<field>` 同式自洽）。
func brokerCredNS(ctx context.Context, tx pgx.Tx, venue, proto string) (string, error) {
	var ns string
	err := tx.QueryRow(ctx, `SELECT cred_ns FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&ns)
	if err == nil {
		return ns, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	seg := proto // cfd 类平台子属性=凭据键段（platform=mt5——MT4 方言另件）
	if proto == "cfd" {
		seg = "mt5"
	}
	probe := venue + "." + map[string]string{"mt5": "login", "cex": "api_key"}[seg]
	var bare bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_secrets WHERE key=$1)`, probe).Scan(&bare); err != nil {
		return "", err
	}
	if bare {
		return venue, nil
	}
	return venue + "." + seg, nil
}
