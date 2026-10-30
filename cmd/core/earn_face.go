// earn_face.go —— FEAT-01 Earn 自动停泊开关 RPC（单用途写径：禁泛化 config 写）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "o/internal/gen/dashboardv1"
)

// SetEarnAutoPark 切 app_config earn.sweep_enabled——UPDATE 触发 config_changed
// pg_notify→config.Client 热载下一拍生效；每次切换落 ops_event 审计行
// （actor=cockpit、old→new、来源=RPC）。读回 generation 作并发撞写对拍。
func (c coreService) SetEarnAutoPark(ctx context.Context,
	req *dashboardv1.SetEarnAutoParkRequest) (*dashboardv1.SetEarnAutoParkResponse, error) {
	val := "false"
	if req.Enabled {
		val = "true"
	}
	var old json.RawMessage
	_ = c.pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key='earn.sweep_enabled'`).Scan(&old)
	var gen int64
	var ts time.Time
	err := c.pool.QueryRow(ctx,
		`INSERT INTO app_config(key,value,generation,updated_at,updated_by)
		 VALUES('earn.sweep_enabled',$1::jsonb,1,now(),'cockpit')
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value,
		   generation=app_config.generation+1,updated_at=now(),updated_by='cockpit'
		 RETURNING generation,updated_at`, val).Scan(&gen, &ts)
	if err != nil {
		return nil, fmt.Errorf("app_config 写败: %w", err)
	}
	note := fmt.Sprintf("earn.sweep_enabled %s→%s（actor=cockpit src=RPC gen=%d）", string(old), val, gen)
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('cockpit','earn_autopark_toggle','',$1)`, note); err != nil {
		// 审计行败不阻断已生效的写——但须告警（审计痕是第一性诉求）
		return nil, fmt.Errorf("开关已切但审计行写败: %w", err)
	}
	return &dashboardv1.SetEarnAutoParkResponse{
		Enabled: req.Enabled, Generation: gen, Ts: timestamppb.New(ts)}, nil
}
