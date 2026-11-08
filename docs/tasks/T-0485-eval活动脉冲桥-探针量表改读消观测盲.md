# T-0485 eval 活动脉冲桥——per-template 新鲜度派生表镜像消观测盲（T-0479 裁量径②）

- 状态：已批（CTO 裁量批 2026-11-08——T-0479 呈裁三径取②；径①以
  rejected-only 族 eval 死永盲换消音=劣、径③养告警疲劳=劣，皆弃）
- 领取：无头施工席常径（swe-2-high——projector+migration 面）
- 优先级：P2（族沉默探针日频误报断源——#3226/#2450/#3446/#3445/
  #2869/#2868 六案同签名）
- 收口移交归：决策席
- 立项：T-0479 诊断定案——SG fin_opp_eval 镜像谓词=仅被
  fin_proposal 引用行（D-0030 钦定承重墙，PG-md ~93upd/s 体量闸
  不可全量镜像）→rejected-only 族 SG 侧结构性恒零→族沉默探针+
  eval_zero 量表日频误报；病灶=观测域错位非评估链 bug

## 钦定语义

- 新增小表（如 `md_eval_pulse`）：per-template max(ts)/count(24h)
  派生行——量级=模板数（~10 行），镜像体量可忽略；
- 数据源：projector serve 派生（PG-md fin_opp_eval 侧聚合写——
  走 projector 写域现径，不新开写权）；
- 消费面改读：hunger 族沉默探针（silence.go eval 存活判据）+
  gauges eval_zero 量表——eval 新鲜度改读脉冲表（fin_opp_eval
  镜像行数不再作 eval 活性判据）；
- 「eval 因谓词不可见」与「eval 链死」从此可分：rejected-only
  族脉冲行新鲜=活，脉冲行陈旧=真死——探针语义保真。

## 范围

1. `migrations/` 新表（列集: template/last_eval_ts/eval_n_24h/
   refreshed_at——命名随链序钦定）；
2. `internal/projector/`（eval 拍尾聚合写脉冲行——serve 派生
   现径，谓词承重墙零触）；
3. `internal/portfolio/hunger/silence.go` + gauges 量表读面改读
   （eval 活性判据源切换——探针阈/签名面/类册不动）；
4. 桥册/数据流.md canon 行同 commit 补；
5. 桩测：rejected-only 族脉冲行新鲜→探针静默 断言（修前红
   =SG 恒零误报谱形 fixture+修后绿）。

## 验收

- 桩测断言绿+check_all 对口（migration+projector 面全谱档）；
- 部署后探针同窗不再报 `观测盲:xvenue-carry` 同签名（自验锚）；
- rejected-only 族 eval 死探测能力保留（脉冲行陈旧→真死可报）。

## 红线

- fin_opp_eval 镜像谓词（D-0030 承重墙）零触；
- 探针阈/签名面/类册禁改；
- 不动真径执行/钱径码位；
- 先红桩律适用。
