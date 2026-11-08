---
schema_version: 2
id: oci-ci-sg-key-restricted-bridge-gate
kind: pitfall
risk: medium
status: verified
keywords: [oci, check_all, bridge-audit, ssh, 隧道, restrict, ci, 戳跑]
scope: [ci, check_all, bridge-audit, oci]
summary: OCI 面 SG SSH 键是 restrict 隧道键（command=/bin/false 仅 permitopen）——check_all 的 bridge-audit 远端 psql 面跑不了，离线筛须延后该闸到本盒同 sha 补跑且延后径不得单独产戳
first_seen: 2026-11-09
last_seen: 2026-11-09
evidence_refs:
  - type: commit
    ref: b67cea909
---

## OCI 甩 check_all：SG 侧 SSH 权限比「通道现成」窄——网络闸须延后本盒补跑，延后径不得产戳

- **实证**：提案「check_all 甩 OCI」施工期 OCI 机（149.118.58.98）跑
  check_all 在 `bridge-audit` 步静默死（`set -euo pipefail` 下 `sg_q` 的
  ssh 管道败直杀壳）。根因两层：① OCI→SG 的 `oci-collect` 公钥在 SG
  `authorized_keys` 是 `command="/bin/false",restrict,port-forwarding,
  permitopen="127.0.0.1:15433"...`（隧道专用，任意命令 exit 1 无输出）；
  ② OCI 现有隧道 DSN=`review_rw@127.0.0.1:15433` 对 `fin_eval_reject` 等
  源表 permission denied，`role_table_grants` 也只见 16 行自家授权——
  「有 DSN」≠「有闸所需读权」。
- **教训**：跨机跑全链闸前，先逐闸核对**凭据面**（不只是连通性）：远端
  命令权限（authorized_keys 的 command/restrict）、DSN 角色对每张被查表
  的 SELECT、catalog 视图可见性，三者任一缺都会在 `set -e` 门禁里表现为
  「静默死」而非显式红。把「通道在跑」当「闸可跑」是误判。
- **处置**：OCI 侧 check_all 用 `CHECK_ALL_DEFER_BRIDGE=1` 延后
  bridge-audit——该径只写 `<stamp>.deferred` **不写 check-green 戳**
  （延后本身不能产戳=不构成绕闸面）；本盒 `oci_ci_accept.sh` 见
  `netgates=local:bridge-audit` 后校验 HEAD==被核 sha 且树净，再跑原闸，
  绿才落戳，侧档如实记 `netgates=`。要恢复 OCI 全闸，须另配最小只读
  CI 角色/键（生产授权面变更，归决策席裁，不在施工自动扩权）。
