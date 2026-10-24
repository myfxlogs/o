// claims.go —— mt5verify 拆件公共面：声明跑批器 U（并发对拍 20s 窗——
// 与单文件版闭包 U 逐行同源，行数棘轮自主拆分）。
package main

import (
	"context"
	"sync"
	"time"
)

var (
	claimsWG sync.WaitGroup
	gctx     context.Context
)

func initClaimRunner(ctx context.Context) { gctx = ctx }

// U 单条声明异步跑批（20s 窗）——结果入 results（rec 内 mu 串行）。
func U(skill, name, expect string, f func(context.Context) (string, string)) {
	claimsWG.Add(1)
	go func() {
		defer claimsWG.Done()
		cctx, cc2 := context.WithTimeout(gctx, 20*time.Second)
		defer cc2()
		st, det := f(cctx)
		rec(claim{skill, name, expect}, st, det)
	}()
}
