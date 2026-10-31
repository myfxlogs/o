---
schema_version: 2
id: ops-event-ticket-newline-pollution-pipeline-garbled-row
kind: pitfall
risk: medium
status: verified
keywords: [ops_event, report.sh, ticket, 换行污染, 管道, 乱码行, GetPipeline]
scope: [ops_event, pipeline, report.sh, 调用方卫生]
summary: report.sh 调用方多行传参→ops_event.ticket 带内嵌换行→管道按字面票号生成乱码行；落簿前应归一化 ticket
first_seen: 2026-10-31
last_seen: 2026-10-31
evidence_refs:
  - type: provenance
    ref: T-0278 结果回填·观察件（SG o-postgres rows 740/743/745 实拍）
---

## ops_event.ticket 多行传参污染→驾驶舱管道乱码行

- **实证**（T-0278 施工席 2026-10-31 SG 产侧亲查）：`ops_event` rows 740/743/745 的 ticket 字段带内嵌换行（`'T-0277\nT-0270'`、`'T-0276\nT-0276\nT-0276'`）——report.sh 调用方把多行文本（如 relay 回显块）整块当 `<T-编号>` 实参传入，脚本原样落簿。GetPipeline 以 ticket 字面 GROUP BY（pipeline_face.go），乱码票号各生成一条管道行，业主驾驶舱可见（T-0278-stream-push-live.png 末行 `T-0277 T-0270` 即是）。
- **教训**：ops_event 是管道/工单板/完结线共同数据源，ticket 字段=轴键，调用方传参卫生必须脚本内兜底——落簿前对 ticket/note 归一化（剥换行/截空白），不能信调用方自觉。report.sh 现行 `${TID//\'/}` 只防引号不防换行。
- **回改点**：report.sh 第 23 行 psql INSERT 前加 `TID=$(echo "$TID" | head -1 | tr -d '\n\r')`（取首行剥控制符）；存量污染行处置（是否清理/如何归并）归决策席另单裁。
