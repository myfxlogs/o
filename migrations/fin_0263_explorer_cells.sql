-- 0263 探索面双细胞（0261被O-39460在途占跳号钉（工单O-39480——「已部署分解眼」decompWatcher+
--   「机会面普查眼」oppCensus，做梦器官细胞原型·感知件）：
-- ① fin_opp_map 机会矩阵落册——(inst_id,mechanism) 幂等 upsert；
--   stale 列=本拍未刷新的陈值标记（读败/出宇宙键保留旧判定不删）；
-- ② explore.* 阈件八键落册（depenum/config-write-domain 同步登记——
--   探针 fail-closed 键缺不探）；
-- ③ ops_esc_valence 两新类权重——decomp_pos_fragment=60（已部署分解
--   眼——部署面效率信号，低于管道断档 65）、unexploited_opp=70
--   （桌上有菜没点——可执行未评估机会，重于族卡断流）；
-- ④ fin_surface_lifecycle 登记——积累面（census 每拍 upsert 消费）。

CREATE TABLE IF NOT EXISTS fin_opp_map (
    inst_id      text NOT NULL,             -- md_funding/md_ticks 同键（OKX instId 方言）
    mechanism    text NOT NULL CHECK (mechanism IN ('funding','grid','basis','momentum')),
    legs         jsonb NOT NULL DEFAULT '{}',   -- 四腿事实 {spot_buyable,borrowable,perp_exists,cfd_listed[]}
    net_est_bps  numeric,                       -- 年化净估 bps（负=成本吞光；NULL=不可估）
    verdict      text NOT NULL CHECK (verdict IN
                    ('executable','structurally-dead','net-negative','unjudgeable')),
    stale        boolean NOT NULL DEFAULT false, -- true=本拍未刷新（陈值——读败/出宇宙保留不删）
    evidence     jsonb NOT NULL DEFAULT '{}',   -- 全实据（费率/借率/spread样本/腿解析/判词依据）
    updated_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (inst_id, mechanism)
);
COMMENT ON TABLE fin_opp_map IS '机会面普查矩阵（O-39480 oppCensus——inst×机制族判定落册；感知件只述事实，定性/提案归唤知）';
COMMENT ON COLUMN fin_opp_map.stale IS 'true=末拍未评估刷新（事实源读败/inst 出宇宙）——陈值保留供唤知辨新旧，非删除语义';

INSERT INTO fin_surface_lifecycle(surface,accumulates,consumer_ref,terminal_path) VALUES
  ('fin_opp_map', true, 'oppCensus census beat（hungerTick 同槽细胞）',
   '每 census_ivl_h 拍全量重估 upsert；inst 出宇宙/读败行 stale=true 陈值保留——终态=人工/唤知处置后自然陈旧')
ON CONFLICT (surface) DO NOTHING;

INSERT INTO app_config(key,value,updated_by) VALUES
  ('explore.decomp_min_gain_usd','0.5','fin_0263'),
  ('explore.decomp_drag_ratio','1.5','fin_0263'),
  ('explore.decomp_cooldown_h','24','fin_0263'),
  ('explore.census_ivl_h','6','fin_0263'),
  ('explore.census_min_net_bps','500','fin_0263'),
  ('explore.census_esc_max','10','fin_0263'),
  ('explore.census_spread_min_n','3','fin_0263'),
  ('explore.census_esc_dedup_h','24','fin_0263')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('decomp_pos_fragment',60,'已部署分解眼——正分量持续增∧负件吞噬比超阈（bot/paper 两面，工单O-39480）'),
  ('unexploited_opp',70,'桌上有菜没点——opp_map 可执行∧净估超阈∧eval臂未评格（工单O-39480）')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;
