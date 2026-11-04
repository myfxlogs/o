---
schema_version: 2
id: psql-c-variable-interpolation-absent
kind: pitfall
risk: medium
status: verified
keywords: ["psql", "-c", "变量插值", ":'var'", "syntax error at or near \":\"", "SQL 注入", "转义", "shell 拼参"]
scope: [scripts/meta/*.sh, psql 写径, ops_event INSERT 族]
summary: psql -c 不做冒号变量插值——字面量直达服务器报 syntax error（-v 传参形同虚设）。带用户文本（公告标题等任意串）的 INSERT 须 bash 侧单引号双写转义后直拼（gauges 同构），或走 COPY/STDIN 径。
first_seen: 2026-11-05
last_verified: 2026-11-05
skills: [施工席]
evidence_refs:
  - type: server_fact
    ref: 2026-11-05 T-0348 T2 注入干跑 emit rc=1——服务器回 syntax error at/near 冒号，LINE 1 VALUES 尾见裸冒号变量原文
  - type: doc
    ref: scripts/meta/venue_watch.sh emit()（修后单引号双写转义在案）
---

# psql -c 不插值 :'var'——字面量撞服务器语法错

## 事实/症状

`psql "$DSN" -v ticket=T -v note=N -c "INSERT ... VALUES('x',:'ticket',:'note')"`
→ 服务器收到的就是裸 `:'ticket'` 文本：`ERROR: syntax error at or near ":"`，rc=1。
同 DSN 同表直拼字符串的写径（hb()）却正常——差异唯参数形态。
T-0348 合成注入干跑 ops_event 零行落账，全因 emit() 用了 -v 形。

## 根因或策略解释

`:'var'`（带引号字面量）与 `:var`（裸拼接）是 psql **交互/meta 命令层**的插值语法；
`-c` 串按原样透传不进该层（版本实证 2026-11-05 在案 psql 直接回语法错）。
「-v 传参更安全」的直觉对 -c 径不成立——防注入必须在构造 SQL 串的一侧做。

## 处置

正形：bash 侧单引号双写转义后直拼（标准 conforming strings 下仅 ' 需双写）：
```bash
local esc=${1//\'/\'\'}
psql "$DSN" -qAt -c "INSERT ... VALUES('...','$esc')"
```
gauges_tick 升件径同构（存量先例）；标题/URL 任意文本安全（换行/@tsv 已转义层）。

## 证据

- check: `psql "$DSN" -qAt -v note=x -c "INSERT INTO ops_event(note) VALUES(:'note')"`
- result: ERROR: syntax error at or near ":"（rc=1）；同语句 bash 转义直拼 rc=0 落行
- ref: T-0348 票面结果段（修后 T2 干跑 1 行落账+重放零重复）

## 反证条件

- check: 任意 psql ≥9 版本重跑 -v+ -c :'var' 组合
- falsifies_when: 插值成功落行（说明版本行为差异成立，转义非必需但仍无害）

## 适用边界

psql -c 写径（脚本族）。交互 psql 会话内 :'var' 照常可用；
COPY FROM STDIN/生成 SQL 文件再 -f 等径不受此坑影响。

## 伴生量表（T-0332 强制——新坑条目必填，无表=门禁红）

- gauge_name: n/a
- class: n/a
- query: n/a
- threshold: n/a
（豁免由：SQL 构造形机制坑，失败显形=rc≠0 同拍可见，非水位类漂移——无量表适配面。）
