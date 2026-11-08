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

	"o/internal/data/okxws"
	"o/internal/platform/config"
	"o/internal/platform/opschain"
	"o/internal/portfolio/hunger"
	"o/internal/trading/okxexec"
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
	s := newSensor(pool, cfg, okxexec.NewClient(okxexec.Credentials{}, false).PublicGet) // T-0443 宇宙扩件挂线——公开面零凭据（collector 同例）
	// T-0386 验链拍同槽——全链重算纯读；断链/伪戳→账链断+升件自报；败情让位不阻 Detect
	cv := opschain.New(pool)
	if rep, err := cv.Verify(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "hungertick: 验链拍败（让位不阻 Detect）: %v\n", err)
	} else if !rep.OK() {
		fmt.Fprintf(os.Stderr, "hungertick: 账链断 检出=%d 首处=%s@#%d——自报双行\n",
			len(rep.Findings), rep.Findings[0].Kind, rep.Findings[0].ID)
		if err := cv.ReportBreak(ctx, rep); err != nil {
			fmt.Fprintf(os.Stderr, "hungertick: 账链断自报落败: %v\n", err)
		}
	}
	if err := s.ProbeFamilies(ctx); err != nil { // T-0384 族沉默探针同槽一拍——败情让位不阻断 Detect 主拍
		fmt.Fprintf(os.Stderr, "hungertick: 族沉默探针败（让位不阻 Detect）: %v\n", err)
	}
	if err := s.ProbeLegCoverage(ctx); err != nil { // T-0392 腿覆盖探针同槽一拍——hungerTick 同位补位（T-0443 拍壳漂移根修）
		fmt.Fprintf(os.Stderr, "hungertick: 腿覆盖探针败（让位不阻 Detect）: %v\n", err)
	}
	if err := s.ProbeDeps(ctx); err != nil { // T-0394 依赖对枚举探针同槽一拍——hungerTick 同位补位（T-0443 拍壳漂移根修）
		fmt.Fprintf(os.Stderr, "hungertick: 依赖对探针败（让位不阻 Detect）: %v\n", err)
	}
	if err := s.ProbeFeedStale(ctx); err != nil { // T-0410 feed 死闸同槽一拍——败情让位不阻断 Detect 主拍
		fmt.Fprintf(os.Stderr, "hungertick: feed死闸探针败（让位不阻 Detect）: %v\n", err)
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

// newSensor 拍壳 sensor 装配（T-0443——宇宙扩件挂线同 cmd/core 供给件
// 语义：okxws.UniverseCatalog 单源实读+TTL 缓存，探针宽谱判据不滤
// 黑名单；okx.symbols="*" 下未挂=探针 fail-closed「宇宙扩件探针未挂」
// ——生产升件 #2630/#2631 同签名病灶本体，拍壳侧同根残余挂接面）。
func newSensor(pool *pgxpool.Pool, cfg *config.Client, get okxws.ScanGet) *hunger.Sensor {
	uni := okxws.NewUniverseCatalog(get, okxws.UniverseTTL)
	return hunger.New(pool, cfg).AttachUniverse(uni.Symbols)
}
