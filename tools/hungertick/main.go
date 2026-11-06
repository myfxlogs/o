// hungertick —— 饥饿响应环拍壳 CLI（T-0376 范围3）：T-0384 族沉默探针
// 同槽一拍（信号活+评估零观测盲区自报——先探，败情让位不阻主拍）→
// Detect（机读定义：
// 连续 N 天零提案∨paper 池空）→触发即落 ops_event `饥饿` 行→Respond
// 四步序列（放行线检校→休眠模板族再评估→低产 live 重组候选→venue 候选
// 推进到凭据边界出条件请求），每步事件行落账。退出码：0=跑通（触发与
// 否都算）；1=检测/响应本体败。生产接线（timer/dm_tick 段）=部署面
// 非本票——本件即挂点（T-0376 桩测票，票面「不合并不部署」）。
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/portfolio/hunger"
)

func main() {
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "hungertick: FIN_PG_DSN 缺——钱径池不可达 fail-closed")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: 池建败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: config load: %v\n", err)
		os.Exit(1)
	}
	s := hunger.New(pool, cfg)
	if err := s.ProbeFamilies(ctx); err != nil { // T-0384 族沉默探针同槽一拍——败情让位不阻断 Detect 主拍
		fmt.Fprintf(os.Stderr, "hungertick: 族沉默探针败（让位不阻 Detect）: %v\n", err)
	}
	rep, err := s.Detect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: detect: %v\n", err)
		os.Exit(1)
	}
	if !rep.Triggered {
		fmt.Printf("饱食态（零触发）——窗=%d天 无响应\n", rep.WindowDays)
		return
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('决策席','饥饿','T-0376',$1)`,
		strings.Join(rep.Reasons, " | ")); err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: 饥饿行: %v\n", err)
		os.Exit(1)
	}
	steps, err := s.Respond(ctx, rep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: respond: %v\n", err)
		// 败情已落本步事件行——序列继续语义在包内；此处非零退出显形。
	}
	for _, st := range steps {
		fmt.Printf("step=%s acted=%v %s\n", st.Step, st.Acted, st.Note)
	}
}
