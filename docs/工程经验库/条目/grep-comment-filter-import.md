---
schema_version: 2
id: grep-comment-filter-import
kind: pitfall
risk: medium
status: verified
keywords: [grep 滤注释, import 依赖漏报, 审计假阴性, 依赖断言, -v '//']
scope: [依赖勘察/移植范围审计, grep 审计手法]
summary: 滤注释的 `grep -v '//'` 会把 Go import 路径（含 "/"）行当注释滤掉——「某包零活引用」类依赖断言由此出假阴性；勘依赖面必须含 import 块核查，滤注释只准用行首锚 `^\s*//`
first_seen: 2026-10-25
last_verified: 2026-10-25
skills: [code-audit]
evidence_refs:
  - type: session_fact
    ref: T-0108 TRX 迁移勘察，「TRX 后端零 CMC 活引用」结论被业主纠正——service_hedge_structure.go:18 实 import nox/internal/data/collect/cmc
---

## 实案

勘 T-0108 迁移依赖面时跑 `grep -rn 'cmc' <TRX 码面> | grep -v '//'` 意图滤注释，
结果把 `"nox/internal/data/collect/cmc"` import 行（路径含 `/`，模式 `.*//`
命中）一并滤掉 → 报「零活引用、CMC 依赖已剔除干净」→ 票面源清单漏列
collect/cmc 整链。业主一句「TRX 迁移涉及 cmc 采集源」纠回，复查实锤：
对冲工作台 service_hedge_structure.go 实 import + 读 fin_cmc_* 四表 +
cmcLiveProvider 内存态，是 TRX 对冲腿实源。

## 根因

`.` 在正则会吞任意字符——`.*//` 匹配「行内任意位置出现两个斜杠」，
import 路径、`http://`、`x/y` 表达式全中招。滤注释的正确姿势是
**行首锚**：`grep -v '^\s*//'`，且注意 `code // note` 行内注释属代码行
不应滤。

## 判据

- 依赖断言（「A 不依赖 B」「X 面零引用」）输出前必查 import 块——
  `grep -n 'import' -A 30` 或 `go list -deps`，不许纯内容 grep 下结论。
- 「零引用/零活依赖」类结论 = 高危断言面——同族手法见
  probe-silent-stream-false-negative（无活动态零帧≠能力不存在）。
