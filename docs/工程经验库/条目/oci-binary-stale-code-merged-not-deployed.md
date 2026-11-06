---
schema_version: 2
id: oci-binary-stale-code-merged-not-deployed
kind: pitfall
risk: medium
status: verified
keywords: [deploy_oci, o-collecto, 二进制陈旧, 合并未部署, md_tick_scan, T-0416]
---

# OCI 裸二进制陈旧——码合入≠功能上线

## 现象

T-0401 粗扫层（ScanPoller→md_tick_scan）合并后生产表始终 0 行。
取证：OCI `~/o/collector` 二进制戳早于合入 commit 时刻，
`strings` 无 `okxscan`——跑的是合入前二进制，码在了但从未进
生产径。systemd 单元 active 只是进程活，不证功能在。

## 修法/防法

- 「功能空转」先对拍二进制时刻/`strings` 特征串与合入时刻，
  别先怀疑码错——部署断层是首嫌；
- 合入 o-collecto/o-projector/o-research 三臂码面的票，
  生产实证锚=换件后行为，换件唯一径 `scripts/deploy_oci.sh`；
- SG 侧同律：`docker inspect o-core .Config.Image` 对
  `git rev-parse --short=12 HEAD`（o-core:<sha12> 产地钉）。
