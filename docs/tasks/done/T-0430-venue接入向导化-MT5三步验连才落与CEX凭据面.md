# T-0430 venue 接入向导化——MT5 三步验连才落+CEX 凭据面

- 状态：完成（ops_event 完成行 2026-11-07 在案——机械轨票翻）
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

- 状态：待验（施工完——分支 t/0430-onboard-wizard 已推送，绿戳对口）
- 领取：外派施工（O-Dev 径）——claude-t0430 @ /opt/o-wt/t0430 @2026-11-07
- 实现（三步向导 verify-then-write）：
  - 后端 cmd/core/broker_face.go 重写：SubmitBrokerCred 形闸（proto 收窄
    mt5|cex——mt4 提交拒 400〔业主令不处理〕；cex venue 册首档 okx
    fail-closed）→verify-then-write（MT5=mtapi connect+AccountSummary
    实测 45s 帽，Session 窄面 seam=dialSession 桩测注假；CEX=venue 只读
    端实测 okx=/api/v5/account/balance，cexBalance seam）→验败
    CodeInvalidArgument 即回原因零落表（先验后开写——零行保证构造性
    成立，skill tx-rollback 同语义更简形）→cryptoutil.Seal 密文轨单事务
    族 upsert（mt5=login/password/server+host 选填；cex=<venue>.cex.
    api_key/api_secret/passphrase 三件）→ops_event「venue凭据提交」
    审计行加 esc_scope=venue（phase 派生源）；凭据件永禁入审计注/响应
  - ListBrokerOnboard 扩面：cex 族+存量 okx.* 无中缀三键（首户遗留，
    键名亲证 okxpriv/collector.go:64 同名）读面并入同表；接入态 phase
    三源派生 registered→connecting→verified/failed（末拍败径>管道态
    probed/mapped/paper>提交审计行；台账 registered=connecting；cex
    落库即 verified——提交门禁构造保证；mt4/无凭据行=空 raw 态如实
    显形）——判态服务端单源，前端零判据纯渲染
  - 前端 brokerboard.tsx 三步向导替平表单：①venue（MT5 手填+「broker
    搜索面待补」注记——mtapi SearchBroker 类 RPC 缺席亲证：internal/
    gen/mtapi 仅 Connection/MT5/Streams/Subscriptions/Service/
    QuoteHistory/TickHistory 七服务；CEX=okx 首档枚举）→②凭据（MT5：
    login/password/server+host 选填双轨备轨；CEX：api 三件）→③预览
    （掩码不回显——raw 值零渲染）+「验证并接入」；败=即回原因显形留步
    返改（凭据未落库）；提交成功凭据即弃回步1
  - 入口归位（票面二选一钦定施工裁）：顶导航「接入」语义位——新路由
    /onboard+Onboard 页挂卡，复盘页第三卡撤（venueboard 邻接建议形
    不选：邻接位=复盘页本卡原位，即业主「语义不搭」判的病灶位）
  - proto 扩面：SubmitBrokerCredRequest+host/api_key/api_secret/
    passphrase 四字段；Response+verified/verify_note；BrokerOnboardRow
    +phase（registered|connecting|verified|failed）
  - 消费侧不动：onboard 四拍管道/credscan '.mt5.' 族锚/T-0377 提案径
    零改（小心点③守）；MT5 验连按 skill canon 序 connect+AccountSummary
    （SymbolList 留管道探测拍——能力账序不重复烧）
- 验收锚对账：
  - 红桩 MT5 假凭据→验连败→app_secrets 零行落：PASS
    （TestSubmitBrokerCredVerifyFailRedStake 三子测——connect 败/
    AccountSummary 败，各断言 400 携原因+零凭据行+零审计行）
  - CEX 假 key 同断言：PASS（同上第三子测——cexBalance 桩返败，零行零审计）
  - 真形桩→凭据落 <broker>.mt5.* 四件+行态 verified：PASS
    （TestSubmitBrokerCredVerifyThenWrite——fakeSession 喂真 connect
    回包〔Session seam 注假禁桩真 venue〕，exness.mt5.{login,password,
    server,host} 四键密文回解=原值+读面 phase=verified）
  - CEX 桩→<venue>.cex.* 三件落+行态 verified（验连桩返 totalEq）：PASS
    （TestSubmitBrokerCredCEX——okx.cex.* 三键回解=原值+totalEq=7.89
    摘要+phase=verified+凭据件零出面零回显）
  - UI 桩测三步流 domtest（步序/返步/败因显示）+MT4 选项不存+卡位断言：
    PASS（brokerboard-domtest 29/29——步序①②③/上一步返步/败因即回/
    value="mt4" 零渲染/复盘页撤卡+/onboard 注册+Onboard 挂卡/掩码钉/
    phase 态四值）
  - 绿戳对口+全链：PASS（check_all 全链绿戳 exit 0——戳落
    check-green-<hash>）
- 机检：build✓ vet✓ gofmt✓ test✓ domtest✓（29/29）经验库 schema✓
  env✓——check_all 逐闸绿（输出锚=收口 commit 后戳文件）
- 偏离说明：
  - 「四件」解读：验收「凭据落 <broker>.mt5.* 四件」与范围「②凭据
    （login/password/server）」三件表述不一致——按 mt-accounts skill
    ant canon（brokerCompany/brokerServer/brokerHost+login+password）
    取 login/password/server+host=四件（host=双轨寻址备轨 ant
    brokerHost 对形，选填非空才落，无 host 提交=三件）；歧义留决策席裁
  - phase 派生「提交审计行」第三源为施工补面：验连成功落库与管道探测
    拍之间有事件窗，单靠管道态会把刚验过的凭据显示成「候探测」——
    提交审计行（esc_scope=venue，验败零行故只证 verified）补此窗
  - 列表 cex 行 login 列恒空（—）：凭据域唯一出面字段=MT5 login，cex
    api_key 永不解密出（扩 skill「login 唯一出面」律到 cex 族）
- 涉及面：proto/dashboardv1/core.proto、internal/gen+web/src/gen（buf
  正径再生）、cmd/core/{broker_face.go,service.go,broker_face_test.go}、
  web/src/react/components/brokerboard.tsx、web/src/react/pages/
  {Review.tsx,Onboard.tsx}、web/src/react/main.tsx、web/scripts/
  brokerboard-domtest.mjs、docs/数据流.md（同 commit 链路更新）
- 范围外确认：未动 onboard 四拍管道消费侧（onboard 包零改）、未动
  T-0377 提案径、未动部署面/迁移面、未动 app_config 写域（app_secrets
  直写径=T-0407 既有域）
