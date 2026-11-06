# T-0378 hungertick 接线——饥饿感知器挂 lifecycle 周期拍

- 状态：已批（2026-11-06 业主令——挂账两件的 ①；DM 已在
  T-0376 裁决注记「hungertick 接线=部署面后续票」）
- 领取：业主手动外派 @ 2026-11-06（外部工人 run——提示词
  docs/prompts/T-0378-O-Dev施工提示词.md，DM 勿自动领）
- 优先级：P1——T-0376 自治环唯一未接活件
- 收口移交归：决策席（外派件收口回 DM 常径裁——86efad3 新规）

## 病灶

`internal/portfolio/hunger`（T-0376 落地：Detect 两触发形+
Respond 四步响应+DMSet 参数调整行）**零 caller=死件**——
cmd/ 无入口、dm_tick 无调、lifecycle 无挂。饥饿环机制在库
不跑。

## 范围

1. 接线位：`internal/portfolio/lifecycle/run.go` 周期拍挂
   `hunger.New(pool, cfg)`——Detect 先于 Respond，每拍至多
   响应一次（窗口判据内部已带）；
2. 节拍选择照 run.go 现有拍结构（scan 内挂或独立 ticker——
   与现有拍节奏对齐，禁新建 goroutine 野径）；
3. 失败隔离：Detect/Respond 败=log+不拖 lifecycle 主拍
   （T-0369 容错律：败必须自报 ops_event 升件行，非仅 stderr）；
4. 职卡/数据流同步一行（hungertick 入岗）。

## 验收锚

① lifecycle 进程内 hunger.Detect 每拍被调（桩测/日志实证）；
② 触发形构造（零提案窗/papar 池空桩）→ Respond 四步跑通→
   ops_event 饥饿响应+参数调整行落账；
③ 败径桩测：Detect 假败→主拍不炸+自报行落；
④ check_all 绿戳对口收口 SHA；镜像面件——并入下次发窗。

## 红线

- 不改 hunger 包本体语义（接线非重写）；
- 不动业主面键/凭证；
- 不动真单径。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
