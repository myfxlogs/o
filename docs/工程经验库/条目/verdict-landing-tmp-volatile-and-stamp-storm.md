---
schema_version: 2
id: verdict-landing-tmp-volatile-and-stamp-storm
kind: pitfall
risk: high
status: mitigated
keywords: [sweep, 清债排查, 判词缺位, /tmp 易失, check_all, 绿戳, 共享测试库, 锁风暴, lander]
scope: [meta, eval, headless-audit, T-0249, pre-push]
summary: 批量审计判词落 /tmp 剥态副本=tmp 一清判词全灭（11 票判词缺位）；审计各自 check_all 盖戳并发互踩共享测试库 :5434=1800s 超时潮——判词落定须仓内 worktree commit+串行收口
first_seen: 2026-10-30
last_seen: 2026-10-30
evidence_refs:
  - type: doc
    ref: ~/audit-sweep-20261029-harvest.log（判词缺位 11/31）
  - type: commit
    ref: 33ed21a
  - type: ticket
    ref: docs/tasks/T-0249-历史票假完工清债排查.md
---

## 事实/症状

T-0249 清债排查代际一实收 31 票中 11 票『判词缺位』：审计进程 rc=0 正常
退出且日志明载判词，但票面任一副本（正本/worktree//tmp）均无判词字段。
另两枚实证：T-0009 审计日志自承「判决已落 /tmp/sweep-T-0009-ticket.md
状态字段」——该文件随 /tmp 清理灭失，判词陪葬。

代际二补跑又撞第二病灶：审计按旧补令自行跑 check_all 盖戳推票，多席
（2 sweep 审计+3 在飞单）并发全谱互踩共享测试库 o-postgres-test:5434，
go test 普遍顶 1800s 超时——T-0008/T-0009 两代审计各自在盖戳段烧满
2400s 单票预算被 timeout 斩杀，判词 commit 留在 worktree 未推未并。

## 根因或策略解释

①判词落点选了易失介质：剥态副本存 /tmp 本意防抄答（正确），但把判词
**落**在副本上=把产出写进易失面——tmp 清理/重启即灭，收割器三路
（正本/副本/日志）只剩日志孤证。
②盖戳闸设计假设串行世界：pre-push 绿戳要求 check_all 全谱绿，全谱含
真实 PG 测试，多席并发=同一测试库连接/迁移锁互踩，每个 suite 都顶到
1800s——并发度越高单跑越慢，自激超时潮。

## 处置

- 判词落径钦定（MANAGE-93）：审计 verdict 一律落仓内票面正本——隔离
  worktree（/opt/o-wt/audit-<tid>，分支 t/dec-<tid>-audit）commit；
  /tmp 剥态副本仅为审计输入，落判=未落。
- 落定改串行：审计 commit 即出；新增 `eval/bin/sweep_lander.sh` 收口器
  ——flock /tmp/sweep-checkall.lock 串行盖戳→push→PR→merge --auto，
  周期自扫 audit-* worktree 兜底死审计残判（跨代自愈）。
- 越面保险：lander 推前核 diff 全在 docs/tasks/ 面，越面 commit 拒推。

## 证据

- check: `grep 判词缺位 ~/audit-sweep-20261029-harvest.log`；`ls /tmp/sweep-T-*.md`
- result: 11 票判词缺位；/tmp 副本全灭
- check: `pgrep -af 'go test ./...'`（10-30 14:xx 窗口）
- result: 5–6 并发全谱 suite，o/internal/projector、dcd、guard 各顶 1800s FAIL
- ref: meta 33ed21a（sweep_lander）/ 94e4c4c（driver+queue）

## 反证条件

- check: 判词落 /tmp 后经 reboot/tmp 清理仍可被 verdict_resolve 三路解析
- falsifies_when: /tmp 在本机被证明持久（systemd-tmpfiles 不清理）——则
  病灶①不成立（病灶②并发盖戳与介质无关仍立）

## 适用边界

适用于一切「无头批量审计+判词机读」轨（eval/清债/回放）；单机单测试库
环境。判词类产出物禁易失介质的道理通用于任何 .done/.out 类机读标记。
