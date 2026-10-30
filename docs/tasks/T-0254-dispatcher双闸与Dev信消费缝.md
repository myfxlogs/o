# T-0254 dispatcher 双闸补闭——spawn 去重硬锚+Dev 非指令件消费缝（M3 缝二）

> 状态：**完成** | 领取：devin-headless-T-0254 @ 2026-10-30 | 优先级：**高（业主令 MANAGE-100 meta 序次件）**
> 执行径：**无头** | 收口commit：0448b4dde4c62c5b15137e8a88c609694bc41525
> （实现件在 /opt/meta 仓支 t/0254-dispatch-gates；本仓仅票面）
> 立项：决策席 @ 2026-10-30（MANAGE-75 回灌两件：双 spawn 实证+Dev report
> 件无消费者）

审计席：headless @ 2026-10-30

终态核销：决策席 @ 2026-10-30——判词「审计通过」在案（审计席落票 commit 71dd674 已并 master）；簿记翻终态=完成。

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

## 结果回填（施工席 @ 2026-10-30，meta 仓支 t/0254-dispatch-gates @0448b4d）

### 实现摘要

- **`mbus-dispatch.sh` 双闸**：
  - `.spawned/<tid>` claim 硬锚——`claim_active` 非破坏查（pid 活∧cmdline
    含 tid=真在飞；空锚未逾 60s=claim 进行中）、`claim_spawn` 入口闸
    （先 touch 后 spawn 原子序；在飞拒；退壳 SPAWN_GRACE=120s 宽限窗内
    死 pid 亦拒，防「快退壳+重投」双产；逾龄陈锚回收保崩殁票自愈）；
  - `route_directive`：in_flight→claim_spawn 双查后 spawn；
    `route_done`/`route_verdict` 返修：flight_wait（≤5s 留 wrapper 尾窗）
    后同查锚据——**三径一闸**，pgrep 竞态窗与补扫并发闭合；
  - spawn_run/spawn_audit 落锚 pid——「每票至多一个 spawn」跨施工/审计
    同锚。
- **Dev 非指令件消费缝**：`collect_dev_mail` 于 spawn_run 内暂存 Dev 箱
  未 ack 非 directive 件→`.spawned/<tid>.mail`(+.mailids 幂等不双录)；
  `HEADLESS_INJECT` 递 wrapper——`devin-headless-run.sh` 在 prompt 尾追加
  `<!-- 在途信件 -->` 段；spawn 成功后 `ack_staged_mail` 代 ack 落执
  （消费=入眼即履；已被他径消费跳；ack 失败留箱待 STALE 兜底）。
- `DISPATCH_LOCKFILE` 可覆写——桩测沙盒锁与生产 dispatcher 锁隔离
  （原硬编码 /tmp 撞现役服务）。
- `test_dispatch.sh` 增 E/F/G 三节桩测。

### 验收锚对账

1. **双闸桩测 PASS**：同票 directive 重投→`gate T-9005：spawn 锚据——
   不重复 spawn（双径同闸）`，spawn 计数=1，被拒件留箱待 STALE（E1–E4）；
   补扫并发实证=dispatcher 重启 boot_sweep 撞同锚→spawn 仍=1、锚据行
   ≥2（G1/G2）；
2. **Dev 注入桩测 PASS**：report 件→`/tmp/headless-T-9006-prompt.md` 尾
   含 `<!-- 在途信件 -->` 段+正文标记；件移 acked+`代 ack Dev 件`日志行
   （F0–F4）；
3. **T-0249 枚举复核 PASS**：sweep_collect.sh 今扫 194 行=旧 queue.tsv
   193 行全数仍在（comm 差集零漏收）+T-0250 复闭新入；三态格式尽收——
   逐票按解析径归族统计：行式 22/表式 109/## 态：节式 35/无态载证 27；
4. **DRYRUN 全谱绿**：`META_SRC=/opt/meta-wt/t0254 test_dispatch.sh
   /tmp/dispatch-e2e-t0254` → PASS=25 FAIL=0「E2E 全绿」；
5. **坑处置：无坑**——钦定缝修未踩新坑；既有双发根因（relay 冻结径
   并行）归 T-0256 wrapper flock 面，两闸互补成链（见偏离）。

### 机检行

`bash -n` mbus-dispatch.sh/devin-headless-run.sh/test_dispatch.sh 三件
语法过；DRYRUN E2E 25/25 绿（exit 0）；meta 仓无 Go/proto 面无 build/
vet 谱。

### 偏离说明

- `devin-headless-run.sh` 属 T-0256 并行钦定面（flock/role.md）——本票
  仅动 prompt 块注入段（HEADLESS_INJECT 五行），与 T-0256 文首 flock 区
  不重叠；T-0256 已先并 meta master（7cc45f4），本支基其顶零撞。
- relay v5.5 冻结径 spawn 不取 claim（钦定界外）——wrapper 层互斥由
  T-0256 flock 兜底：claim 闸=路由层去重，flock=进程层互斥，两闸互补。
- 注入暂存件 `.spawned/<tid>.mail` 跨 spawn 轮次累积——驳回返修轮仍携
  历史在途信段（重复注入无害，入眼即履语义保持）。
- 生效=meta 合并+重启 mbus-dispatch.service（决策席部署径）；桩测沙盒
  全隔离零生产面触碰。
