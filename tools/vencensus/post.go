package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// postGapsPG 差集按族落 ops_event 记事行（seat='研究'——研究员席消费钦定面）。
// 抑重：同族键 vencensus:<venue>:<family> 在案即跳（重跑不重报）。
func postGapsPG(ctx context.Context, dsn string, gaps map[string]map[string][]string) (int, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer pool.Close()
	n := 0
	for _, v := range []string{"okx", "mt4", "mt5", "binance"} {
		fams := make([]string, 0, len(gaps[v]))
		for f := range gaps[v] {
			fams = append(fams, f)
		}
		sort.Strings(fams)
		for _, f := range fams {
			key := fmt.Sprintf("vencensus:%s:%s", v, f)
			var have bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event WHERE seat='研究' AND event='记事' AND note LIKE '%'||$1||'%')`, key).Scan(&have); err != nil {
				return n, err
			}
			if have {
				continue
			}
			eps := gaps[v][f]
			note := fmt.Sprintf("venue面普查 %s——%s/%s %d 面未登记（canon+码面调用点对拍）；用途：%s；例：%s",
				key, v, f, len(eps), use(v, f), sample(eps))
			if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note) VALUES('研究','记事','venue-census',$1)`, note); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

func sample(eps []string) string {
	s := append([]string(nil), eps...)
	sort.Strings(s)
	if len(s) > 3 {
		s = s[:3]
	}
	return strings.Join(s, " ")
}
