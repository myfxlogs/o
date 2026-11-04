# T-0331 坟场复发——worktree/t分支退休纪律与 GC 器

| 字段 | 值 |
|---|---|
| 状态 | **完成**（决策席验收通过 devin @ 2026-11-04——返修①合 master，判词在案） |
| 立项 | `[角色:COS]` devin @ 2026-11-04（业主令「找出来外派修」——淤积巡检实捞第一批） |
| 领取 | （空=未领） |
| 优先级 | P2——淤积类大病：物件越积越多，找东西越来越慢 |

## 病灶（实测实数）

- `/opt/o-wt/` **280 个目录**；`t/*` 分支 **895 条**；
  未合 master **382 条**
- T-0318 清过一次坟场（墓条目在案）——**无退休纪律故疯长复发**：
  每票一树一分支，合完不摘不删，永久堆积
- 附带病：`ops_headless` 死心跳道淤积（seat=DM 10-29 死行在账——
  同类「退休物不清」）

## 钦定方向（施工细化）

1. **退休纪律**：票翻完成→席（或收口件）顺手 `git worktree remove`
   +`git branch -d`（合入即删——未合枝保留记档）
2. **GC 器**：`scripts/meta/wt_gc.sh`——扫超龄件：树≥7天无活 run
   /枝已合入 master → `worktree remove --force`+`branch -d`，
   ops_event `GC` 行录清场数；干跑--dry-run 开关钦定
3. **本次大扫**：280 树+895 枝按纪律清——实录分类计数（已合删/
   未合保/孤儿树删）

## 验收锚

- [ ] `wt_gc.sh` 干跑/实跑双录——分类计数行引票
- [ ] 本次大扫后：`ls /opt/o-wt|wc -l` 与 `git branch --list 't/*'|wc -l`
  实测值落票（前后对比）
- [ ] `ops_headless` 死道清场（>7天无活行删除——计数实录）
- [ ] 退休纪律进 SKILL/AGENTS 章面（合完即摘写进流程节）

## 范围外

- 未合 382 枝的逐枝裁决（合/弃不归本票——记档清单随票，弃枝裁决
  归决策席另案）

## 查档

| # | 风险 | 处置 |
|---|---|---|
| 1 | 在飞工树被误摘（o-worker 跑着的树） | GC 判据必查 unit active+树龄双闸——禁碰活 run 工树 |
| 2 | 未合枝里的独家 commit 被删丢 | `branch -d`（非 -D）天然拒删未合枝——钦定只用 -d |

## 施工回填（devin 无头施工 run @ 2026-11-04，分支 t/0331-wt-gc）

- GC 器：`scripts/meta/wt_gc.sh [--dry-run]`——双闸钦定（unit active 精确
  比对禁碰活 run+脏树跳；枝已合 master 才 `branch -d`——-d 天然拒删
  未合）；清场数 ops_event `GC` 行录。
- 大扫实录：干跑件摘件分类列；实跑**摘 534 件**——`ls /opt/o-wt`
  **282→262**、`git branch --list 't/*'` **897→404**（未合 382 枝全保——
  逐枝裁决范围外记档清单随票）。
- `ops_headless` 死道：实测 >7 天无活行 **0 条**（前扫已清——DELETE 0
  行实录；同类淤积归 T-0332 heartbeat_dead 量表盯）。
- 退休纪律：`skills/施工席` 补退休纪律节（合完即摘写进流程+GC 兜底径）。

## 返修①回填（活 run 闸恒死根修+清场数口径，同支续提 @2026-11-04）

- **违①根修——活 run 闸改实查在飞集**：废 `is-active o-worker-$tid`
  （basename 小写无连字符 vs unit `o-worker-T-0330` 双错位恒死件）；
  现起器先拍快照 `systemctl --user list-units 'o-worker-*'
  --state=active,activating --plain --no-legend` 剥 `o-worker-`/`.service`
  得在飞 TID 集，工树名 `tid_norm` 取首个数字段归一 `T-%04d`
  （`t0330`/`audit-t-0001`/`t-0226`/`t0079-v2` 方言全覆盖）`grep -Fxq`
  精确比对。
- **实证（亲跑随票）**：本拍在飞 `o-worker-T-0331`（本 run 自体）+
  `o-worker-T-0335`，干跑输出
  `SKIP 活run: /opt/o-wt/t0331 (unit o-worker-T-0331)`、
  `SKIP 活run: /opt/o-wt/t0335 (unit o-worker-T-0335)`——闸活；
  同拍 `t0330`/`t0332`/`t0333` unit 已死+已合 → 正常摘（非误伤
  在飞件），`o-worker-T-0330/0332/0333` is-active 亲证 inactive。
- **违②根修——GC 行 note 分类计数**：死变量 `rm_tree/rm_branch/
  keep/skip` 接线（原管道 `|while` 子壳吞计数，改进程替换
  `< <(...)` 主壳计数）+拆 `skip→skip_active/skip_dirty`+新增
  `orphan`（注册在而目录亡——实跑 `worktree prune` 兜底）。实跑
  ops_event **id 1696** note=`wt_gc 清场拍 trees=3 branches=3
  orphan=0 skip_active=2 skip_dirty=26 keep=228`。
- **回填数=note 数=实测差三对**：`ls /opt/o-wt` 265→262（Δ-3=
  trees=3）；`git branch --list 't/*'` 407→404（Δ-3=branches=3，
  另 20 条已合枝挂活树 `branch -d` 拒删——删枝败行实录，非漏记）；
  注册树 266→263（Δ-3=orphan=0+摘 3）。分类总账 3+2+26+228=259
  =266 注册-/opt/o 1-detached 6（detached 无 branch 行不入扫，
  原器同口径）。上拍 534 vs 513 差 21 即此类未分类暗账——本口径
  逐类出账后零差额。
- **缺③**：干跑分类列全档见末节「干跑实录」（返修收口态重拍：
  trees=0——本拍已清；branches=20 已合枝挂树列在案；skip_active=2
  活闸实证；keep=228 未合保）。

## 干跑实录（wt_gc.sh --dry-run @返修①收口，活闸 SKIP 行亲证在列）

```
SKIP 活run: /opt/o-wt/t0331 (unit o-worker-T-0331)
SKIP 活run: /opt/o-wt/t0335 (unit o-worker-T-0335)
摘树: 0 件（本拍实跑已摘 t0330/t0332/t0333 三件，账见违②对账）
孤儿树: 0 件
SKIP 脏树: 26 件 / KEEP: 228 件（未合+年轻，清单=stdout 逐行
  「SKIP 脏树/KEEP」前缀，共 254 行从略）
删枝（预测，实跑挂树拒删）: t/0015 t/0025 t/0043 t/0058 t/0061
  t/0068 t/0071 t/0076 t/0079-v2 t/0083-rw t/0083-vfy t/0085-vfy
  t/0088-fix t/0088-rest t/0088-v2 t/0088-vfy t/0153-ci-fix
  t/0277-cockpit-v41 t/0311-regime-analyst t/0312-regime-push
DRY-RUN 完 wt_gc 清场拍 trees=0 branches=20 orphan=0
  skip_active=2 skip_dirty=26 keep=228
```
