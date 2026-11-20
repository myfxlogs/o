-- fin_0222: 建制单元注册表（工单#26444——o-self.timer disable 静默躺尸 2.5d
-- 实证坑根修：宿主建制单元清单×期望态×产出行签名落 canon，注册表=审计轨
-- 本体——期望 vs 实际差分即账，谁停的怎么停的不必追）。
-- 键 ops.unit_registry owner 类（基建注册面同 repair.feeds 先例）：
--   host=models 面（SG/OCI 面注册表留扩展位——同键可追 host 段多册）。
--   units[]: unit=systemd 单元名；expect_enabled=is-enabled 期望字面
--     （enabled/disabled/static/linked/not-found——种子=首拍实测基线，
--     变更=注册表先修码后生效的审计契）；expect_active=is-active 期望
--     （可选——oneshot 派发服务常态 inactive 不钉）；
--     prod_seat/prod_event/prod_period_min=产出行签名（可选——单元活着
--     ≠拍子在产：ops_event 末行 seat/event 龄>3×period=heartbeat_stale）。
--   watch=本地查件节拍（unit_watch.sh 每 period_sec 拍+beat 键）。
-- 判读纪律：判一 enabled/active 差分=unit_state_drift；判二产出行陈=
-- <unit>_heartbeat_stale；注册表缺/空=fail-closed 升件不静默。
BEGIN;

INSERT INTO app_config(key, value) VALUES ('ops.unit_registry', '{
  "host": "models",
  "watch": {"period_sec": 300, "beat_ticket": "UNITW",
            "fact_seat": "建制机", "fact_event": "单态"},
  "units": [
    {"unit": "o-self.timer", "expect_enabled": "enabled", "expect_active": "active",
     "prod_seat": "自我", "prod_event": "判词", "prod_period_min": 360},
    {"unit": "o-self-dispatch.service", "expect_enabled": "disabled"},
    {"unit": "o-deploy-watch.timer", "expect_enabled": "enabled", "expect_active": "active",
     "prod_seat": "DEPLOY", "prod_event": "心跳", "prod_period_min": 90},
    {"unit": "o-deploy-watch.service", "expect_enabled": "static"},
    {"unit": "o-research.timer", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-research-dispatch.service", "expect_enabled": "static"},
    {"unit": "o-ops-relay.timer", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-ops-relay.service", "expect_enabled": "static"},
    {"unit": "o-ops-relay-urgent.timer", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-ops-relay-urgent.service", "expect_enabled": "static"},
    {"unit": "o-cognition.timer", "expect_enabled": "disabled"},
    {"unit": "o-cognition-dispatch.service", "expect_enabled": "static"},
    {"unit": "o-mirror-push.timer", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-mirror-push.service", "expect_enabled": "linked"},
    {"unit": "o-worker.timer", "expect_enabled": "disabled"},
    {"unit": "o-worker-dispatch.service", "expect_enabled": "static"},
    {"unit": "o-audit.timer", "expect_enabled": "disabled"},
    {"unit": "o-audit-dispatch.service", "expect_enabled": "static"},
    {"unit": "o-overwatch.timer", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-overwatch.service", "expect_enabled": "static"},
    {"unit": "o-mech.service", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-feishu-bridge.service", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-collecto-tunnel.service", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-readcred-tunnel.service", "expect_enabled": "enabled", "expect_active": "active"},
    {"unit": "o-oci-bus-tunnel.service", "expect_enabled": "enabled", "expect_active": "active"}
  ]
}'::jsonb)
ON CONFLICT (key) DO NOTHING;

COMMIT;
