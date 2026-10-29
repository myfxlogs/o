# T-0250 projector refreshSets 活跃集归一臂补齐——basis 现货腿+dcd uly（T-0216 根因②根修）

> 状态：**待派** | 领取：虚位 | 优先级：**急诊**（钱径——fill 锚现役于
> dualwrite 旁写泵顶水，非正形不可长持；急诊抢占=管理令 MANAGE-82 钦定）
> 执行径：**无头**
> 立项：决策席 @ 2026-10-30（管理令 20261030-012400-MANAGE-82）

## 面况

T-0216 上线排障实证根因两段：
- ① collector 旧件未订现货腿——已办（arm64 新件 17:12Z 重启，OCI md_ticks
  BTC-USDT 流实证）；
- ② **projector refreshSets 活跃集=裸 inst_id**（fin_position 未平∪
  fin_strategy 非retired，`internal/projector/projector.go:125`）——缺
  basis 现货腿归一+dcd uly 归一双臂，serve 谓词把 BTC-USDT 现货 tick
  滤在桥外；collector 侧 activeSet 第三/四 UNION 臂（T-0216 返修件，
  `internal/data/okxws/subscribe.go:78-83`）已有对应语义，projector
  侧漏配=T-0216 审计残差未捕段。

管理急件直办在役：`o-projector serve.conf -mode=dualwrite` 旁写泵通水
（SG BTC-USDT ticks 17:19Z 鲜、286990 首笔 funding fill
+0.006881@17:18:55Z 落地）——**dualwrite=旁写泵非正形，全量镜像写量
不可长持**，根修部署后须复原 `-mode=serve`。

## 钦定界

- `refreshSets` 活跃集补归一双臂，**语义对齐** subscribe.go activeSet
  第三/四 UNION 臂：
  - basis 现货腿：`regexp_replace(regexp_replace(inst_id,'-[0-9]{6}$',''),
    '-USD$','-USDT')`，限 `template_ref IN ('quarter_basis_long',
    'perp_quarter_spread')` 且 state 非 retired（口径含 paper——记账
    价源同 collector 臂）；
  - dcd uly 归一：`regexp_replace(uly,'-USD(G)?$','-USDT')`——**SG 侧
    表可用性先对账**：subscribe.go 臂 JOIN `md_dcd_product`（PG-md
    域表），SG fin 库若无此表/桥面对应物须如实改造取径（禁盲抄 SQL
    致 relation does not exist——同 T-0187 fin_position 坑型）。
- 桩测：refreshSets 断言归一臂入集（basis paper/live 策略→现货对入集；
  dcd 策略→uly 归一入集；retired 不入）；非 skip 实跑。
- 部署后配置复原衔接口径入结果段（serve.conf 切回 `-mode=serve`=
  部署面动作，施工交付=码面+桩测，复原由部署径执）。
- 链变即 `docs/数据流.md` 投影活跃集段同 commit 更新。

## 范围外

collector 侧（已修已重启）；dualwrite/复原配置实操（部署面）；basis
执行面其他件（T-0216 已审计过）。

## 验收锚

1. 双臂语义对齐实证（diff 对 subscribe.go:78-83 逐臂）+SG 表可用性
   对账注记；
2. 桩测：归一臂入集断言绿（非 skip）；
3. serve 模式生效后产验锚：SG `md_ticks` BTC-USDT 现货行续鲜+
   `fin_paper_fill` 续出（部署复原后由决策/管理验）；
4. 数据流.md 同 commit 更新；
5. 票面回填+KB（若产坑）。
