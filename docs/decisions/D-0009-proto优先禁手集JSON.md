# D-0009 proto 优先，禁手集 JSON（业主令 2026-10-24）

## 业主令
「这里新增1个严重规则，禁止手集json，能使用bufproto的地方，绝对不使用json」

## 裁定（分面钉死）

**定义**：「手集 JSON」= 字符串拼接/模板化组装 JSON（`fmt.Sprintf("{...")`、`+ "\"k\":`、手写 JSON 字面量喂接口）——与 `encoding/json` 对类型化 struct 的 `Marshal/Unmarshal` 区分：禁的是拼装，不是编解码。

### 有 proto 的面=绝对 proto（零例外）
- **mtapi（MT5）全程 protobuf**：Connect/AccountSummary/订单/流全走 `mt5pb` 生成类型——禁为「方便」绕 proto 手拼 JSON 载荷，禁把 JSON 串塞进 proto string 字段当 transport（venue proto 自带 string 字段如 `EventsReply.result` 是 venue schema，非法外之地：能走类型化 RPC 的消息面=禁改吃内嵌 JSON 串）
- **内部消息面（我们能控两端的）**= proto/类型化结构——新建内部线禁 JSON

### venue 只给 JSON 的面=类型化编解码
- OKX REST/WS venue 无 proto——JSON 物理必需，但**只许 `encoding/json` + 类型化 struct**：禁字符串拼装 JSON 请求体/响应模板（`Sprintf` 模板、`+` 拼接均拒）
- DB 持久层 JSONB 列/`app_config` JSON 值同理=类型化 marshal，禁拼装

### 门禁
- `check_no_forbidden_patterns.sh` 增 C 面：`internal/`、`cmd/`、`tools/` 内 `Sprintf`/`+` 拼装出 `{"...":` 形 JSON 字面量=拒
- 复审面：mtapi/内部面新代码出现 `encoding/json` 处理 venue 消息=退回核——`internal/mt5` 中 encoding/json 仅限读 `app_config` 配置值（DB 契约），禁用于 RPC 载荷

## 背景

业主追问 mt5 信息流返回值格式时裁此令——防「手拼 JSON 当协议用」的病原形态：类型漂移无编检（`exec.hedge_band_bps` 裸数字静默坑实证），venue/内部消息面有 proto 时必须吃类型化安全。

## 连锁影响

- `internal/mt5` 现存 `json.Marshal(map[string]any{...})` 为事件落库载荷（JSONB 列——持久层豁免）；`json.Unmarshal` 读 `app_config` 值豁免——其余 venue 消息面零 JSON
- OKX 面继续用类型化 `encoding/json`（venue 唯一协议）
- 新内部组件选型：默认 proto，需 JSON 处写豁免注释+理由
