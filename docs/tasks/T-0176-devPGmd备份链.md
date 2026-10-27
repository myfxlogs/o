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
