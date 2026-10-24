---
schema_version: 2
id: verify-harness-verdict-semantics
kind: pitfall
risk: medium
status: verified
keywords: [验证 harness, PASS/DIFF/FAIL 语义, 空集常态, 参数校验拒, 探针判定, 假阴性, 假阳性]
scope: [能力账验证 harness/探针/对拍件, 一切 venue]
summary: 验证 harness 的判定轴必须把「空集常态/参数校验拒/传输超时/真回归」分型——空集判 DIFF、校验拒判 FAIL 都是噪声，噪声淹没真信号
first_seen: 2026-10-24
last_verified: 2026-10-24
skills: [协议面取证]
evidence_refs:
  - type: doc
    ref: tools/okxverify/main.go
  - type: doc
    ref: tools/mt5verify/main.go
---

## 事实/症状

okxverify 首跑噪声行：`positions ∅`→DIFF（空集=账户本平常态非回归）、`transfer-state` 虚参→FAIL（venue 参数校验拒=端点反证在岗非面缺）。噪声行混进真回归同一等级，读表成本翻倍且稀释注意力。

## 根因或策略解释

「想要值 vs 得到值」的对拍轴太粗——venue 响应的真类别有四：成功携值/成功空集/参数校验拒/传输层错。空集与校验拒都是「面在」的证据，FAIL/DIFF 只该标「语义回归」或「面缺」。

## 处置

- 判定轴四分：**PASS**（errfree+语义对）/**EMPTY-OK**（空集为合法常态，单列行注记）/**PARAM-ALIVE**（校验拒证端点在岗，计入 PASS 类但注方言）/**DIFF|FAIL**（仅真回归与面缺）
- 每行 `want` 写语义期望非值期望（"errfree" vs "非空"）
- 断言要真值的行：预造条件（如 transfer-state 须真 clientId——由 mutating 段供）或移出读矩阵

## 证据

- check: okxverify 首跑 3 噪声行 vs 修后语义分型
- result: positions/ordersPending ∅→EMPTY-OK PASS；transfer-state 移出矩阵由真 clTId 实证
- ref: 2026-10-24 两轮 okxverify 输出比对

## 反证条件

- check: 某 venue 空集本身就是病（应有行而无）
- falsifies_when: 出现「空集=真回归」场景——届时行级注记补 want 语义，分型轴仍适用

## 适用边界

适用一切「声明→对拍」式验证件；对生产监控告警同理（空读≠故障）。与 probe-silent-stream-false-negative 同族：都是「缺席信号的解读纪律」。
