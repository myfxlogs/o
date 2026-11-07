# T-0434 hostmetrics 监控自监——写器死活根修 + 前端陈旧硬闸

- 状态：施工中
- 领取：claude-t0434 @ 2026-11-07（外派施工 O-Dev 径，worktree
  /opt/o-wt/t0434 支 t/0434-hostmetrics-selfprobe——基于
  origin/master 落 7461c873 后并入本地 master 9675dab1 使票面
  commit 成祖，偏离说明：origin/master 未含立票 commit，不并
  无法翻转票面，已核实 9675dab1 为 docs-only）
- 优先级：P1——system 页渲染 8 天前尸体数据（fin_host_metrics
  'sg' 行冻结 2026-10-30 04:18，df 实拍 97% vs 页面假数）；
  观测面撒谎=运维决策基于尸体
- 收口移交归：外审径（业主令 2026-11-07「外派外审外部署，
  不去给DM做积压」成例——外派件闭环=外派施工→外审席裁→外部署）

## 病灶（2026-11-07 实证三查）

- `fin_host_metrics` 唯一行 host='sg' ts=2026-10-30 04:18，
  disk_total≈450G/disk_used≈252G；同期 `df` 实拍 used≈411G——
  前端展示的「磁盘余量」是 8 天前的尸体；
- 写器=`o-core` 内 `hostmetrics.Run(ctx,pool,"sg","/",30s)`
  （cmd/core/main.go:132）——o-core RPC 仍正常应答，
  **只有自报 goroutine 死了**；
- 死因设计：`Run` 里 `pool==nil→return` 一次永退（无重试）、
  `Upsert` 失败仅 log（「观测面缺不拖业务面」钦定静默）——
  **盯人的探头没人盯**，静默死第五条实证；
- 前端：API 已回 `Stale` 标（ts 龄>2min），但页面照常渲染
  旧数值——标在尸体仍展出。

## 面（钦定，三件一体）

① **写器死活根修**：先实锤 10-30 死因（o-core 日志/启动序
   pool 就绪时序），按根因修——禁止只加 log 糊弄：
   - pool 未就绪不永退：带退避的重试环直到首拍成功；
   - Upsert 连败阈值→ops_event 升件（写不进库=观测断供
     是一等事件，非 log 噪音）；
   - 断言红桩：模拟 pool nil 启动→恢复后自报环必活；
② **监控自监探针**：`fin_host_metrics.ts` 陈旧→ops_event
   （复用 feedstale/ctx-probe 同构形——第三实例即成族，
   抽公共「自报表陈旧→升件」模子勿第三份拷贝）；
③ **前端陈旧硬闸**：Stale 行禁正常渲染——整块置灰+红标
   「数据陈旧·非实时」，数值可留但视觉层级必须让人一眼
   识破是尸体；无行=显式「无数据」态不空跑。
   API 侧 Stale 语义不动（已正确），改的是渲染契约。

## 验收

- 10-30 死因实锤结论回填票面（不许「原因不明就修了」）；
- 桩测：pool nil→ready 恢复自报环活；Upsert 注入连败→
  ops_event 行落；fin_host_metrics.ts 手工回拨>阈→升件；
- 前端：stale 行渲染断言（灰+红标存在）+fresh 行零回归；
- 绿戳对口+全链；坑处置随 commit（静默死族新实证若
  够格→坑条目）。

## 备注

「自报件陈旧→升件」三实例成族后，建议顺手在票面留一句
出生律候选：新自报表若无陈旧探针=交付不齐——本票不强制
立律，留给外审席判。
