---
name: 裁定席
description: 经济案无头裁定席（T-0312——fin_econ_lesson pending 逐案裁 verdict+回改点窄白名单三步+升不裁红线）
domain: 经济
---

# 裁定席 SKILL——经济案无头裁定协议全录（T-0312 第三刀）

## 职责
`fin_econ_lesson WHERE status='pending'` 逐案裁——verdict∈{canon,dismiss,merge}+
理由一句话+回改点词。**只裁经济案**——venue 注册/资金/席位立换归业主撞即升。

## 读面
SG psql（FIN_DSN env 径）`SELECT id,kind,entity_ref,numbers FROM
fin_econ_lesson WHERE status='pending' ORDER BY id`。

## 裁定词集（schema v3 契约同集——回改点词闭集勿自造）
- `kill_trigger:<family>` / `sizing:<x>` / `combo_gate:<x>` / `eval:<x>` / `semantic:<slug>`
- verdict 三值：canon（成例入库）/dismiss（驳回）/merge（并先例）。

## 落面（窄白名单三步——越域禁）
1. 机械词→`SELECT key FROM app_config WHERE key='gate.<类>.<名>'` 在册才
   `UPDATE app_config SET value=to_jsonb('true') WHERE key=<同键>`——
   **key 不在册拒写**（verdict_ref 注明「键未册」两拍分权）；
2. semantic 词→`cd /opt/o && go run ./tools/skillgen <slug> <id>`→
   `go run ./tools/kbgen skills`→git add+commit（skills/econ-* 命名域——
   越域写=越界禁）；
3. `UPDATE fin_econ_lesson SET status='verdicted',verdict_ref='<verdict>:<理由一句话>' WHERE id=<n>`。

## 升不裁
kind 不熟/理由写不出/涉红线（资金/新平台/席位）→ops_headless stalled 行
+**留 pending 不强裁**——裁定席只裁能裁的（dismiss 也须一句话理由）。

## 先例读面
`skills/econ-*/SKILL.md`+`docs/经济经验库/条目/`已册案先读——同型案照
先例裁，新立案留 match 签名（evidence 引 lesson id）。

## 败退
- 查询败/写败→本拍 skip 下拍续（fail-soft 不炸）；
- devin 输出校验败（词集外/无理由）→该行不动留 pending。
