// write.go —— app_config 写径（T-0178：updated_by 写者身份留痕——live_switch
// 翻位者无审计痕病灶根修；写=generation 序列热启帧同源（Watch 件现径零改））。
package config

import (
	"context"
	"encoding/json"
)

// Set 写值+写者名（actor=进程名/席名——app_config 钦定审计列留痕；
// updated_at now() 同拍记——翻位者可考；value=JSON 原生型标量契约守）。
func (c *Client) Set(ctx context.Context, key string, value json.RawMessage, actor string) error {
	_, err := c.pool.Exec(ctx, `INSERT INTO app_config(key,value,updated_at,updated_by)
		VALUES($1,$2::jsonb,now(),$3)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,updated_at=now(),updated_by=EXCLUDED.updated_by`,
		key, string(value), actor)
	return err
}
