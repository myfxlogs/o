# T-0430 venue 接入向导化——MT5 三步验连才落+CEX 凭据面

- 状态：施工中
- 领取：外派施工（O-Dev 径）——claude-t0430 @ /opt/o-wt/t0430 @2026-11-07
- 优先级：P2——接入 UX 根修+CEX 凭据补面（OKX 钥已在库，
  不挡钱径；新 venue 接入者体验=本票标的）
- 收口移交归：决策席
- 关联：T-0379（凭据标准径+onboard 四拍管道——本票不动
  其消费侧）、T-0407（brokerboard 卡本件——重做对象）、
  **必读 skills/mt-accounts**（三步向导/verify-then-write/
  密码明文策略——ant canon 11-06 已迁本仓）+
  skills/mt-gateway（SearchBroker 类 RPC 能力面查证）

## 病灶

- 平表单一把梭：venue/proto/login/password/server 同帧提交
  →写 app_secrets→管道异步验——败因走 ops_event 自报，
  用户无「凭据错了」的即回；
- proto 闸 mt5|mt4 二值——CEX 凭据 UI 零径（OKX 钥系
  credx 工具一次搬，新 venue 只能走工具/手 SQL）；
- 入口埋错页（复盘页第三卡）——venue 接入与复盘语义
  不搭（业主「完全错误」判）。

## 范围

1. **三步向导（照 mt-accounts skill 钦定形）**：
   - MT5：①broker 选（SearchBroker 类 RPC 若 mtapi 面
     有则用，无则保留手填+「搜索面待补」注记）→②凭据
     （login/password/server）→③**验连成功才落库**
     ——后端提交点先跑 connect+AccountSummary 实测，
     败=拒写+即回原因（照 skill tx-rollback 语义：
     无效凭据零落表）；
   - **MT4 下掉**（业主令不处理）：UI proto 选项删 mt4，
     后端闸收窄 mt5|cex 两值（存量 .mt4. 键读面兼容不删）；
2. **CEX 凭据面**：proto='cex'——venue=交易所名（首档
   okx 枚举，他所见 venue 册再扩）；字段 api_key/
   api_secret/passphrase 三件→`app_secrets <venue>.cex.*`
   键形（与 `<broker>.mt5.*` 对形；存量 `okx.*` 无中缀
   三键=首户遗留命名，读面兼容列进同表）；验连=
   用提交凭据调 venue 只读端（okx=/account/balance），
   败=拒写即回；
3. **状态机可见**：卡列表行加「接入态」细化——
   registered→connecting→verified/failed(携败因)——
   onboard 管道态读面已半有（ListBrokerOnboard rows），
   补 verified/failed 两态显示+败因列；
4. **入口归位**：卡挪 venue 板块同区或顶导航「接入」
   语义位（二选一钦定给施工裁——venueboard 邻接为
   建议形）；
5. **密码策略不动**：app_secrets 密文轨照旧（skill
   「明文转发必需+TLS+访问控」论断同构适用——
   cryptoutil.Seal 正径）。

## 验收

- 红桩：MT5 假凭据→验连败→app_secrets 零行落（断言
  verify-then-write）；CEX 假 key 同断言；
- 真形桩：fakeExec/桩 Session 喂真 connect 回包→
  凭据落 `<broker>.mt5.*` 四件+行态 verified；
- CEX 桩：okx 形凭据→`<venue>.cex.*` 三件落+行态
  verified（验连桩返 totalEq）；
- UI 桩测：三步流 domtest（步序/返步/败因显示）+
  MT4 选项不存+卡位断言；
- 绿戳对口+全链。

## 结果回填

- 状态：待派（外派径）
