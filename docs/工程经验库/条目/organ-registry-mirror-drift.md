---
schema_version: 2
id: organ-registry-mirror-drift
kind: pitfall
risk: medium
status: verified
keywords: [organRegistry, GHOSTS, 器官册, 镜像漂移, floorFocus, 悬置探盲]
scope: [cmd/core, organ_esc, web/react, 探测]
summary: 前后端器官册两处手维必漂——floorFocus GHOSTS 先增而 organRegistry 漏镜=该器悬置探盲区
first_seen: 2026-11-13
last_verified: 2026-11-13
evidence_refs:
  - type: ticket
    ref: 13881
  - type: commit
    ref: 4870f6b38
  - type: doc
    ref: cmd/core/organ_esc.go
---

## 器官册前后端双册手维：漏镜侧=探针盲区

- **实证**：「自我意识」器官 2026-11-13 落地——floorFocus `GHOSTS`
  十器齐（前端正确显 done），而 `organ_esc.go` `organRegistry`
  停在九器漏镜。后果不是显示错而是探测缺：organOrphanEscCheck
  只枚举册内器官，漏镜=该器真悬置（无 done 断言且无 OPEN 工单）
  也不升件——册漏即盲区，工单#13881 附记照出。
- **教训**：同一对象清单两处手维（前端渲染册+后端探针册）必然
  漂移，且漏镜侧沉默（不报错只漏覆盖）。器官落地流程须把「双册
  同键」当验收件：新增器官=GHOSTS+organRegistry 同 commit 同键
  （自我意识键系=自我意识/自观/自感知）。远期可收单源生成，现形
  双册对拍靠新增时自检。
- **关联病灶**：facesnap 白名单与 GetDecisionTrail facts 窗同为
  「探面-界面双册」同构——任一侧新增都要问对侧册子齐不齐。
