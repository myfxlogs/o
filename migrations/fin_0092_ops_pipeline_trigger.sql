-- 0092 工作流管道事件轴（驾驶舱 v4 三卡合一——ops_event 流水按票重建管道，
-- 禁新表）：ops_event 新插行→NOTIFY ev_pipeline（语句级纯唤醒语义——读面现
-- 查防陈拍，同 fin_0088 ev_headless 径）。o-core Watch LISTEN ev_pipeline→
-- 载荷帧=GetPipeline 全量（活跃票 ≤20 行）。
-- 完结=部署上线（业主裁）：部署事件由 scripts/deploy_image.sh 尾拍落
-- ops_event（seat=DEPLOY,event=部署,ticket='',note=sha rc）——同 append-only
-- 簿零新表；管道完结判定=max(部署.ts)≥票末事件 ts。
CREATE OR REPLACE TRIGGER ev_pipeline_trg
AFTER INSERT ON ops_event
FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_pipeline');
