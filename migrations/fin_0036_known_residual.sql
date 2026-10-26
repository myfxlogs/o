-- 0036 已知残差豁免表（T-0081：豁免只压「已定性部分」——新发生 unexplained
-- 超额仍即报；expires_at 超期自动复报防永久盖布；首行=T-0080 粉尘账外 $3.4）。
CREATE TABLE IF NOT EXISTS fin_known_residual (
    id         bigserial PRIMARY KEY,
    venue      text NOT NULL,
    ccy        text NOT NULL,
    amount_usd numeric NOT NULL,
    reason     text NOT NULL,
    ticket_ref text NOT NULL,
    expires_at timestamptz NOT NULL
);
INSERT INTO fin_known_residual(venue,ccy,amount_usd,reason,ticket_ref,expires_at)
VALUES('okx','USDT',3.4,'venue 非 USDT 粉尘账外（ETC/TRX 等）','T-0080',
       now()+interval '7 days');
