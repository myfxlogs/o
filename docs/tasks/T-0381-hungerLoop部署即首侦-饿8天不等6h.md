# T-0381 hungerLoop 部署即首侦——饿 8 天不等 6h

- 状态：已批（2026-11-06 业主令——「已经饿了这么久了，为
  什么还要等6h，这不合理」）
- 领取：常径待派（DM 常径机派——钱径件优先）
- 优先级：P0 微件——饥饿环首响延迟修正
- 收口移交归：决策席

## 病灶

T-0378 落地的 hungerLoop 用 `time.NewTicker(6h)`——首拍也
要等 +6h 才响。生产已连续零提案 ~8 天，Detect 触发几成定局，
部署后再等 6h 是 ticker 起表副作用非设计意图。

## 范围

1. `internal/portfolio/lifecycle/hunger.go` hungerLoop：进
   ticker 前**先跑一次 r.hungerTick(ctx)**（部署即首侦），
   之后照旧 6h 复拍——改幅 ~3 行；
2. 桩测断言：拉起即触发一次 Detect（不等 ticker）；
3. 周期常量 hungerBeatEvery 不动（6h 复拍合理面保留）。

## 验收锚

① 桩测：Run 拉起→Detect 立即被调一次（桩注 now 不等 6h）；
② 既有 hungerLoop 测试回归不破；
③ check_all 绿戳对口；镜像面件——**本件到即发窗**（业主令
   钱径优先，不等 0379/0380 攒批——单独一窗或搭现窗）。

## 红线

- 只加首侦，不改 Detect/Respond 语义；
- hunger.go 包本体（internal/portfolio/hunger/）不动。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
