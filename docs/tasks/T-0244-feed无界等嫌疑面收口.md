# T-0244 feed 无界等嫌疑面收口——okcli 产品读件显式超时+scan 期界

> 状态：审计通过 | 优先级：高（T-0243 实证残余——22:44→01:43 进程挂死同源嫌疑）
> 审计席：headless @ 2026-10-30
> 立项：决策席 @ 2026-10-29（T-0243 回填钦定『超界另单』件）

## 病灶

o-research 产品枚举臂 2026-10-28 22:44Z→29 01:43Z 进程级挂死 3h——
T-0243 已证迁移窗+挂死实态，嫌疑面=feed HTTP **无显式超时**
（okcli.Get 内网默认）→scan 无界等。挂死机理未钉死（网络半开/对端
RST 不答皆可使无超时 Get 永候），但无界等=唯一可复现的无恢复径。

## 钦定界

- okcli REST 读件显式超时界（context deadline——产品/earn/dcd 枚举
  读件全覆盖，面选 30-60s 量级）；
- scan 单拍期界包络（臂级 deadline 兜底——单拍超界 err 化非永候）;
- 超时触 rej 计数显形（禁静默返零——三律同口径）；
- 不碰判据语义/阈闸值。

## 范围外

不动 armwatch 本体（T-0243 已修）;不迁服务面;不改 feed 接口签名
对外语义。

## 验收锚

- 桩测：feed 端点挂起（不答）→scan 期界内 err 返回+rej 计数+1
  （非永候）；
- 超时值落配置键可调（热启同 research.* 惯例）；
- CI 绿。

## 判决注记（审计席 headless @ 2026-10-30——收口 commit a912b801 逐锚对账）

- 锚① **过**：`TestScanDeadlineEnvelopes`（internal/research/dcd/run_test.go）
  走真路径——app_config 写 research.dcd.scan_timeout_s=1+cfg.Load，feed 桩
  select ctx.Done 让位，ScanOnce 挂 goroutine 断言 4s 内返回（1s 期界下）。
  期界触达后 run.go 确定性执行 ctx.Err()!=nil 支——dcd_scan_timeout+1、
  dcd_feed_failed+1 兼计+wrapped err 返回：err 与 rej 增量在码径内确定
  成立，唯桩测未断言此二项（自白「主锚=不永候」）——断言牙口偏弱，
  回归防护窄，存注不阻断。
- 锚② **过**：四键（research.dcd.scan_timeout_s/funding.scan_timeout_s/
  gapscan.scan_timeout_s/grammar.scan_timeout_s）各臂 scan 顶层按次
  cfg.GetFloat 读取——config.Client 快照由 LISTEN config_changed 重载
  （platform/config/client.go），按次读=严格热启同 research.* 惯例；
  缺/≤0 落 480s 兜底。注：后三键裸域前缀破 research.<域>.* 惯例
  （research.earn.stale_hours/research.grammar.scan_interval_min 为
  同域先例），功能无碍、运维循惯例写键会落空，偏记。
- 锚③ **过**：gh run list --commit a912b801 实证 ci/master/push
  completed=success（7m28s，2026-10-29T08:16:55Z）。
- 钦定界/范围外旁支对账：
  ① okcli.Get 读件本有显式界——client.go:50 Dial 10s+:63 http.Client
     Timeout 15s（blame eaa36ee7，T-0008 2026-10-23 先票存在）+
     NewRequestWithContext 透传包络 ctx——票面「feed HTTP 无显式超时」
     前提被实测修正；真无界径=N 对×get429 重试链无总 deadline，已由
     480s 包络封。「30-60s 量级」件未另落（现存 15s 更紧），偏记。
  ② 超时 rej 专键仅 dcd 臂落（dcd_scan_timeout 注册+触界 Inc）；
     funding 期界内 DB/feed 全 ctx 化=err 返 errc 显形+armwatch 静默
     看门兜底（无静默返零径——cfg 读败降级续走后 pool.Query 触界仍
     err 返）；gapscan→gap_probe_failed、grammar→surface_read_failed/
     写错键通用显形。funding 无超时专键，偏记。
  ③「四臂 DeadlineCtx 共件」实态=dcd/grammar 走 armwatch.DeadlineCtx
     共件，funding/gapscan 为内联同形件——宣称口径偏大、功能等价，
     偏记。
  ④ armwatch.go 仅尾加 DeadlineCtx 件，Watch 本体（Check/SetBeat）
     未动——「不动 armwatch 本体」范围外条不违；import 方向白名单
     两新边（grammar→armwatch、armwatch→platform/config）随件登记。
  ⑤ Feed 签名/判据语义/阈闸值未触；dcd rej 键集仅重排+dcd_scan_timeout
     新增，无语义变更。hypothesis/researcher/silentwatch 无 venue 读件
     不在界。
