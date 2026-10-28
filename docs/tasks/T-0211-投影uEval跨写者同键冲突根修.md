# T-0211 投影 uEval 跨写者同键冲突根修

> 状态：已立单待施工 | 优先级：P1（宿主迁移复犯级）
> 实证：2026-10-28 OCI 迁入后投影臂崩环 41 次（SQLSTATE 23505
> `fin_opp_eval_key`)——启动对账 `正向缺口` 批插撞自然键，桥断 ~10min。

## 病灶

`internal/projector/mirror.go` `uEval` 的 `ON CONFLICT (id)`——
`fin_opp_eval` 是**自然键状态表**(UNIQUE=(inst_id,template,direction),
id 是发号），投影臂按 `id>sgMax` 判「缺口」批插。

**跨写者场景必撞**：新 md 宿主（重灌/迁移）上的写方对同键发新 id——
SG 存旧 id 行，批插触发自然键唯一违例，`Reconcile` 整体失败→进程退→
崩环（Restart=always 也救不回，同错必复）。

## 本次处置（已做，运维级）

盒 10 行 eval 的 id 逐对 UPDATE 归位 SG 已发号（md 发号域不变式）——
对账过、桥复、信号面回鲜。**但下次宿主迁移/序列重灌=必然复犯。**

## 钦定修法（择一，施工先出设计注）

**甲·自然键 upsert+eval_id 映射层**（推荐方向，动码较大）:
- `uEval` 改 `ON CONFLICT (inst_id,template,direction) DO UPDATE
  ... WHERE EXCLUDED.ts>fin_opp_eval.ts`（同键异 id→更新 SG 既有行）
- 但 `fin_proposal.eval_id` 参照 eval.id——镜像 proposal 时须把 md 侧
  eval_id 映射到 SG 侧同键行 id（投影臂内维护 key→sg_id 翻译，
  或 proposal 批写前先把引用 eval 落 SG 再回查 id）
- 风险面：FK 约束形态先核（fin_proposal.eval_id 是否硬 FK）；
  自然键 upsert 的 ts 单调语义与 WHERE 条款交互须桩测钉死

**乙·发号域对账自愈**（轻）:
- `repairForwardGap` 批插前先按键对账：md 行 (inst_id,template,
  direction) 若已存 SG（异 id)→把 md 行 id UPDATE 归位 SG id
  （本次手工径码化），再按 id 缺口批插
- 收益：同键异 id 永不成缺口；局限：仍是「迁一次撞一次再救一次」

**另议**：eval 语义本就是「每键最新态」——id 列在镜像模型下的存在
意义可复审（若提案引用是唯一 id 消费面，映射层是正解；否则可考虑
镜像键改自然键、id 仅作内部序）。

## 验收锚

- 桩测：造 md 侧同键异 id eval→Reconcile 过、SG 行按 ts 新鲜者赢
- 迁移重演实证：盒上人为造一次「新写者发新 id 同键」→不崩、不丢行
- fin_proposal.eval_id 参照完整性不破（若走甲案）

## 顺带面（同票捎）

- 系统页采集健康板处置文案仍写「查 dev collector」——迁移后宿
  主名陈（前端/健康检查文案面，同 commit 更新）

## 施工回填（@分支 t/0211-ueval-natkey）

**择案=甲**（自然键 upsert+eval_id 映射层）——票内推荐向，正解在「eval
语义=每键最新态，id 仅发号」；乙案仍留「迁一次救一次」复犯窗。

### 落件

1. `uEval`（mirror.go）：`ON CONFLICT(id)` →
   `ON CONFLICT (inst_id,template,direction) DO UPDATE ... WHERE
   EXCLUDED.ts>ts`（ts 单调语义原位不动；键列组自 SET 清单摘——同值
   无改写义）。`pick` CTE 兜 PK 防撞：md id 被 SG 异键行占用（重灌
   序列回掷场景）→`max(id)+1` 另发号——不用 nextval：projector_rw
   无序列 USAGE 位，且投影臂=SG eval 唯一写方，单写者下 max+1 无竞争。
2. `remapEvalIDs`（ops.go）：提案批 eval_id 译码——eval upsert 后按
   自然键回查 SG id 覆写提案批（unnest 三列键联查）。键缺席=源侧
   eval 已删异常→该提案跳过+WARN（SG 无行=下轮 id 集差自重试，
   不毒化整批管）。fin_0053 重指钦定径沿用：提案行自带
   inst/direction/net_apr，eval 引用=溯源面非载荷。
3. 健康板（ops_face.go）：host `dev`→`oci` 十行+处置文案换
   `o-collecto`/OCI 盒径（采集/蒸馏/MT5桥全迁毕的宿主名对齐）。
4. `docs/数据流.md`：eval 镜像键=自然键+eval_id 译码+序列守卫
   现态（撞平自抬 T-0195 形）同步。

### 桩测锚（projector_test.go 三枚，全绿）

- `TestEvalNaturalKeyCrossWriter`：md 同键异 id+提案引用→SG 行持
  自有 id 按 ts 新鲜赢、提案 eval_id 译 SG 发号、重演幂等
- `TestEvalPKCollisionSeqRewind`：md id 撞 SG 异键行→另发号落行、
  异键行不覆写、提案指另发号行
- `TestEvalTsNewerWins`：同键异 id 陈帧不覆鲜行

### 注记

- 生产实证已两役（dev→OCI 迁期 10 行+研究臂产出期 509 行同键异
  id——手工 id 归位各救一次）；本票落地后同类迁徙=零人工。
- `fin_opp_eval_id_seq`（SG）无消费方——eval 全由显式 id/兜底
  max+1 落，序列自抬面留 md 侧（T-0195 原位）。
