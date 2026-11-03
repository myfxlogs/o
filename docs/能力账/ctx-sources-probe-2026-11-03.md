# ctx-sources 探针证据（2026-11-03，OCI 出网，真键探针——键不露面）

## UA 隔离矩阵（v1 series/observations + 真键 query）

| UA | 结果 |
|---|---|
| curl 默认 | 200（464B） |
| Go 默认（Go-http-client/2.0） | 200 |
| `o-ctx/1.0` | tarpit 挂死 20s |
| `Mozilla/5.0 ... o-research/1.0` | tarpit 挂死 |

结论：**自造 UA 被 Akamai 边池拉黑**——拦截对 UA 不对客户端栈/协议。

## v2 界桩矩阵（真键）

| 探针 | 结果 |
|---|---|
| `fred/v2/series/observations` + Bearer | 404 |
| `fred/v2/series/observations` + query api_key | 404 |
| `fred/v2/release/observations?release_id=18` + Bearer | 200 |
| v1 `series/observations` + query api_key | 200（基线） |

结论：v2=整 release 批发面（Bearer 头鉴权），无按序列端点；v1=按序列取点正道。

## 落行实证（撤 UA 后 13:44）

`md_ctx_fact` source=fred：DGS10（4.94 @10-30）、DGS2（4.56）、UNRATE、CPIAUCSL 四序列全落。
