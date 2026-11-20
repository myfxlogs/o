-- fin_0221_okxgrid_launch_hold.sql —— 工单#26337 拉起歧态势抑重账：
-- launchGate 赎回/拉起任一棒返回不确定态（超时/5xx/断连——票钦）→
-- 记案升件+hold 行落账禁自动重试补单（钱径双花防第一性）。人工核查
-- 后删行放行（esc 行 note 载处置径）。strategy_id 唯一=一策略一 hold。
CREATE TABLE IF NOT EXISTS fin_bot_launch_hold (
    strategy_id bigint PRIMARY KEY,
    venue       text NOT NULL DEFAULT 'okx',
    inst_id     text NOT NULL,
    stage       text NOT NULL,           -- 'redeem'|'launch'——歧态发生棒
    reason      text NOT NULL,           -- 错误原文（升件行同文）
    created_at  timestamptz NOT NULL DEFAULT now()
);
