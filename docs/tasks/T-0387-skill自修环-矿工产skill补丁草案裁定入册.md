# T-0387 skill 自修环——矿工产 skill 补丁草案裁定入册

- 状态：已批（2026-11-06 业主令——Memento/agent-os 技能
  自修环移植：canon 从只加不修变活件）
- 领取：常径待派（DM 常径机派）
- 优先级：P1——自我迭代加固族②；依 T-0385 矿工产出物
- 收口移交归：决策席

## 病灶

skills canon 只能加不能修——mt5-*/okx-* 能力账若与实况
漂移（RPC 改名/端点死/字段变）,错账会带偏施工。Memento
环=发现弱 skill→改进→写回；O 的等价缺口=矿工产出物没
有「skill 补丁」这个去向。

## 范围

1. T-0385 pending 行载荷扩 `target` 域（lesson|skill_patch）
   ——矿工聚类命中 skill 相关签名（坑条目 ID/能力账键/
   RPC 名）时产 skill_patch 型 pending：载荷=目标 skill
   +病灶锚+补丁草案（diff 或节替换文）；
2. 裁定席裁流扩 skill_patch 型：verdict=approved→补丁
   入 skills/<name>/ 落盘（裁定席写径扩 skills/** 白名单
   位——canon 写域按名册钉,机器不直改）；
3. 防漂移闸：补丁入册须附「漂移证据锚」（实测对拍行/
   事件锚——无证据锚的补丁草案拒入,防 LLM 臆改 canon）；
4. kbgen 索引随件重建；
5. 防灌：skill_patch 型日额 ≤2（比 lesson 更严——canon
   是高权文本面）。

## 验收锚

① 桩测：skill 签名聚类→skill_patch pending 落（目标/
   锚/草案齐）→verdict→skills 落盘 diff 断言；
② 无锚草案拒入断言；③ 日额闸；
④ 既有 lesson 型流回归不破（T-0385 域内）；
⑤ check_all 绿戳对口；镜像面件并入发窗。

## 红线

- canon 写域仍裁定席独占——矿工/LLM 不直改 skills/**；
- 补丁必须证据锚随行（臆改拒）；
- 不改 T-0385 pending 表构（载荷扩非表变）；
- 不动真单径。

## 结果回填

（施工写：实现摘要 / 测试证据 / 偏离说明）
