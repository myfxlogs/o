---
id: eval-fingerprint-dedup-freshness-starvation-ttl
title: eval 指纹闸去重+72h 新鲜度——评估供给断流被 armed_ttl 当信号死收尸
kind: pit
date: 2026-11-26
source: O-41365 armed_ttl 死因审计（CASHCAT/MEME/CRCL-USDT-SWAP 三尸 probe→retired）
---

# 评估供给断流≠信号死亡——armed_ttl 只见 eval 行不见臂覆盖

**坑形**：`fin_opp_eval` 写径指纹闸（工单#22904 防盘烧）同
(inst,template,direction) verdict+net_apr 未变→跳 INSERT 复用行 id。
叠加 `dirEntryGate`/`signalAlive` 判词=「本向最新 eval verdict='proposed'
∧ ts<72h」——**ts=信念末次变迁时刻非末次评估时刻**。评估臂对某 inst
停评（stale 棒/薄棒/宇宙剔出/投影桥断）>72h→gate 判死→probe 满 ttl
`armed_ttl_expired` 收尸——**臂从未对该 inst 下过 rejected 判词，
信号真伪无人审判，死因实为供给侧断流**。

**三尸实证**（O-41365）：三 inst 全时段零 rejected dir eval、1D 棒
85-106 根 confirmed 鲜活（SG 镜像实测）、臂同期在产其他 inst 卡——
唯独这三键 eval 行断供>72h，probe 恰在新鲜度边界上被 ttl 收走。

**结构放大器**：SG 侧 `fin_opp_eval` 是自然键**单行镜像**
（projector uEval 同键覆写保 id）——镜像上看不到 eval 版本史，
「行龄」只能反映末次投影时刻；md 正本版本史才有全量拍距，
审计/排障时勿在 SG 面上找历史行（会得出「从未评估」假阴性）。

**判别**：probe/armed 死于 `armed_ttl_expired` 且死因链上无对应
rejected eval 行——先查供给面（臂心跳/该 inst 最近 eval ts/投影
水位）再谈信号死；`fin_eng_lesson` 已有 `opp_eval_stall` 签名在册。

**未解挂账**（O-41365 留给下票）：供给断流的上游定位（md 侧停评 vs
projector 投影断）需 md 池直连取证；gate 侧「臂活+inst 在宇宙但无
新行」与「信号真死」目前不可分——修法方向=供给覆盖感知证据源
（评估臂对每 inst 落 coverage/heartbeat 行）而非放宽 72h 判词
（禁线：premise 真死该收尸）。
