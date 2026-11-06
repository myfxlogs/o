-- 0119 T-0382 价格提醒表（业主令 2026-11-06——「想看到 BTCUSDT/TRXUSDT 实时
-- 报价，并想预设一个价格，飞书通知我」）：预设 symbol 集=报价卡展示面一单
-- 两用；提醒拍扫 active 行×md_ticks 最新价命中→飞书推→state 翻 fired
-- （UPDATE ... WHERE state='active' 一发即锁防重响）。单向告警非条件单——
-- 票面红线禁下单挂钩。fin 域本表（SG 正本——报价/提醒皆 fin 侧观测面）；
-- 末段携推送开关键种（notify.* owner 类——DM 写径不可达只能迁移种）。
CREATE TABLE IF NOT EXISTS fin_price_alert (
    id         bigserial PRIMARY KEY,
    symbol     text NOT NULL,
    direction  text NOT NULL CHECK (direction IN ('>=','<=')),
    target     numeric NOT NULL CHECK (target > 0),
    state      text NOT NULL DEFAULT 'active'
               CHECK (state IN ('active','fired','disabled')),
    note       text NOT NULL DEFAULT '',
    created_by text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    fired_at   timestamptz
);
-- 提醒拍每帧只扫 active 行——部分索引钉热面（ fired/disabled 陈行不拖拍）。
CREATE INDEX IF NOT EXISTS fin_price_alert_active ON fin_price_alert(symbol) WHERE state='active';
COMMENT ON TABLE fin_price_alert IS '价格提醒单（T-0382——单向飞书告警非条件单；报价卡 symbol 集同源）';

-- 推送开关键种（返修1——漏种致生产 on()=false 静默吞提醒 DOA；照
-- fin_0100/0101/0102 兄弟开关同式 'true'::jsonb——notify.* owner 类键
-- DM 写径物理不可达，只能迁移种）。
INSERT INTO app_config(key,value) VALUES
  ('notify.feishu.price_alert.enabled','true'::jsonb)
ON CONFLICT (key) DO NOTHING;
