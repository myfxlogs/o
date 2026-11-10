---
schema_version: 2
id: depenum-config-reader-register-or-orphan
kind: pitfall
risk: medium
status: verified
keywords: [depConfigReaders, depenum, depblind, facesnap, app_config, 孤儿对, 观测盲, repair.feeds, repair.guard, 读面册, 一键三报]
scope: [internal/portfolio/hunger, depenum.go, app_config 新键接入, dep 册面注册]
summary: 新 app_config 键消费端代码实读而未在 depConfigReaders（depenum.go ④族读面册）登记读者标签→depblind 报「产者活消费∅」孤儿对、facesnap 连报 config/dep_pair/gate_domain 三类新面——一键漏册产 3 件升件噪声；新键接入清单必含读面册登记。
first_seen: 2026-11-10
last_verified: 2026-11-10
skills: []
evidence_refs:
  - type: commit
    ref: 14219ec2a
  - type: doc
    ref: internal/portfolio/hunger/depenum.go:224
  - type: doc
    ref: internal/repair/watch.go
  - type: server_fact
    source: postgres
    query: SELECT id,esc_scope FROM ops_event WHERE id IN (9011,9013,9015)
    observed_at: 2026-11-10T01:40Z
    server_revision: ""
    response_hash: ""
---

## 事实/症状

`internal/repair/watch.go`（采集臂修复环）实读 `repair.feeds`/`repair.guard` 两个 app_config 键已上线在产，但 `depConfigReaders` 读面册未登记 → 探测面连产三件升件：#9011 depblind `config:repair→∅`（产者 repair 窗内行=2 而消费者 ∅）、#9013 facesnap `face:dep_pair:config:repair→∅` 新面、#9015 facesnap `face:gate_domain:repair` 新面。同一漏册病灶按读面/面清单两探针维度各报一遍。

## 根因或策略解释

`depConfigReaders`（depenum.go 224 行起）是 app_config 键域→读者标签的**注册式读面册**：depblind 探针判「产者活而消费∅」时只认册载读者，码面 grep 可见的真实读者不入册=视野外。新键接入时补了消费端代码、过了 check_all（config-write-domain 闸只验写域归属不验读面册），读面册漏登记无任何闸拦——漏报路径是静默的，直到探针夜班扫出孤儿对+新面三连。

## 处置

补册：`depConfigReaders` 加 `repair→采集臂修复环` + facesnap 白名单预登记三个新面（config 对/gate_domain/service）——commit `14219ec2a`。此后新 app_config 键接入清单照走：①消费端代码（cfg.Get*）；②`depConfigReaders` 登记读者标签；③若键域新 → facesnap `faceBaselineWhitelist` 预登记 `config:<域>→<读者>`/`gate_domain:<域>` 面（间歇写席另登记 service 面）。

## 证据

- check: 升件 #9011/#9013/#9015 三键同根；修后部署生效窗口内 depblind 对该对落「依赖对豁免」行留账不再升件
- result: 全绿 merge `32ea7c31f`（2026-11-10）
- ref: `14219ec2a`；`internal/portfolio/hunger/depenum.go` depConfigReaders 册

## 反证条件

- check: 新 app_config 键实读后 depConfigReaders 未登记，但 depblind/facesnap 零升件
- falsifies_when: 上条件成立 → 存在第二读面注册径（如 depenum 自动派生），本条「手动登记唯一册」结论失效
