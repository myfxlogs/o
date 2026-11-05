// dmconfig —— DM 配置写径 CLI（T-0376 范围2「新小件」）：判词证据→写
// app_config→ops_event 参数调整行——三拍一体（config.DMSet 单事务）。
// 写域闸 fail-closed（分册 scripts/config-write-domain.tsv dm 类外键一律
// 拒——业主面=凭证/资金上限/停机总闸不可达，锚④）。DM 会话（决策席）
// 自调参数唯一径：直写 SQL 绕闸=违例。
//
// 用法：
//
//	FIN_PG_DSN=… dmconfig -key research.funding.min_net_apr -value 0.07 \
//	  -evidence "eval证据:fin_opp_eval#42 net_apr=0.072 连续3评估越阈" -ticket T-0376
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
)

func main() {
	key := flag.String("key", "", "app_config 键（分册 dm 类）")
	value := flag.String("value", "", "新值（JSON 标量——数字/布尔/字符串原生形）")
	evidence := flag.String("evidence", "", "判词证据锚（禁空——无证据不自调）")
	ticket := flag.String("ticket", "", "归属票号（审计行出处必携）")
	flag.Parse()
	if *key == "" || *value == "" || *evidence == "" || *ticket == "" {
		fmt.Fprintln(os.Stderr, "用法: dmconfig -key K -value V -evidence E -ticket T（四参全必携）")
		os.Exit(2)
	}
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺——钱径池不可达 fail-closed")
		os.Exit(2)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dmconfig: 池建败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	res, err := config.New(pool).DMSet(context.Background(),
		*key, json.RawMessage(*value), *evidence, *ticket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dmconfig: %v\n", err)
		os.Exit(1)
	}
	old := "∅"
	if res.Old != nil {
		old = string(res.Old)
	}
	fmt.Printf("域=%s %s 旧=%s 新=%s（ops_event 参数调整行已落——%s）\n",
		config.DomainDM, res.Key, old, string(res.New), *ticket)
}
