# T-0176 备份链最优构——SG 单信任锚收双机 dump 推 R2

## 态：已批（11-01 清账定性留队·SETTLE-SWEEP）
> 清账定性（`[角色:决策]` @ 2026-11-01，SETTLE-SWEEP/MANAGE-3551）：**已批**——devPGmd备份链恢复演练实录+dev-inbox停供告警桩语义债仍欠（scripts/arm/o-backup-md.sh在案，演练实录档缺席）

审计席：headless @ 2026-10-30

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

## 审计判决（审计席：headless @ 2026-10-30）

**判词：驳回**——锚④双缺不 Through；锚①③实证过、锚②实质过（瑕疵在案）。

对账径：收口链 7fb174f→merge 5299d39 全量 diff 亲读；生产面只读取证
三通道——本机（dev 宿主 models）`/opt/o-backups-md` 实物+backup.log+
crontab；SSH→SG `/opt/o-backups` 实物+backup.log+部署脚本实文+crontab；
SG 侧 `rclone lsl` 桶拍全前缀（凭据经 env 源用零外泄）。

- **锚① SG 次日 log `R2 pushed`+sg/ 新对象 ✓**：SG backup.log
  `[2026-10-28T03:00:39Z] R2 pushed（sg/ 前缀）`，10-29/10-30 同形续响
  在案；桶拍 `sg/` 前缀 5 键实体在桶（o-20261027T055223Z.age 起）。
- **锚② dev 次日出件+dev-inbox 同件+dev/ 对象 ◐（实质过，瑕疵实记）**：
  dev 宿主 `/opt/o-backups-md/o-20261027T193001Z.sql.gz`(1.2M)+sha256
  实物在——19:30Z=次日 03:30+8 cron 槽实证；桶拍 `dev/` 2 键：
  `o-md-20261027T054827Z.age`（10-27 05:52Z 端到端正路走通
  dev-inbox→收罗→R2）+`o-20261027T193001Z.sql.gz.age`（10-28 23:24Z
  修复径落桶）。瑕疵实记：次日件 scp `Connection closed` 首跳失败
  （dev backup.log 在案）未经 dev-inbox 正路，修复径保留原名未走
  o-md- 改名；dev 侧本日 mluser/root 双 crontab+/etc/cron.d 均无
  备份链在册——10-27 后推链实死，惟 T-0234 钦定前缀归位 oci/
  （dev 段退役）属拓扑演进非本票缺陷面，锚面实质达标记 ◐ 不升级。
- **锚③ 各前缀 ≤14 份 ✓**：桶拍 sg/=5、dev/=2、oci/=3 全 ≤14；
  lsl+deletefile 滚动段在仓码（7fb174f diff）与 SG 部署件双证
  （部署面已按 T-0234 转 sg+oci 双前缀、dev/ 历史键冻结留原地——
  冻结键 2≤14 锚值仍立）。
- **锚④ 双缺 ✗（驳回事由）**：
  a) **恢复演练实录脱票面**——施工线（9702caa/ea14436 于
     t/0176-pgmd-backup 支）票面回填段含演练实录（pg_restore 空库
     20 表+fin_proposal/fin_opp_eval 行数对拍 12587=12587+age 头
     /rclone 拉回/size 885156 三层验）；merge 5299d39 仅携 scripts
     两件、解冲未携票面回填，且 ea14436 非 master 祖先（git 实证）。
     现票面正本无实录=锚文「入票面」未达标。修复=回填段落回票面，
     docs-only。
  b) **dev-inbox 停供 26h 告警桩语义错位+零实证+实战静默**——实现件
     `find dev-inbox -name 'o-md-*.sql.gz' -mmin +1560` 只检「件滞留
     inbox」（SG 收罗死场景）；锚文「停供」+票面自钦「推链死=响非默」
     =dev 上游推件断→inbox 空→哨兵无可检对象=**静默**。实战反证在案：
     dev 推链 10-27 19:30Z 实死（scp Connection closed）迄今三日，
     SG backup.log 零 CRITICAL——哨兵对真实停供静默实证成立。
     修复=哨兵改检「inbox 最新件龄/到件心跳」（无件即陈化）或独立
     心跳文件。风险注记：同缺陷已由 T-0234 oci-inbox 哨兵同形继承
     （部署件原文对拍），属 T-0234 域列此备查。

审计席：headless @ 2026-10-30
