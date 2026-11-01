# T-0282 deploy_image.sh 含票锚校验闸（MANAGE-3310 串行闸机内化——未并支覆写根绝）

| 字段 | 值 |
|---|---|
| 状态 | 在飞（无头） |
| 立项 | `[角色:决策]` @ 2026-10-31（审计留案②裁——MANAGE-3310「串行闸+镜像含票锚校验」落定形态；并发覆写实案三度：941c47a2→4131cdbf→PIPEFIX 部署险重演） |
| 领取 | devin-headless-59454 @ 2026-11-01 06:24Z |
| 优先级 | 中（系统性缺口——每次部署都可能静默回退在飞码；缓冲=超集件人工惯例） |
| 涉及面 | `scripts/deploy_image.sh`（出处闸段加校验）；坑条目随票 |

## 背景

SG 部署锁（`mkdir /opt/o/.deploy-lock`）只管并发互斥——持锁后部署什么内容
零校验。出处闸只验「HEAD ∈ origin/master∨显式放行」，不验「HEAD ⊇ 现跑码」。
后果实证：未并分支产物（T-0278 支 4131cdbf、决策 docs 件 941c47a2）两度覆盖
在飞票改动；PIPEFIX 排队部署若非超集件将第三度重演（T-0277/278 静默出网）。

MANAGE-3310 指令原文：「多票并发时拒/排队并报明，杜绝在飞票改动被静默覆盖」。
锁已串行，缺的是**内容向**校验——本票即其落定形态。

## 目标

deploy_image.sh 出处闸后加「现跑镜像祖先校验」：

1. `ssh SG docker inspect o-core --format '{{.Config.Image}}'` 取现跑产地
   `o-core:<sha12>`（取败/非 o-core tag=WARN 放行，不可证伪径不拦）。
2. `git merge-base --is-ancestor <sha> HEAD`：
   - **在祖先** → 放行（部署单调前进）；
   - **本地无此 object** → WARN 放行+提示「sha 不在本地——无法证伪回归，
     superset 件支请留存（t/*-deploy-superset 惯例）」；
   - **非祖先** → die「现跑码含 HEAD 未有件=未并支覆写嫌疑」，
     显式放行键 `O_DEPLOY_ALLOW_REGRESS=1`（WARN 先打，入 ops_event 留痕）。
3. holder/事件面现状不动。

超集件惯例（871ddbeecc96/d22219ed9b4a 两案）因此闸从人肉纪律变机检：
未并支在跑期间，master 直部署=拒，superset merge 件=过。

## 验收标准

- [ ] 三态干跑实证：构造 ancestor / non-ancestor / absent-sha 三 HEAD 各过一遍闸（拒/放/WARN 逐字贴出）
- [ ] 现网真值径：以当前 `o-core:d22219ed9b4a` 为现跑——master HEAD 部署拒、含该 sha 的支放（或当时的真值等价证明）
- [ ] `O_DEPLOY_ALLOW_REGRESS=1` 放行径实证+ops_event 留痕行实查
- [ ] 既有四闸（出处/锁/tag/健康）零回归——脚本 diff 只加不改

## 范围外

- SG 锁本身语义（30min 陈锁/FORCE 抢锁现状不动）
- 排队/报明通道（拒即报明——die 文案+DEPLOY 投件已覆盖 3310「报明」义务）
- 非 o-core 部署面

## 参考

- `scripts/deploy_image.sh`（出处闸 ~L60-75）；MANAGE-3310 原件（meta mail）
- 覆写实案：T-0276 审计注记①②（941c47a2/4131cdbf 两度+超集解法）
- `t/0276-deploy-superset`、`t/pipefix-deploy-superset` 支（superset 留支惯例）

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
