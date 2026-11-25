package main

import (
	"context"

	"o/internal/portfolio/hunger"
	"o/internal/trading/okxexec"
)

// censusPubCli 普查公共事实兜底件（O-39480——/public/* 无签名面零凭据
// 可达；okcli 在册时优先复用同一 client）。
var censusPubCli = okxexec.NewClient(okxexec.Credentials{}, false)

// censusFeed oppCensus 腿矩阵供给件适配（okxexec.CensusFacts→hunger
// 中性形——hunger 侧禁 trading/* import 方向闸；get=nil 缺席回落兜底件）。
func censusFeed(get func() *okxexec.Client) func(context.Context) (*hunger.CensusFacts, error) {
	return func(ctx context.Context) (*hunger.CensusFacts, error) {
		cli := get()
		if cli == nil {
			cli = censusPubCli
		}
		f, err := cli.CensusFacts(ctx)
		if err != nil {
			return nil, err
		}
		return &hunger.CensusFacts{SpotLive: f.SpotLive, BorrowRate: f.BorrowRate}, nil
	}
}
