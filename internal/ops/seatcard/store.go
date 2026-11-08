package seatcard

// Store 的 pgx 实作（T-0458——FIN_PG_DSN 单池：ops_event 推导读+
// ops_seat_card 卡册写。队列深 SQL=scripts/meta/dm_queue.sh 判据同源
// 复刻——改判据两处同票同步）。

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore Store 的 pgx 实作（FIN_PG_DSN 单池——ops_event 与 ops_seat_card
// 同库同角色，fin_0145 授权同族）。
type PGStore struct{ Pool *pgxpool.Pool }

// SeatRows 单查：别名集近窗全行（量小——48h 窗×三席行数极窄）。
func (s *PGStore) SeatRows(ctx context.Context, seats []string, since time.Time) ([]EventRow, error) {
	rows, err := s.Pool.Query(ctx, `SELECT seat,ts,event,ticket,note FROM ops_event
		WHERE seat=ANY($1) AND ts>=$2 ORDER BY ts DESC`, seats, since)
	if err != nil {
		return nil, fmt.Errorf("seatcard rows: %w", err)
	}
	defer rows.Close()
	var out []EventRow
	for rows.Next() {
		var r EventRow
		if err := rows.Scan(&r.Seat, &r.Ts, &r.Event, &r.Ticket, &r.Note); err != nil {
			return nil, fmt.Errorf("seatcard scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QueueDepth dm_queue.sh 判据同源复刻（scripts/meta/dm_queue.sh REVIEW_PRED/
// APPROVED_PRED 逐字——含 T-0380 外派可逆条款；改队列判据须两处同票同步）。
func (s *PGStore) QueueDepth(ctx context.Context) (int, int, error) {
	const q = `WITH l AS (SELECT ticket, max(ts) mt FROM ops_event WHERE ticket<>'' GROUP BY ticket)
SELECT
 count(*) FILTER (WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('收口','移交','收口待审'))
     AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('裁决','完成','驳回'))),
 count(*) FILTER (WHERE EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event='已批')
     AND NOT EXISTS (SELECT 1 FROM ops_event e WHERE e.ticket=l.ticket
        AND e.ts=l.mt AND e.event IN ('派单','领单','拉起','裁决','完成','驳回','挂起'))
     AND NOT EXISTS (SELECT 1 FROM ops_event w WHERE w.ticket=l.ticket
        AND w.event='外派' AND NOT EXISTS
          (SELECT 1 FROM ops_event a WHERE a.ticket=l.ticket
            AND a.event='已批' AND a.ts>w.ts)))
FROM l`
	var review, approved int
	if err := s.Pool.QueryRow(ctx, q).Scan(&review, &approved); err != nil {
		return 0, 0, fmt.Errorf("seatcard queue depth: %w", err)
	}
	return review, approved, nil
}

func (s *PGStore) CardRow(ctx context.Context, seat string) (string, string, string, bool, error) {
	var mid, chat, hash string
	err := s.Pool.QueryRow(ctx, `SELECT message_id,chat_id,last_hash FROM ops_seat_card WHERE seat=$1`, seat).
		Scan(&mid, &chat, &hash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", "", false, nil
		}
		return "", "", "", false, fmt.Errorf("seatcard card row: %w", err)
	}
	return mid, chat, hash, true, nil
}

func (s *PGStore) SaveCard(ctx context.Context, seat, messageID, chatID, hash string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO ops_seat_card(seat,message_id,chat_id,last_hash,updated_at)
		VALUES($1,$2,$3,$4,now())
		ON CONFLICT (seat) DO UPDATE SET message_id=$2,chat_id=$3,last_hash=$4,updated_at=now()`,
		seat, messageID, chatID, hash)
	if err != nil {
		return fmt.Errorf("seatcard save card: %w", err)
	}
	return nil
}

func (s *PGStore) SaveHash(ctx context.Context, seat, hash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE ops_seat_card SET last_hash=$2,updated_at=now() WHERE seat=$1`, seat, hash)
	if err != nil {
		return fmt.Errorf("seatcard save hash: %w", err)
	}
	return nil
}

// RecordErr 卡败落账（seatcard_err——节流由消费面管，此处纯 INSERT；
// 触发器链账 hash 自算，写径角色同 ops_event 册）。
func (s *PGStore) RecordErr(ctx context.Context, seat, note string) error {
	_, err := s.Pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'seatcard_err','',$2)`, seat, note)
	if err != nil {
		return fmt.Errorf("seatcard record err: %w", err)
	}
	return nil
}
