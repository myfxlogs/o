# T-0254 dispatcher 双闸补闭——spawn 去重硬锚+Dev 非指令件消费缝（M3 缝二）

> 状态：**在飞（无头）** | 领取：devin-headless-T-0254 @ 2026-10-30 | 优先级：**高（业主令 MANAGE-100 meta 序次件）**
> 执行径：**无头**
> 立项：决策席 @ 2026-10-30（MANAGE-75 回灌两件：双 spawn 实证+Dev report
> 件无消费者）

## 面况

- **双 spawn 实证**（T-0249 案）：一票两无头席同活——调度器『每票至多
  一个 spawn』核心契约未闭合（in_flight=pgrep 有竞态窗/补扫与路由并发），
  MANAGE-75 钦定此锚=调度器带病不过的根修件。
- **Dev 非指令件无消费者**：MANAGE→Dev report 件（如 MANAGE-75 回灌指引）
  落箱即死——无头席按 directive 逐件 spawn，inbox 内 advisory 永不入眼；
  dispatcher 只路由 directive/收口/判决/监理四型。

## 钦定界

- **spawn 去重硬闭**：/opt/meta/mbus-dispatch.sh——路由入口即取
  `.spawned/<tid>` claim 文件（先 touch 后 spawn 原子序）或 flock 段内
  先查后产；boot_sweep 与 route_directive 同走一闸（双径不可各产一）；
  桩测复现：同 directive 二连投+补扫并发→仅一 spawn。
- **Dev 消费缝收口**：spawn 时把 Dev inbox 未 ack 件（非 directive 型）
  文本注入 prompt 文件尾部（`<!-- 在途信件 -->` 段）+spawn 成功后由
  dispatcher 代 ack 落执（消费=入眼即履）——零新监听型，复用既有
  spawn 点；桩测：预置 report 件→prompt 件含其文+件移 acked。
- T-0249 枚举修正实证复核：sweep queue.tsv 193 行已全谱（三状态格式
  尽收——表格式/单行/## 态:），结果段注复核行。

## 验收锚

1. 双闸桩测：directive 重投/补扫并发→spawn=1（日志去重行实证）；
2. Dev 注入桩测：未 ack report 件→prompt 尾部含文+件落 acked；
3. 全谱枚举复核行；4. DRYRUN 全谱绿；5. 票面回填+KB（若坑）。
