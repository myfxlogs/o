// watch_payload.go —— Watch v2 载荷路由（T-0220a：NOTIFY 一跳推数——按 kind
// 查对应表最新行，protojson→structpb 类型化直转进 WatchEvent.data，前端收帧
// 即渲染零重拉）。复用既有读面方法（零重写查询）；查/转败=调用方帧照发+log
// 显形（信号语义保留不炸流）；无载荷面 kind 返回 nil（纯信号帧）。
package main

import (
	"context"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"o/internal/gen/dashboardv1"
)

func (c coreService) watchPayload(ctx context.Context, kind string) (*structpb.Struct, error) {
	var resp proto.Message
	switch kind {
	case "ev_cashflow":
		r, err := c.ListHedgeFlow(ctx, &dashboardv1.ListHedgeFlowRequest{Limit: 10})
		if err != nil {
			return nil, err
		}
		resp = r
	case "ev_position":
		r, err := c.ListClosedPositions(ctx, &dashboardv1.ListClosedPositionsRequest{Limit: 5})
		if err != nil {
			return nil, err
		}
		resp = r
	case "ev_hostmetrics":
		r, err := c.GetHostMetrics(ctx, &dashboardv1.GetHostMetricsRequest{})
		if err != nil {
			return nil, err
		}
		resp = r
	case "ev_headless":
		r, err := c.GetHeadless(ctx, &dashboardv1.GetHeadlessRequest{})
		if err != nil {
			return nil, err
		}
		resp = r
	case "ev_pipeline":
		r, err := c.GetPipeline(ctx, &dashboardv1.GetPipelineRequest{})
		if err != nil {
			return nil, err
		}
		resp = r
	default:
		return nil, nil // ev_account/ev_strategy/watch_open=纯信号帧
	}
	b, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(resp)
	if err != nil {
		return nil, err
	}
	st := &structpb.Struct{}
	if err := st.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return st, nil
}
