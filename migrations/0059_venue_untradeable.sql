-- 0059 T-0142 执行径幽灵标的终局分类——venue_untradeable 学习黑名单：
-- catalog state=live ≠ 本户可交易（SHEIN/CSOPSKHYNIX2L 实案——51001 区域/
-- 资质门拒语）。venue 第一次拒=学费一次：拒过的 inst_id 入册，消费面=
-- ①universe 扩件滤（不进订阅/评估）②proposal 起草面滤（同族不再武装）
-- ③openLive 开仓面滤。只对 instrument 级，不波及同 venue 他标的（票面禁线）。
CREATE TABLE IF NOT EXISTS fin_venue_untradeable (
    inst_id           text PRIMARY KEY,
    code              text NOT NULL,
    msg               text NOT NULL DEFAULT '',
    first_rejected_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_venue_untradeable IS
  'venue 拒过的标的黑名单（T-0142）——执行径终局码（51001/51003）学习入册，订阅/评估/起草/开仓四消费面滤除';
