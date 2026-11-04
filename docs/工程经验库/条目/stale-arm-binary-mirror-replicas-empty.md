---
schema_version: 2
id: stale-arm-binary-mirror-replicas-empty
kind: pitfall
risk: high
status: verified
keywords: [部署漏件, projector, bridge-reverse, 反向镜像, o-research, 断供, stakeCarryScan, 二进制陈旧, schema先码后]
scope: [internal/projector, internal/research/hypothesis, OCI盒, 部署面, T-0326, T-0337]
summary: 反向桥票只重建消费臂（o-research）漏搬运臂（o-projector）——bridge-reverse 副本表 DDL/授权/消费码全落而镜像零行，stakeCarryScan 输入三件缺席静默零产 12h+；部署件清单必须含「机制所在进程」重建项
first_seen: 2026-11-04
last_verified: 2026-11-04
skills: []
evidence_refs:
  - type: doc
    ref: docs/tasks/T-0337-stakeCarryScan生产断供-模板卡产而concrete评估零行.md
  - type: doc
    ref: internal/projector/reverse_fin.go
  - type: server_fact
    source: postgres
    query: "SELECT count(*) FROM fin_trx_vote_apr（PG-md副本）→修复前0/修复后2442；SG正本2441"
    observed_at: 2026-11-04T08:01:00Z
    server_revision: "OCI PG-md + SG PG-fin"
---

## 事实/症状

T-0326 返修①把五张 fin 表升 bridge-reverse（md_1087 副本 DDL+SG 八表授权
+reverseFinMirrors 全表 diff 码+o-research 重建重启，04:56Z 全落）——部署
偏离说明明记「o-collecto/projector/sweeper 未动，旧件对 md_1086 零写依赖
如实」。判词漏了 projector：**副本表建给的是 projector 的写面**——镜像码
reverse_fin.go 在该进程里，旧件（2026-11-03 15:30 build）无
reverseFinMirrors→五副本建而恒空。stakeCarryScan 三件套②③
（trx_onchain/cmc_realized 读面）缺席→「该对不评如实」静默跳→concrete
cmc:TRX 零产。T-0333 部署窗验项④红项根即此。

## 根因或策略解释

部署完备性判别式按「新 schema 谁写」漏了「新机制码在哪个进程」——bridge-
reverse 的搬运者是 projector 不是消费臂；消费臂重建只解决「读得到表」
（42P01→行空），搬运臂不重建=副本永远零行，缺席分支照常静默。与
domain-switch-cross-domain-read-face-audit 同谱系（跨域读面缺席静默化），
病灶层从「表不存在」下沉到「副本存在但无人填」。

## 处置

T-0337：dev `GOARCH=arm64 GOOS=linux go build ./cmd/projector`（master
56bc0d9=origin/master）→scp→`systemctl --user restart o-projector`→boot
Reconcile 首拍五副本全满（vote_apr 2442/account 3041/txs 814/cmc_history
39/cmc_positions 8390，与 SG 正本逐行合）。判别式补钉：**部署票据点清单=
「受影响进程」非「受影响文件」**——反向桥类机制票的验收锚必须含搬运臂
二进制 build 时点>机制 commit 时点+副本行数>0 实录。

## 证据

- check: `grep -ac 'fin 副本群' ~/o/projector`（OCI 运行件）
- result: 修复前=0（旧件无 reverseFinMirrors）/新件=1；五副本 0→满行
- ref: T-0337 施工回填；projector.bak-t0337 旧件留档

## 反证条件

- check: 部署票若另有 projector 重建行/ops_event 部署行含 projector 项
- falsifies_when: 证明 04:56Z 后 projector 已是新件仍副本空——则根因在镜像
  码本身非部署漏件

## 适用边界

bridge-reverse/投影类「码+schema+授权三件套」部署票通用；单进程自含机制
（采集器读自己写的表）不在此谱。断供定性残余面：副本满行后 concrete 仍
零者=venue 事实缺席（本案 cmc 短腿 10-31 真平仓 qty=0/okx funding 负号），
属如实缺席非断供。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: stake_carry_eval_zero
- class: 断供
- query: SELECT count(*) FROM fin_opp_eval WHERE template='stake-carry' AND ts>now()-interval '24 hours'
- threshold: -1（v<1 即触——评估臂近24h产出零行）
