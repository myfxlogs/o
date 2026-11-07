-- 0143 T-0455 候选机批类册——meta.autoapprove.registry 册本体（签名→钦定
-- 文件径白名单）。册=红线词表同源：凡不在册签名一律不过机批（红线域
-- 钱径/凭据/方向锚/新venue/部署窗策略/判词语义 永禁入册——类册面钦定
-- 判据见 T-0455 票面裁定节）。册修=dmconfig 写域（决策席面）。
INSERT INTO app_config(key,value) VALUES ('meta.autoapprove.registry','{
  "观测盲:*":             ["internal/research/sigdist", "internal/portfolio/hunger"],
  "写败:LIFECYCLE-HUNGER":["internal/portfolio/hunger"],
  "写败:LIFECYCLE-MINER": ["internal/portfolio/hunger"],
  "账链断:ops_event":     ["internal/platform/opschain"]
}'::jsonb)
  ON CONFLICT (key) DO NOTHING;
