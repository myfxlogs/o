// condreq.go —— fin_condition_gap 请求盒件（T-0177/D-0014：各臂缺凭据/
// 权限/条件时写请求行非孤儿日志——pending 撞=刷新 requested_at+payload
// 非重行；Resolve=人工 UPDATE 口径本票留径）。
package condreq

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Request 条件请求（pending 撞同键=刷新戳+payload——非重行；kind/venue/
// surface 键面钦定闭集）。
func Request(ctx context.Context, pool *pgxpool.Pool, kind, venue, surface, payload string) error {
	_, err := pool.Exec(ctx, `INSERT INTO fin_condition_gap(kind,venue,surface,payload)
		VALUES($1,$2,$3,$4::jsonb)
		ON CONFLICT (venue,surface,kind) WHERE state='pending'
		DO UPDATE SET requested_at=now(),payload=EXCLUDED.payload`,
		kind, venue, surface, payload)
	return err
}

// Resolve 请求结案（state 翻 fulfilled/rejected/dismissed+resolve_note——
// 人工 UPDATE 口径本票留径；pending 才动防重复结案）。
func Resolve(ctx context.Context, pool *pgxpool.Pool, id int64, state, note string) error {
	_, err := pool.Exec(ctx, `UPDATE fin_condition_gap SET state=$2,resolved_at=now(),resolve_note=$3
		WHERE id=$1 AND state='pending'`, id, state, note)
	return err
}
