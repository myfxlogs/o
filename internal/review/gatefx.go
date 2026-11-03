// gatefx.go —— T-0314 机械类消费腿（裁定回改点翻译器——裁定面人工触发
// 非事件直写 gate 键；key 不在插行只 UPDATE 在册键；gate.* 键域白名单册）。
package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GateFx 裁定回改点翻译器（fin 池写 app_config——key 不在册=拒写回执）。
type GateFx struct{ fin *pgxpool.Pool }

func NewGateFx(fin *pgxpool.Pool) *GateFx { return &GateFx{fin: fin} }

// Apply 裁定词译键（kill_trigger:<family>→gate.kill.<family>=true；
// sizing:<x>/combo_gate:<x>→gate.sizing.<x>/gate.combo.<x>=value——
// key 不册=拒写不插行；value 参数化零拼接注入防）。
func (g *GateFx) Apply(ctx context.Context, verdictWord, value string) error {
	var key string
	switch {
	case strings.HasPrefix(verdictWord, "kill_trigger:"):
		key = "gate.kill." + strings.TrimPrefix(verdictWord, "kill_trigger:")
	case strings.HasPrefix(verdictWord, "sizing:"):
		key = "gate.sizing." + strings.TrimPrefix(verdictWord, "sizing:")
	case strings.HasPrefix(verdictWord, "combo_gate:"):
		key = "gate.combo." + strings.TrimPrefix(verdictWord, "combo_gate:")
	default:
		return fmt.Errorf("未译词 %q——机械类词集外不触键", verdictWord)
	}
	res, err := g.fin.Exec(ctx, `UPDATE app_config SET value=to_jsonb($2::text)
		WHERE key=$1`, key, value)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("键 %s 不在册（key 不在插行——钦定）", key)
	}
	return nil
}
