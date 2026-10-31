---
schema_version: 2
id: meta-event-plane-external-blind-spots
kind: pitfall
risk: high
status: mitigated
keywords: [meta, 事件驱动, 闭环, 外部票, 审计, 部署, webhook, 断点, dispatcher]
scope: [meta, dispatcher, deploy, ci, automation]
summary: meta 事件面只覆盖内部件流——外部系统态（GitHub CI）与外部票生命周期
  （收口→审计/驳回→返修）皆无消费者，每段完工静默等人肉接；实证门4收口悬空9h。
  修=门4/门5/GHEVT/EXTAUDIT/REWORK/DEPLOYNF 六件补环，端到端无人值守部署实跑通。
---

# meta 事件面外部盲区——闭环断点谱系

## 病灶

「事事有应答」尺量：meta 自动化的是「席的唤起」而非「交付流水线」。三类断点：

1. **外部票生命周期断**：dispatcher 只扫 o 仓 `docs/tasks/`，meta 票面
   `收口待审`/`驳回` 态无人路由——实证门4 收口后审计悬空 9h、DOOR5 驳回
   返修靠人肉接续。
2. **外部系统态断**：GitHub CI 绿/master push 不在事件面——并码部署靠人肉
   轮询 `gh pr checks`。
3. **长任务完工断**：deploy_image.sh 跑完零投件——部署进度只能看日志。

## 对策（六件已实跑）

- **GHEVT** `/opt/meta/bin/ghhookd`：GitHub webhook→mbus 桥（验签/GUID幂等/
  白名单吞/fail-closed），事件源补环
- **EXTAUDIT**：dispatcher 扩扫 meta 票面 `收口待审`→自动 spawn 审计
- **REWORK**：`驳回:`→自动 spawn 返修（R1/R2 自派，R3 熔断升人裁——防机器死循环）
- **门4**：audit-passed+allow.list→`gh pr merge --merge`（五闸 fail-closed）
- **门5** `bin/door5-deploy.sh`：master头+CI绿+allow.list→deploy_image.sh
  （sha级幂等/四安全闸/5min间隔）
- **DEPLOYNF**：deploy_image.sh EXIT trap→MANAGE 箱投 DEPLOY-<sha> 件

## 实证锚

2026-10-31 端到端首跑：allow.list 落行→CI绿闸→deploy_image 自拉→
`o-core:adf8b34cd2b1 healthy`→DEPLOY 投件回箱，5min 零人工。

## 教训

- 授权闸（allow.list 手工行）是唯一设计内人工点——授权即放行，放行即全链。
- 票面字段约定须与解析正则同源核验（EXTAUDIT 一审驳回实案：`伪票号：` 正则
  vs `伪票号 \`X\`` 票面实写——自造格式 vs 现票面漂移是沙箱通病）。
- 判词件多 SHA 同字段行=解析取首值歧义（DOOR5 三 SHA 案，二审席按 HEAD 全态
  对账才零实害）——字段约定待理票。
