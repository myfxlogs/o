# T-0230 Phase-0 归档径——arch_signal/arch_eval/arch_eval_reject + 活跃集日快照

> 状态：待验 | 领取：devin-t0230 @ 2026-10-29 | 优先级：P0（D-0036 终裁 §12.5 序3——唯一不可逆缺陷，
> 日行日损）
> 立项：决策.终席 @ 2026-10-28（D-0036 定稿§12.3 Phase 0 钦定）
> 前置：T-0223 存在性列在场（本票触发器依赖其显形兜底）
> 依据：docs/decisions/D-0036 §3-P0-1、§12.2 八问裁1/7

## 病灶

评估/信号/拒绝三族全部「当前态覆盖写」——`fin_opp_eval`（md_1053
态表）、`md_signal`（md_1057 禁时序）、`fin_eval_reject`（fin_0068）
只留最后一拍，历史判词序列不可重建。P5 策略评审/误判归因要的
「该标的历史判词如何演变」永远答不了——**不可逆资产损失**。

D-0018 豁免矛盾同 commit 销：D-0018 §2 给历史表豁免快照禁令，与
md_1053 现态表化并存——本票修订 D-0018 明确「arch_* 为唯一合法
历史载体，现态表禁快照但须 arch 镜像其变化」。

## 钦定修法

- `arch_signal` / `arch_eval` / `arch_eval_reject` 三表：
  **append-only + change-only**（同键值重复跳），180d 起步配置化
  保留期（字段 `arch_keep_days`，90d 拿实测行数复裁）。
- **写径优先做 change 判断**（D-0036 §6-Phase0 钦定——避免触发
  器放大）：源表 UPDATE/INSERT 写径同事务判「值变化→arch 追加」；
  无写径点（纯 PG 触发面）才落触发器——施工调研逐表选型回填。
- **活跃集行情日快照**：cron/systemd timer 日级快照活跃集
  tick/mark（显式标注 archive-only 非业务刷新）。
- **失败不阻断主写**：arch 写失败=旁路告警（健康板行/日志），
  禁回滚主事务——施工钦定实现，回填。
- **备份面同 commit**：arch_* 纳入 PG-md 备份（接入 T-0229 落
  成的 OCI→SG→R2 链——若 T-0229 未毕则先立本地 dump 调度）。
- **arch-源水印回退径**：源表 max(id/ts) vs arch max 日级对拍，
  滞>24h 告警（接入 T-0223 存在性列回退径位）。

## 验收锚（D-0036 §8 P0 全项）

- 24h 可重建任一标的 signal/eval 变化序列（arch 查询证）。
- `arch_*` 行数随「变化」增不随 tick 增（桩测：同值写两遍→
  一行；变值→追加）。
- 决策路径零读 arch_*（grep 主链 SQL 无引用）。
- 主写径延迟无可测变化（迁移前后同查询对拍）。
- arch 写失败桩测：主写事务照常提交+告警行产出。
- D-0018 修订段同 commit；`docs/数据流.md` arch_* 条目同 commit
  更新（链路变更票铁律）。
- 迁移注册+迁移测试过 `scripts/check_all.sh`。


## 结果回填

**选型**：触发器统一三表（非写径改点）——写径调研=fin_opp_eval 写点 6 处
跨 3 包（dcd/grammar/funding/gapscan/hypothesis/projector mirror），弥散面
逐点同事务改判=漏点险+双写放大险；触发器内 change 门单点幂等且未来写
径零漏接（D-0036 §6「无写径点才落触发器」——本面=写径弥散型，单点闸
优于多点改，选型理由即此）。fin_eval_reject 虽单写点仍同构触发器——三表
语义统一免新写径型。

**落件**：md_1075 arch_eval+arch_eval_trg、md_1076 arch_signal+arch_signal_trg
+md_daily_snapshot、fin_0084 arch_eval_reject+trg（均 AFTER INS/UPD 触发器
+IS NOT DISTINCT FROM change 门）；domain.tsv 四表入册（md×3/fin×1）；
migrate_table 登记；scripts/arch_daily.sh（日拍三件：快照/保留/dump 滚动14）；
check_bridge_registry.sh 第四列水印（源 max(ts)−arch max(src_ts)>24h 红+
源有行 arch 零行闸死红——T-0223 留位接入）；D-0018 修订段+数据流.md
arch_* 条目同 commit。

**产侧实况**：OCI md_1075/76+SG fin_0084 已应用（schema_migrations 行在册）；
基线种——arch_eval 11748/arch_signal 532/arch_eval_reject 133 存量起点行+
live 捕获持续（拍后行数续增实证）；cron `5 2 * * * arch_daily.sh` OCI 装毕
（首跑 snapshot OK/retention keep=180d/dump 44K）；四列闸现面全绿（首跑
曾抓 seed 缺口红——闸死语义实证有效）。

**桩测**：scratch 库三表全过——同值双写→arch=1 行；变值→2 行（eval/sig/
rej 六断言全对）。主写延迟=单句 INSERT+单索引读（微秒域无可测变化——
触发器内读自身表零外联）。

**偏离**：备份链面=本地 dump 径先立（T-0229/T-0234 OCI→SG→R2 链他席在工
未毕——本件=零凭据保底径，链落由 inbox 收罗自动并）；arch_eval_reject 落
SG 侧（fin_eval_reject 正本在 fin 链——表随源走律）；保留期清径=日拍
cron 读 app_config.arch_keep_days（缺=180d）。
