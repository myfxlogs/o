---
schema_version: 2
id: split-consumer-miss
kind: pitfall
risk: medium
status: verified
keywords: [migrate_table, 拆件, grep, 消费面, 跟件, deploy_image, check_schema_live, pg_md_migrate]
scope: [部署管道, scripts, 迁移登记表]
summary: 文件拆件只更主消费脚本漏全仓 grep 清点——单文件名 grep 的下游消费面=隐性断点（deploy_image pg_md_migrate 假败/check_schema_live 空参崩溃/登记表查重漏半面/夹具丢链四案同根）。拆件纪律=提交前全仓 grep 文件名逐点跟件。
first_seen: 2026-11-22
last_verified: 2026-11-22
skills: []
evidence_refs:
  - type: server_fact
    source: ops_event
    query: SELECT note FROM ops_esc_pending WHERE id=32765
    observed_at: 2026-11-22T11:20Z
    server_revision: "deploy 60ba0ef51896 pg_md_migrate 假败——reg_md 恒空"
---

# 文件拆件漏跟消费面（split-consumer-miss）

## 坑形

把一个大文件拆成 `X.go + X_ext.go` 后，只更新了「最显眼的
消费脚本」；其余 `grep X.go` 的下游面（部署闸/哨兵/契约检/
测试夹具）仍单读旧件——新件内容对它们隐形，产生**静默漏检
或假败**两类断点。

## 实案（migrate_table 拆链 2026-11-22 四连）

拆 `migrate_table.go` → fin 本件 + `_md` 件，跟件只更
check_migrations.sh REG→REGS。漏四面：
- `deploy_image.sh pg_md_tail`：grep md 条目恒空→reg_md 空→
  pg_md_migrate 假败升件#32765（真水位 1101 已上，纯面败）；
- `check_schema_live.sh`：md 链 reg 空参→`-lt` 崩溃；
- `check_migration_contract.sh`：登记表级版本查重漏 md 半面；
- `check_migrations_dep/rehearsal_test.sh`：夹具丢 md 链=覆盖
  缩水。

## 判词

拆件不是「改一个文件」是「改一个文件名契约」——消费面=全仓
`grep '<老文件名>'` 的点集，提交前必须逐点判定跟件/不跟件并
在 commit 载明。glob 跟件形：`migrate_table*.go` + `grep -h`
（抑文件名前缀保行形）。

## 防点

任何「文件改名/拆分/移动」提交，跑
`grep -rn '<老路径或文件名>' scripts/ deploy/ cmd/ internal/ docs/`
把命中点列进 commit message 逐项标注跟件状态；check_migrations
式 REGS glob 是跟件正形。
