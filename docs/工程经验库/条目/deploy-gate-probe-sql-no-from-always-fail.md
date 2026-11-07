---
schema_version: 2
id: deploy-gate-probe-sql-no-from-always-fail
kind: pitfall
risk: high
status: verified
keywords: [deploy_oci, 健康闸, 探测SQL, 恒败静默, 误回滚]
scope: [deploy, health-gate, probe, fail-closed, oci]
summary: 健康闸探测SQL缺FROM恒语法错→静默回退0→闸恒红恒误滚——闸面自身代码也要进测试面/实跑面，「闸在」≠「闸灵」
first_seen: 2026-11-07
last_seen: 2026-11-07
evidence_refs:
  - type: production
    ref: deploy_oci T-0425 首真跑——新投影器9s灌528行md_tick_scan实证干活,仍被健康闸误滚两连
---

## deploy_oci 健康闸探测 SQL 缺 FROM——自写成日恒败静默坑

- **实证**：`tick_probe` 两探测查询 `'select coalesce(max(ts),epoch)'`——
  无 FROM 子句恒 `column ts does not exist`，`epoch` 未引号同样恒错；
  败被 `|| echo 0` 吞成 0→`t1>t0` 永不真→健康闸**从 T-0400 写成日
  起恒红**，每次部署必误回滚。新投影器实机 9 秒灌 528 行
  md_tick_scan 干活完全正常，照样被滚两次。
- **教训**：
  1. **闸面自身代码也要进测试面**——deploy 脚本的探测查询
     从未在真库跑过即上线，「闸在」≠「闸灵」；
  2. `|| echo 0` 兜底把 SQL 错静默转成「未推进」判词——
     失败面被兜底语义吞掉（错误显形律：探测败与写停要
     分流，不可同归 0）；
  3. 热身节拍要实测——collecto WS 重订阅实机 ~3min，90s
     闸先天不够（同窗修：90→180s）。
- **收口**：probe SQL 补 `from md_ticks`+`'epoch'::timestamptz`；
  探测败与写停显式分流=后续票可议（本坑只根修语法+时窗）。
