-- fin_0238_annualized_floor.sql —— 年化判据部署下限键播种（工单O-34963，
-- 值守裁定#34962 业主授权"你或者O自行决定"）：
--   annualized.deploy_floor_usd=500——部署腿时加权均值低于下限时真账
--   年化判据回退 NAV 口径（小分母放大使判据失真实证：部署$76/净$20.68
--   读 330.61% 被 death_factor 判词消费为假达标，3min 后滑窗 -67% 锯齿）。
--   读败/非正值 fail-closed 缺省 500（cmd/core/annualized.go
--   annualDeployFloor 同源注记）。键域=dm（判据参数域——money「如何去赚」
--   分册 dm 类；config-write-domain.tsv+dmwrite.go 双册同录）。
INSERT INTO app_config(key,value) VALUES
  ('annualized.deploy_floor_usd','500')
ON CONFLICT (key) DO NOTHING;
