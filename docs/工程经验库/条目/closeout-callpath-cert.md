---
schema_version: 2
id: closeout-callpath-cert
kind: pitfall
risk: high
status: verified
keywords: [收口, 完工回执, 调用径实证, 桩测, psql -v, CREATE FUNCTION, shell脚本, 审计谓词, O-32269, O-30514]
scope: [docs/工单规约.md D节, scripts/cognition/closeoutcert.sh, ops_event 完工回执行]
summary: 函件/脚本/SQL迁移类交付只审语义不审调用径——桩测全绿≠真实调用过；O-30514 ops_esc() 复刻审过但 psql -c 不插值 -v，esc 写径双径全断生产复踩；收口词须附真实调用回执摘录，机械闸=closeoutcert.sh
first_seen: 2026-11-23
last_verified: 2026-11-23
skills: []
evidence_refs:
  - type: ops_event
    ref: "#32177 psql -c 不做 -v 插值实证——esc 升件写径双径全断；fix/esc-psql-cvar 修（commit c29de91b4）"
  - type: ops_event
    ref: "O-30514 母票——验收漏调用径，语义复刻审过而调用径未实测"
  - type: code
    ref: "scripts/cognition/closeoutcert.sh——diff 触发类判据+实证段形闸+dispatch 拍前预扫升件（工单O-32269）"
---

# 桩测≠调用径——函件/脚本类收口必须附真实调用回执（O-32269）

## 病灶

O-30514 升件写径归一：`ops_esc()` 库内函件语义复刻审过、桩测全绿，
但 `psql -c` **不做 `-v` 变量插值**——`:'var'` 原样进语句成语法错，
esc 写径主/兜底双径全断，生产复踩（#32177，fix/esc-psql-cvar 修）。
桩测替代不了真实调用径：验收口径盲区=函件类件只审语义不审调用。

## 判据（写死——规约 D 节+closeoutcert.sh 同源）

交付 diff（origin/master...支尖）命中任一=**触发类**，收口词 note
必附 `调用径实证:` 段载真实调用回执摘录（实际 psql 输出/实际运行行/
实际 INSERT 返回 id）：

1. diff 新增/修改 `*.sh`（A/M 态）
2. diff 增行含 `CREATE [OR REPLACE] FUNCTION`（.sql/迁移面）
3. diff 增行含 psql 调用形（命令位 psql+参数/连接串）

豁免=纯 Go/文档码件（桩测覆盖调用径）。生效锚前收口件豁免。

## 机理与落点

- `scripts/cognition/closeoutcert.sh`——dispatch 拍前预扫（细胞件）：
  扫近 7d 收口词行（`^(完工回执|\[施工收口：)` 前缀形）→逐件
  `git merge-base+diff` 判触发 → 缺实证段批升件
  `审计/closeout_callpath`（载 `回执#id`——回执行 append-only
  不可补证，已报 id 永久抑重报）。
- `--check <sha> ['<note>']` 单件判——复审/审计席手跑面：
  rc0=过/豁免、rc1=要件缺不审过、rc2=sha 不可解 fail-open。
- 实证段=**形闸非真闸**（标记+≥20字符+含数字）——真伪归复审
  判词面；spec ban 禁「描述调用径」替代真实输出。

## 教训

- 凡「语义对但调用径没跑过」的交付类（DB 函件/shell/SQL 迁移），
  验收必须含一次**真实调用**证据——桩测覆盖的是代码路径不是
  部署环境的实际调用形（psql 方言/权限/连接串差异全在调用径）。
- 审计谓词要机械查不要靠自觉——收口词是机读面，diff 是机读面，
  两者对拍零成本；「不审过」得有闸不是有约定。
