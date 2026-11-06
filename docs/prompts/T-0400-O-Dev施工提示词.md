# T-0400 O-Dev 施工提示词——OCI 臂部署径

领单唯一权威范围：
/opt/o/docs/tasks/T-0400-OCI臂部署径-环约核查防崩环.md
（AGENTS/collab/施工席SKILL先读；范围红线即全部授权）

先决对账（第一动作非写码）：git log/blame 追 SG_PG_DSN
引入 commit+读 docs/数据流.md dual-write 节定「桥写并采
一体 or 误带」——结论落票面回填再动手。

规程件要点：deploy_oci.sh 照 deploy_image.sh 闸式谱
（出处/锁/staging/健康闸/回滚），加 env 环约对账预检
（新件 strings/符号表或 -help 需求集 vs 单元 EnvironmentFile
集——缺键拒换不硬上）。健康闸=md_ticks max(ts) 推进断言。

收口：新 wt→bootstrap→check_all 绿戳→收口文+推支不合并不部署。
**不碰 OCI 生产盒**——脚本交付不执行实换（实换归决策/管理窗）。

## 自审记录

| # | 发现 | 定性 | 处置 |
|---|---|---|---|
| 1 | env 需求集机读法未钦（strings? help?） | 给原则给法不定死 | 工人选实读径落回填 |
