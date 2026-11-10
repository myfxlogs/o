---
schema_version: 2
id: hypothesis-cards-adopt-via-hypo-tool
kind: ops
risk: medium
status: verified
keywords: [fin_opp_hypothesis, 假想卡, tools/hypo, OCI-md, PG-md, 收养, orphan_hypothesis, 死信库, 正本桥表]
scope: [研究员产出面, fin_opp_hypothesis, OCI aarch64, tools/hypo 窄门]
summary: 能力面扫描假想卡只落 ops_event 文本≠进评估队列——fin_opp_hypothesis 正本在 OCI PG-md（aarch64），收养须 tools/hypo 窄门 arm64 构建+scp+hypo card 落行；dev 5433/5435 是死信/测试库，往那写=空气。
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: [研究员]
evidence_refs:
  - type: doc
    ref: scripts/researcher-spawn.sh
  - type: server_fact
    source: postgres
    query: SELECT id,state FROM fin_opp_hypothesis WHERE id IN (102543,102544,102545)
    observed_at: 2026-11-10T00:00Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

首轮能力面扫描（cap-scan-2026-11-09）三假想卡只以 ops_event seat=研究员 文本行落账 → orphan_hypothesis_proposed 探针报「假想卡收养断」（升件 #9021）：文本卡不进 `fin_opp_hypothesis` 评估队列，drafted 状态积零消费。

## 根因或策略解释

`fin_opp_hypothesis` 是 md 域桥表，**正本在 OCI PG-md**（o-postgres-md 容器，aarch64 机）——FIN_PG_DSN（5544 隧道→SG）只见镜像；dev 本地 5433/5435 是死信/测试库（KB `researcher-dsn-dead-letter-db` 在案）。表唯一写径=`tools/hypo` 窄门（研究员席 spawn 面钦定：hypo 构建+RESEARCHER_DB_DSN+结构性禁 state-INSERT），直 INSERT 不在授权面。orphan 探针盯的是 proposed 久滞=「族注册+executor 落位」需求积压，文本卡根本不是它要的队列行。

## 处置

收养径（已实证）：`GOARCH=arm64 go build ./tools/hypo` → `scp` 到 OCI → `hypo card` 按卡面插行。三卡落 id 102543（okx/xvenue-negfund-carry）/102544（mt5/xau247-basis）/102545（okx/listed-options），全 proposed 待审。注意两坑：①本机 go run 直连写不进去（架构+授权双不对）；②LegPrim 闭集无 `cfd_long`/listed-option 原语——卡 evidence 需落 grammar_gap 注记（同 #1690 先例）。测试行误插桩数据可 UPDATE 正身（表非 append-only）。

## 证据

- check: `hypo list`（OCI 侧）三卡 proposed 在队；ops_event 文本卡与表行同 key 可对拍
- result: 102543-102545 落账（2026-11-10）；#9021 处置行引本径
- ref: `tools/hypo`；`scripts/researcher-spawn.sh`（研究员席窄门设计）；KB `researcher-dsn-dead-letter-db`

## 反证条件

- check: ops_event 文本卡不经 hypo 而进 fin_opp_hypothesis proposed 队列
- falsifies_when: 上条件成立（存在第二写径，如评估器自取 ops_event 卡文）→ 本条「hypo 唯一收养径」结论失效
