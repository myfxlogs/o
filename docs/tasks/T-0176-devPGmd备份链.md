# T-0176 备份链最优构——SG 单信任锚收双机 dump 推 R2

## 态：可领

| 立项 | `[角色:决策]` @ 2026-10-27（债务盘点 A 级；同日 R2 凭据挖回——
/opt/nox-backups/backup-r2.env 在役，arb-backup bucket 实证可写，
SG 侧 env 已落位今晚自验） |
|---|---|

## 最优构（钦定拓扑——非「能用」级）

```
dev cron 03:30  pg_dump o-postgres-md → /opt/o-backups-md/*.sql.gz+sha256
               → scp 最新件 → SG:/opt/o-backups/dev-inbox/
               （复用既有 root SSH——dev 零新凭据，主权梯度不破）
SG  cron 03:00  pg_dump → /opt/o-backups/*.sql.gz（既有）
               → age 加密（本机 dump + dev-inbox 件统一一条加密径）
               → rclone → r2:arb-backup/{sg/,dev/} 两前缀
               → R2 滚动 14 份/前缀（lsl+deletefile——o 脚本现缺）
               → dev-inbox 陈 >26h → critical 告警（推链死=响非默）
```

**凭据主权**：rclone.conf+backup-r2.env+AGE_PUBKEY=**SG 独占**；dev 纯
SSH 推件零新凭据（与 T-0170「dev 永不出现钥」同律）。dev 明文经 SSH
密传到 SG 中转加密——SG 即最高信任域，单加密径单推径可审计。

## 施工面

1. SG `backup_pg.sh` 补 R2 滚动段（照 nox 同款 lsl/deletefile 14 份/
   前缀——o 版只推不转，R2 会无限胀）+ dev-inbox 收罗加密
   +陈件告警（26h 阈值进 alerts critical）
2. dev 侧 `/opt/o-backups-md/`（700）+ user cron `0 3 * * *`：
   dump+sha → scp 最新件 SG dev-inbox（脚本泛化复用 backup_pg.sh
   PG_DSN 参，禁重抄）
3. R2 key 命名 `sg/o-*.sql.gz.age`/`dev/o-md-*.sql.gz.age`——前缀隔离
   滚动互不打（现 o 脚本推 `o-o-*.age` 双前缀癖照修）
4. **恢复演练锚**：R2 拉回一份 age→业主私钥径不可用时至少验
   「密文完整+rclone 拉回+sha256 对」三层（解密层=业主件如实记）

## 已落前置（决策席本席）

- `/opt/o-backups/backup-r2.env` 已 cp 自 nox-backups（600）——
  今晚 03:00 cron 自验 SG→R2 径（桶在役实证至 10-23）
- `.r2_missing_marker` 已清

## 验收锚

- SG 次日 log 出 `R2 pushed`+R2 `sg/` 前缀新对象
- dev 次日 `/opt/o-backups-md/` 出件+SG dev-inbox 同件+R2 `dev/` 前缀对象
- R2 各前缀 ≤14 份（滚动实证）
- 恢复演练实录入票面；dev-inbox 停供 26h 告警桩实证
