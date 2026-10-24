---
schema_version: 2
id: deploy-autopath-race-double-open
kind: pitfall
risk: high
status: verified
keywords: [竞态双开, live_switch, 自动径, 手动触发件, 开关翻位, 首仓序, 部署序列]
scope: [cmd/core, tools/openfirst, 投资域开仓面, 运维序列]
summary: 部署含自动开仓径的镜像后翻 live_switch，再跑手动触发件=双径各自开仓——开关不是互斥锁，是双刃授权
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: []
evidence_refs:
  - type: server_fact
    ref: T-0050 事故：exec10（自动径 09:38:54）+exec12（手动 openfirst 09:39:30）双开 $50×2；strategy 101 自动晋 live 即触发
---

# 部署自动径+翻开关+手动触发=竞态双开（T-0050 事故实录）

## 病灶
- live_switch=true 同时授权两径：部署内 lifecycle.openLive（事件驱自动）与 openfirst（运维手动）——**开关无互斥语义**，两径各自过闸各开各仓。
- 施工席翻开关时未意识到新部署的自动径会即刻消费 probe 晋升→数秒内自动开仓；随后手动执行=第二仓。

## 裁决
- 开仓径**任何时刻只许一条活径**：开自动径前手动件禁跑（或反向——首仓手动期 live_switch 保持 false，Execute 的 live 闸天然互斥单径）。
- 翻开关前的强制自问：此刻有几条径在听这个开关？部署新镜像=行为面变更，翻位前先枚举消费者。
- 多策略行同理：翻开关前逐行核 probe 态判据——开关是全局的，晋升是逐策略的（本事故另用 degraded 单仓控制先行）。

## 判据
- 开仓类开关翻位 checklist：①在仓行盘点 ②他策略晋升预判 ③并行径禁用声明。缺任一不翻。
