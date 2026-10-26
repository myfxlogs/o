-- 0068 fin_eval_reject 拆表（T-0167/D-0032 §三-3/D-0030 §五：lifecycle opener
-- rejected 落件从 fin_opp_eval 拆出归 fin——SG 本地序列 ID 空间与 PG-md 正本隔离
-- （投影 id 撞键风险根除），fin_opp_eval 桥镜像对 SG 严格只读前提成立）。
-- 1h 幂等窗防振荡行洪=opener 读径原语义（fin_opp_eval verdict='rejected'+
-- evidence->>'reason' 等义迁移）；reason 升显式列（机读直取）；键含 reason——
-- 同标的不同拒因各行留痕不互掩。
CREATE TABLE IF NOT EXISTS fin_eval_reject (
    id        bigserial PRIMARY KEY,
    ts        timestamptz NOT NULL DEFAULT now(),
    inst_id   text NOT NULL,
    template  text NOT NULL,
    direction text NOT NULL,
    reason    text NOT NULL,
    evidence  jsonb NOT NULL DEFAULT '{}',
    UNIQUE (inst_id, template, direction, reason)
);
COMMENT ON TABLE fin_eval_reject IS
  'lifecycle 开仓拒因落件（T-0167 拆表）——SG 钱径拒留痕，ID 空间独立于 fin_opp_eval 正本';

-- 保留账入册（T-0153 清道夫 fail-closed）：拒因留痕=审计账，90d 清窗（同
-- fin_paper_fill 口径——1h 幂等窗后行即死账，留痕价值随窗衰减）。
INSERT INTO fin_data_retention(table_name,class,keep_interval,note)
VALUES ('fin_eval_reject','audit_keep',interval '90 days','开仓拒因留痕 90d 清窗')
ON CONFLICT (table_name) DO NOTHING;
