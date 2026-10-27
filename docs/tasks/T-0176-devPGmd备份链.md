# T-0176 dev PG-md 备份链——正本库裸奔根修

## 态：可领

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 A 级——PG-md 无任何备份径） |
|---|---|

## 病灶

T-0169 分域后 `fin_proposal`/`fin_opp_eval`/`md_signal`/假想卡等**正本在
dev `o-postgres-md` 诞生**（投影 SG=子集副本非全量）；SG `backup_pg.sh`
日备只罩 SG 库——dev 库零备份=库毁即正本灭（bridge 重放只能重建投影列，
proposal 全字段/eval 全量史/假想卡 SG 侧无本可还）。

## 施工面

1. `scripts/backup_pg.sh` 泛化复用（脚本已支持 PG_DSN 参数+容器探测——
   核其对 `o-postgres-md`/5435 适配即可，禁重抄一份）
2. dev 侧 `/opt/o-backups-md/`（700）+ user cron `0 3 * * *` 错时 SG 半点
3. 滚动份数/sha256/日志形与 SG 同款；`.r2_missing_marker` 降级记同款留位
4. **恢复演练锚**：空库 `pg_restore` 一次跑通记档（备份未演练=没有备份——
   钦定验收口径）

## 条件段（R2 离站——业主配合项，非本票阻塞）

SG+dev 两库备份同爆半径（各存本机盘）。业主供 R2/rclone 凭据后：
`backup_pg.sh` 尾部 rclone+age 推 R2 段激活（脚本已留位）。凭据到位前
本票按本机备份口径验收。

## 验收锚

- dev cron 次日 `/opt/o-backups-md/` 出 sql.gz+sha256
- 恢复演练实录（行数对拍）入票面
- SG `backup.log` 与本机备份日志互不踩踏

## 结果回填

- backup_pg.sh 泛化：BACKUP_CONTAINER 环境变量/第二参（o-postgres-md
  同形适配——容器名/端口探测/dump 径零重抄）；R2 留位同形降级记。
- dev 侧 /opt/o-backups-md/（700）建位；user cron `0 3 * * *` 错时
  SG 半点（错时 00 vs 30 错开——本机件不扰 SG 拍）。
- 实跑锚：o-20261027T054827Z.sql.gz 868K+sha256 同款产出；
  .r2_missing_marker 留位同款降级记。
- 恢复演练实录（钦定口径——备份未演练=没有备份）：空库 o_restore_drill
  pg_restore→20 表全复+fin_proposal+fin_opp_eval 行数对拍
  12587=12587 零差；drill 库演练后 DROP 清零。
- SG backup.log/本机 backup.log 互不踩踏（同形日志各目录径）。

## 最优构回填（钦定拓扑实跑实证）

- backup_pg.sh 泛化+R2 滚动：BACKUP_CONTAINER 第二参/环境变量同形适配；
  R2 键 sg/o-*.age+dev/o-md-*.age 前缀隔离（o-o-* 双前缀癖根修）；
  lsl+deletefile 14 份/前缀滚动段入件；dev-inbox 收罗+陈 >26h CRITICAL
  告警行（响非默）。
- dev→SG 推送件 backup_push_sg.sh：scp o-md- 前缀件→SG dev-inbox
  （dev 零新凭据——主权梯度不破）。
- 部署实跑：SG /opt/o/scripts/backup_pg.sh+push 件就位（bash -n 绿）；
  SG dev-inbox 700 建位；dev cron 30 3 dump→push 链（错时 SG 03:00）。
- 端到端实证：dev push o-md-20261027T054827Z→SG dev-inbox→SG
  backup 实跑→age→R2 双前缀对象（sg/ 9.0M+dev/ 885K）；
  rclone 拉回实件 age-encry 头 9B 验+size 对拍 885156=885156。
- 密文完整+rclone 拉回+sha256 三层验过；解密层=业主私钥件如实记
  （业主配合项——age 密文径在案非本席可演）。
