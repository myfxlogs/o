-- 0177 惧值面（「常驻独白+valence 权重」棒2件——情感的机器可造切片：
-- 内生显著性权重,升件有「怕的程度」,议程排序有内生偏好）。
--
-- ① ops_esc_valence——惧值表:esc_key='class' 或 'class/scope'(scope 级
--   覆盖 class 级),weight 0-100=在意程度。判据=离钱/离死的距离:
--   钱径真账破裂/脑死/账链断=高位;自指件(想/梦/问)=低位——
--   想不抢饭碗是结构语义非判词。种子=60d 升件面实测分布,
--   新 esc_class 无登记=基线 50(fail-open 不静默灭新信号)。
-- ② ops_esc_pending——未决升件+惧值一体视图(dispatch 唤知阈/
--   会话议程排序/自态帧「最在意」三面同读一源,防散文多口径)。
--   未决口径与 dispatch/escalateUnresolved 同契:后随行引
--   「升件#id」∨范围引「升件#N-M」=已处置。

CREATE TABLE IF NOT EXISTS ops_esc_valence (
  esc_key text PRIMARY KEY,          -- 'class' 或 'class/scope'(scope 优先)
  weight smallint NOT NULL CHECK (weight BETWEEN 0 AND 100),
  note text NOT NULL DEFAULT ''
);
COMMENT ON TABLE ops_esc_valence IS '惧值表——升件显著性权重(棒2 valence 件;0-100,离钱/死越近越高;自指件压尾位)';

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('账链断',95,'事实账本体断=死邻件'),
  ('对账',85,'钱径真账歧=事实层破'),
  ('守卫环',85,'守卫平/动仓=钱径动作在案'),
  ('幽灵',85,'心跳死=尸体信号'),
  ('缺席/escalation_unclaimed',80,'升件无人领=注意力面失能'),
  ('唤知/cognition_dead',95,'脑死=死邻件'),
  ('资金闸/cap_hit',90,'敞口闸到顶=钱径贴线'),
  ('商业产出',75,'零产出=存在性压力'),
  ('断流',75,'面断流=感知失能(钱径feed同源)'),
  ('唤知/stuck_esc',75,'销后复燃=判词不实自证'),
  ('动机',70,'饿/动机账断=内驱信号'),
  ('缺席',70,'认领面失能基线'),
  ('部署',70,'部署受阻=行动面堵'),
  ('执行径演习',65,'演习凭据=毕业径堵'),
  ('断供',65,'供给面断=器官停跳'),
  ('唤知/codework_stall',60,'码面工作滞'),
  ('淤积',60,'队列陈化=议程积压'),
  ('世模',60,'信念失真=世界模型病'),
  ('读败',60,'读面败=局部感知缺'),
  ('资金闸/idle_capital',55,'闲置=次压非急症'),
  ('观测盲',55,'看不见=感知缺口'),
  ('数据缺',55,'缺数据=评估停'),
  ('能力债',50,'能力缺口=慢性债'),
  ('孤儿账',45,'写了没人读=账面孤儿'),
  ('唤知/focus_stale',45,'焦点失指=注意力指针病'),
  ('唤知/learn_gap',40,'求知票滞=外调无主'),
  ('唤知/muse_claim',35,'muse 自证绿核查'),
  ('唤知',50,'唤知类基线(scope 无专载档)'),
  ('唤知/dream_due',20,'做梦=离线巩固不抢饭碗'),
  ('唤知/muse',15,'思想者醒=最低位(队列轻才唤)')
ON CONFLICT (esc_key) DO NOTHING;

CREATE OR REPLACE VIEW ops_esc_pending AS
SELECT e.id, e.ts, e.seat, e.esc_class, e.esc_scope, e.note,
  COALESCE(vs.weight, vc.weight, 50) AS valence,
  EXTRACT(EPOCH FROM (now()-e.ts))::int AS age_s
FROM ops_event e
LEFT JOIN ops_esc_valence vs ON vs.esc_key = e.esc_class||'/'||e.esc_scope
LEFT JOIN ops_esc_valence vc ON vc.esc_key = e.esc_class
WHERE e.event='升件' AND e.ts > now()-interval '7 days'
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
    AND x.note ~ ('升件#'||e.id||'([^0-9]|$)'))
  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
    AND x.note LIKE '%升件#%' AND EXISTS (
      SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
      WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint));
COMMENT ON VIEW ops_esc_pending IS '未决升件+惧值——议程排序真源(valence DESC,age DESC;dispatch/会话/自态帧三面同读)';
