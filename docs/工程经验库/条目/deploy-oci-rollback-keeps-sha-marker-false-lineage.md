---
schema_version: 2
id: deploy-oci-rollback-keeps-sha-marker-false-lineage
kind: pitfall
risk: high
status: verified
keywords: [deploy_oci, 回滚, sha 留痕件, 谱系自检, 假谱系, 臂级跳件, O_DEPLOY_FORCE]
scope: [部署径, deploy_oci, 健康闸回滚, 谱系自检]
summary: deploy_oci 换件先写 $ep.sha 标件再过活性闸；闸败回滚只还二进制不还 .sha——标件谎称新 SHA 而实为旧件，下拍谱系自检「diff 证空」永跳该臂，生产跑陈码无迹；人工兜=O_DEPLOY_FORCE=1 压过
first_seen: 2026-11-07
last_verified: 2026-11-07
skills: [决策席, 外审席]
evidence_refs:
  - type: doc
    ref: scripts/deploy_oci.sh（换件步 install+printf sha>$ep.sha 先于 restart 活性闸；回滚步只 install .bak.$TS 不还 .sha）
---

## 事实/症状

- `deploy_oci.sh` 臂级换件序：`install 新二进制 + printf sha>$ep.sha`
  → `systemctl --user restart` → 活性闸；闸败回滚只
  `install .bak.$TS` 还**二进制**，`$ep.sha` 不回退；
- 后果：标件声称新 SHA、实体跑旧件——下拍谱系自检读标件得
  「部署 <新sha> 起本臂构建源零动」→ **diff 证空永跳该臂**，
  生产陈码且零告警（比静默死更隐：台账谎报已部署）。

## 实证（2026-11-07 发窗实拍）

- o-collecto 换 82bca570 首拍活性闸红（fin_0137 `inst_type` 迁移
  未落——见姊妹条目）→回滚旧件复 active；
- 次拍 `deploy_oci.sh` 谱系自检：`o-collecto 谱系 diff 证空（部署
  82bca5709973 起本臂构建源零动）——跳臂`——标件已谎称 82bca570，
  实体仍是旧件；
- `O_DEPLOY_FORCE=1 scripts/deploy_oci.sh collecto` 压过强换→活性
  闸绿、写流恢复。

## 根因或策略解释

- 谱系自检信 `$ep.sha` 留痕件作臂级部署态——它是**意图记录**非
  **实态证明**；回滚路径只还原可执行体，不还原意图记录，两源
  分叉后自检读错源；
- 对形律同 `oci-binary-stale-code-merged-not-deployed`：单看
  ActiveState/标件不证功能态。

## 处置

- **码修候选**（立票面）：回滚步同拍还 `$ep.sha`（或回滚后删标
  件使下拍「不可证空→照换」fail-closed——单向兜底形）；
- **人工兜**：发现臂被「diff 证空」跳过但疑实态不符——
  `O_DEPLOY_FORCE=1` 压过，并在盒上 `strings 二进制 | grep <新件
  特征>`/`systemd-analyze` 对拍实体版本；
- 部署事后账：`ops_event deploy-oci rc=0 arms=...` 行与盒上
  `$ep.sha`+二进制 mtime 三面对拍。

## 证据

- check: 2026-11-07 发窗 collecto 首拍回滚→次拍自检跳臂实录
  （`[deploy_oci] o-collecto 谱系 diff 证空`）→FORCE 换绿
- result: 标件与实态分叉实锤，跳件静默且会随谱系自检永久延续

## 反证条件

- check: deploy_oci.sh 回滚步还/删 `$ep.sha` 后本条 retired
- falsifies_when: 谱系自检改读二进制内容指纹（sha256/构建戳）
  而非留痕标件后本条 retired

## 适用边界

- deploy_oci 三臂（o-collecto/o-projector/o-research）回滚径；
  不涉 deploy_image（镜像面 tag=产地实体对拍无此缝）。

## 伴生量表（T-0332 强制）

- gauge_name: deploy_oci_arm_sha_vs_binary_divergence
- class: 部署谱系
- query: 盒上 `$ep.sha` 读值 vs `strings $ep | grep <期望特征>` 实态对拍
- threshold: 二者不一致=坑命中（当前无周期探针，人工/发窗后对拍）
