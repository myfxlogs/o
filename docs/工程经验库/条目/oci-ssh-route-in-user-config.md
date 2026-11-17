---
schema_version: 2
id: oci-ssh-route-in-user-config
kind: pitfall
risk: high
status: verified
keywords: [ssh, ProxyJump, OCI, 隧道, 用户配置不入仓, banner timeout, 路由自愈]
scope: [scripts, deploy, oci, 事件总线, check_all]
summary: 产线 SSH 依赖藏用户 ~/.ssh/config ProxyJump 块=不入仓隐形依赖；根修=调用侧探直连+显式选径源件
first_seen: 2026-11-17
last_verified: 2026-11-17
skills: []
evidence_refs:
  - type: doc
    ref: scripts/oci_ssh_route.sh
  - type: doc
    ref: scripts/oci_bus_tunnel_up.sh
---

# OCI SSH 路由藏用户 ssh_config——产线依赖不入仓之坑

## 坑

2026-11-17 03:12Z dev→OCI 22 口直连死（TCP banner 超时，SG 跳板可达）。
桥闸（check_bridge_registry）、CI（oci_ci_*）、部署（deploy_oci/pg_md）、
事件总线隧道（o-oci-bus-tunnel）全断，机器零告警。

止血=dev 用户 `~/.ssh/config` 写 `Host 149.118.58.98 → ProxyJump SG`——
当场救活，但产线依赖从此藏在**不入仓的用户配置**里：下机、换人、
清环境即失；且行为对账者不可见（读仓不知有跳板）。

## 根修

`scripts/oci_ssh_route.sh`——调用侧自带路由：

- 探直连（6s `ssh -o ProxyJump=none host true`），通→显式 `-o ProxyJump=none`，
  败→`-o ProxyJump=root@192.229.85.35`
- **恒显式**：与用户 ssh_config 彻底解耦——止血块在不在行为同一，删块零影响
- **每跑重探**：直连恢复即自动回直连，回退件自然作废，无手工删块步
- git-over-ssh（scratch.git）走 `GIT_SSH_COMMAND` 同径

打点面：桥闸/schema_live/deploy_image(pg_md)/deploy_oci(SSHOPT 单点)/
oci_ci_{provision,push,accept}/o-oci-bus-tunnel（ExecStart→oci_bus_tunnel_up.sh
壳，重起即重探）。

## 教训

- **产线依赖不许藏用户级配置**（ssh_config/env/rc）——径在仓里或没有径。
- 止血块可救场但必须记为欠账：入仓根修前列「拆块验证」步骤，
  否则止血即埋雷。
- 路由类自愈的正确形态=「探测+显式选径」而非「配置回退」——
  显式使行为与外部状态解耦，探测使恢复自动。
