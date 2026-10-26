-- 0042 基金钱口径收口（T-0095 v2+v3）：
--   ① fin_known_residual T-0080 $3.4 尘豁免行清场——豁免面退役，身份口径
--     接管（对拍 venue 侧同口径过滤求和，账外尘在口径内天然排除非豁免）。
--   ② fin_residual_custody 托管如实标表（T-0095 v3 ②岔）：腿平残币不够
--     起投/申购拒=落标（30 天幂等窗）——业主令禁兑换径，要处理业主自行换。
DO $$ BEGIN
  IF to_regclass('fin_known_residual') IS NOT NULL
  THEN DELETE FROM fin_known_residual WHERE ticket_ref='T-0080'; END IF;
END $$;

CREATE TABLE IF NOT EXISTS fin_residual_custody (
    id        bigserial PRIMARY KEY,
    venue     text NOT NULL,
    ccy       text NOT NULL,
    amt       numeric NOT NULL DEFAULT 0,
    marked_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (venue, ccy, marked_at)
);
