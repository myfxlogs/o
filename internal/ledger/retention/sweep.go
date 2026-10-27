// sweep.go —— 数据清道夫（T-0153）：fin_data_retention 账行驱动删窗；
// 未入账表=报缺 fail-closed 拒动；B 类终局行清窗（forward_check judged
// 终局 30d/paper_fill 策略退役 90d）；账链永久表/state 表零删。
package retention

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
)

// Sweeper 清道夫（日拍件——cmd 装配定时臂；AlertFn 注入=测试/告警面）。
type Sweeper struct {
	pool  *pgxpool.Pool
	rej   *rejects.Door
	alert func(ctx context.Context, msg string) // 体积哨兵告警径（log/humanize 件装配）
}

func New(pool *pgxpool.Pool, alert func(ctx context.Context, msg string)) *Sweeper {
	return &Sweeper{pool: pool, alert: alert,
		rej: rejects.New("retention", "write_err", "unregistered_table", "size_alert")}
}

// Sweep 日拍一拍：①未入账表报缺 fail-closed；②audit_keep 账照窗删；
// ③体积哨兵（库>5GB/单表月增>1GB 告警）。
// deadTailSweep 离集陈键自净（T-0197④：状态表行「ts 超龄 AND 键不在
// fin_active_set」→删；谓词读败 fail-safe 不删——防 active_set 读败成
// 全表清空病灶）。
func (s *Sweeper) deadTailSweep(ctx context.Context) error {
	// 谓词读败即整扫 abort（fail-safe 钦定——不拿半读谓词下删刀）
	if _, err := s.pool.Exec(ctx, `SELECT 1 FROM fin_active_set LIMIT 1`); err != nil {
		return nil // 谓词读败=abort 整扫（fail-safe 不删）
	}
	for _, spec := range []struct{ table, tsCol, keyCol string }{
		{"md_ticks", "ts", "inst_id"}, {"md_mark_index", "ts", "inst_id"},
	} {
		tag, err := s.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s < now()-interval '7 days'
			AND %s NOT IN (SELECT inst_id FROM fin_active_set)`, spec.table, spec.tsCol, spec.keyCol))
		if err != nil {
			continue // 逐表隔离不 abort 整扫
		}
		if tag.RowsAffected() > 0 && s.alert != nil {
			s.alert(ctx, fmt.Sprintf("离集陈键清面：%s 删 %d 行", spec.table, tag.RowsAffected()))
		}
	}
	return nil
}

func (s *Sweeper) Sweep(ctx context.Context) error {
	if err := s.deadTailSweep(ctx); err != nil {
		return err
	}
	if err := s.reportUnregistered(ctx); err != nil {
		return err
	}
	if err := s.sweepAuditKeep(ctx); err != nil {
		return err
	}
	return s.sizeSentinel(ctx)
}

// reportUnregistered 库表现存但账册无行=报缺（新表未裁窗——fail-closed 拒动）。
func (s *Sweeper) reportUnregistered(ctx context.Context) error {
	var missing []string
	rows, err := s.pool.Query(ctx, `SELECT t.table_name FROM information_schema.tables t
		WHERE t.table_schema=ANY(current_schemas(false)) AND t.table_type='BASE TABLE'
		AND NOT EXISTS(SELECT 1 FROM fin_data_retention r WHERE r.table_name=t.table_name)
		AND t.table_name NOT LIKE 'timescale%' AND t.table_name NOT LIKE 'pg_%'`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		_ = rows.Scan(&t)
		missing = append(missing, t)
	}
	if len(missing) > 0 {
		s.rej.Inc("unregistered_table")
		log.Printf("ALERT [retention] 未入账表 %d 件——新表须显式裁窗入 fin_data_retention: %v", len(missing), missing)
	}
	return rows.Err()
}

// sweepAuditKeep audit_keep 账照窗删（表/列白名单闭集——禁拼接动态表名，
// 每表谓词钦定）。
func (s *Sweeper) sweepAuditKeep(ctx context.Context) error {
	// 表→删窗谓词（闭集钦定——accountable 账行驱动参数化窗）。
	rules := []struct {
		table, pred string
	}{
		{"fin_forward_check", "judged AND judged_at<now()-($1::interval)"},
		{"fin_paper_fill", `ts<now()-($1::interval) AND EXISTS(
			SELECT 1 FROM fin_strategy s WHERE s.id=strategy_id AND s.state='retired'
			AND s.entered_at<now()-($1::interval))`},
	}
	for _, r := range rules {
		var keep string
		if err := s.pool.QueryRow(ctx,
			`SELECT keep_interval::text FROM fin_data_retention WHERE table_name=$1 AND class='audit_keep'`,
			r.table).Scan(&keep); err != nil || keep == "" {
			continue // 未裁窗=拒动 fail-closed（非误报面）
		}
		tag, err := s.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s", r.table, r.pred), keep)
		if err != nil {
			s.rej.Inc("write_err")
			log.Printf("[retention] %s 清窗败: %v", r.table, err)
			continue // 逐表隔离不 abort
		}
		log.Printf("[retention] %s 清窗删 %d 行（窗=%s）", r.table, tag.RowsAffected(), keep)
	}
	_, err := s.pool.Exec(ctx, `UPDATE fin_data_retention SET last_swept_at=now() WHERE class='audit_keep'`)
	return err
}

// sizeSentinel 体积哨兵：库>5GB 或单表月增>1GB=告警（先于盘满发现无底洞）。
func (s *Sweeper) sizeSentinel(ctx context.Context) error {
	var dbBytes int64
	_ = s.pool.QueryRow(ctx, "SELECT pg_database_size(current_database())").Scan(&dbBytes)
	if dbBytes > 5<<30 {
		s.rej.Inc("size_alert")
		s.fire(ctx, fmt.Sprintf("库体积 %.2fGB 超 5GB 阈——新无底洞查证", float64(dbBytes)/(1<<30)))
	}
	// 单表月增>1GB 哨：现表体−30d 前 last_swept 快照体（粗率换算——
	// last_swept_at 缺/陈旧>30d 的表才评级；账行 size_bytes 现拍回填）。
	rows, err := s.pool.Query(ctx, `SELECT r.table_name,r.size_bytes,r.last_swept_at
		FROM fin_data_retention r WHERE r.class='audit_keep'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var prev int64
			var at *time.Time
			_ = rows.Scan(&t, &prev, &at)
			var cur int64
			_ = s.pool.QueryRow(ctx,
				"SELECT pg_total_relation_size(to_regclass(current_schema()||'.'||$1))", t).Scan(&cur)
			if prev > 0 && at != nil && time.Since(*at) < 35*24*time.Hour &&
				time.Since(*at) > 20*24*time.Hour {
				growth := float64(cur-prev) / (time.Since(*at).Hours() / 24)
				if growth*30 > 1<<30 {
					s.rej.Inc("size_alert")
					s.fire(ctx, fmt.Sprintf("表 %s 月增估算 %.2fGB 超 1GB 阈——体积哨兵", t, growth*30/(1<<30)))
				}
			}
			_, _ = s.pool.Exec(ctx,
				"UPDATE fin_data_retention SET size_bytes=$1 WHERE table_name=$2", cur, t)
		}
	}
	return nil
}

// fire 告警径（装配 nil=log 退路——哨兵必发告警不吞）。
func (s *Sweeper) fire(ctx context.Context, msg string) {
	if s.alert != nil {
		s.alert(ctx, msg)
		return
	}
	log.Printf("ALERT [retention] %s", msg)
}
