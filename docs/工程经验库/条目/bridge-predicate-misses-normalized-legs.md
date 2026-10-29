---
schema_version: 2
id: bridge-predicate-misses-normalized-legs
kind: pitfall
risk: high
status: verified
keywords: [活跃集, 投影谓词, 归一腿, 桥外滤除, 静默断流, 订阅集镜像集分叉]
scope: [采集订阅, 桥投影, md_ticks, 派生键谓词, basis/dcd 现货腿]
summary: 采集端订阅活跃集与桥端投影活跃集是两处独立谓词，派生键（现货腿归一、uly 方言剥尾）补丁只补采集端=桥端仍按裸 inst_id 滤行，生产面「订到了但过不了桥」静默断流——采集日志零错、桥侧零拒，唯记账面鲜度闸拒因显形。凡派生键归一新增族腿，订阅集与镜像集双臂必须同源同改。
first_seen: 2026-10-30
last_verified: 2026-10-30
skills: []
evidence_refs:
  - type: task
    ref: T-0216 上线排障（管理令 MANAGE-82——审计残差只捕 collector 订阅半段）
  - type: code
    ref: internal/okxws/subscribe.go activeSet（basis 现货腿归一第四 UNION 臂）vs internal/projector/projector.go refreshSets（裸 inst_id 无归一臂）
  - type: server_fact
    ref: OCI md_ticks BTC-USDT 17:13Z 起流 vs SG 同键滞留 10-25 陈行——serve 谓词滤除实证；切 -mode=dualwrite 后 17:19Z 桥通、286990 首 fill 17:18:55Z 落
---

## 事实/症状

T-0216 返修把 basis 两族现货腿归一臂补进 collector 订阅活跃集
（`okxws/subscribe.go` 第四 UNION：`BTC-USD-261225→BTC-USDT`），但同义归一
未进 `projector.refreshSets`（桥端谓词=`fin_position 未平 ∪ fin_strategy
非retired` 裸 inst_id）。生产链实证：

1. collector 换件后 OCI `md_ticks` BTC-USDT 行流入（订到了）；
2. `qTicks ... AND inst_id=ANY(activeList)` 把它滤在桥外——SG `md_ticks`
   同键滞留 10-25 陈行，`WHERE EXCLUDED.ts>md_ticks.ts` 判桥连新帧都看不到；
3. SG `basisPaperAccounting` 现货腿恒超 `exec.tick_max_age_s` →
   `basis_mark_stale` fail-closed——审计判定「重启 collector 即通」的残差
   修复只通了半段。

同病灶在 dcd：`dcdPaperAccounting` 现货腿价也读 `md_ticks` uly 归一行
（`dcd_spot_stale` 拒因），桥谓词同样无 uly 归一臂——dormant 缺口非 basis
独有。

## 根因或策略解释

- **归一集是分布式谓词**：订阅集（采集端 WS 订什么）与镜像集（桥端搬什么）
  是两个进程两处 SQL，语义必须同构却没有共享真源——补丁落单端即分叉。
- **静默=可观测性黑洞**：桥谓词滤行不报错、不计 rejects；订到不=到 SG。
  故障唯在消费端鲜度闸显形（`basis_mark_stale`/`dcd_spot_stale`）——
  从拒因反推桥谓词是取证捷径。
- **审计残差须全链验**：「重启 X 即通」类残差断言要沿数据流逐段亲证
  （采集行→桥行→消费行三段实拍），单段畅通不等于链路通。

## 复用面

凡「订阅集+镜像集」双层派生键结构（venue 采集→桥→消费）：

1. 派生键归一（剥尾/方言/跨键映射）新增时，订阅谓词与桥谓词**双臂同改同验**
   ——改单端必查对端谓词是否同源；
2. 验收锚=「消费端读到新键鲜行」实证（如 SG `md_ticks` 现货行 ts 新+fill 落），
   不以采集端订到为终点；
3. 临时径：projector `-mode=dualwrite` 无谓词全量旁写可通水救急
   （本实例即用其抢通），但旁写泵非正形——根修部署后必须切回 serve；
4. 经验律：报「断流」先分三段实拍（源行/桥行/消费行），段段定位再修。
