// api.ts —— O REST shim（T-0113：nox connect client 同签名面——组件零感；
// snake_case REST JSON→camelCase proto 形递归换算；O 未落位面=空回显不伪报）。
import type { GetTrxYieldRadarResponse, GetTrxYieldCurveResponse,
  GetHedgeStructureResponse, TrxAccountRefreshResponse } from './gen/dashboardv1/trx_cex_pb'
import type { ListHedgeFlowResponse } from './gen/dashboardv1/position_pb'
import type { ListWatchRulesResponse, UpsertWatchRuleResponse,
  DeleteWatchRuleResponse } from './gen/dashboardv1/watch_rule_pb'

export function errText(e: unknown): string {
  return e instanceof Error ? e.message : String(e)
}

// camel 递归换算件（REST snake→proto camel——字段名对拍键集无损）。
function camel(v: unknown): unknown {
  if (Array.isArray(v)) return v.map(camel)
  if (v && typeof v === 'object') {
    const o: Record<string, unknown> = {}
    for (const [k, x] of Object.entries(v as Record<string, unknown>)) {
      o[k.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase())] = camel(x)
    }
    return o
  }
  return v
}

async function get<T>(path: string, q?: Record<string, string | number>): Promise<T> {
  const u = '/api/trx/' + path + (q ? '?' + new URLSearchParams(
    Object.fromEntries(Object.entries(q).map(([k, v]) => [k, String(v)]))) : '')
  const r = await fetch(u)
  if (!r.ok) throw new Error(path + ': HTTP ' + r.status)
  return camel(await r.json()) as T
}

export const client = {
  getTrxYieldRadar: (_: unknown) => get<GetTrxYieldRadarResponse>('yield_radar'),
  getTrxYieldCurve: (a: { days?: number }) => get<GetTrxYieldCurveResponse>('yield_curve', { days: a.days ?? 30 }),
  getHedgeStructure: (_: unknown) => get<GetHedgeStructureResponse>('hedge_structure'),
  listHedgeFlow: (a: { sourceFilter?: string; limit?: number; offset?: number }) =>
    get<ListHedgeFlowResponse>('hedge_flow', { source: a.sourceFilter ?? '', limit: a.limit ?? 30, offset: a.offset ?? 0 }),
  trxAccountRefresh: (_: unknown) => get<TrxAccountRefreshResponse>('account_refresh'),
  // 监控面 O 未落位——空集回显非伪报（T-0113 记档；upsert/delete 显式拒）。
  listWatchRules: (_: unknown) => Promise.resolve<ListWatchRulesResponse>({ rules: [] }),
  upsertWatchRule: (_: unknown) => Promise.reject<UpsertWatchRuleResponse>(new Error('watch_rule 面 O 未落位')),
  deleteWatchRule: (_: unknown) => Promise.reject<DeleteWatchRuleResponse>(new Error('watch_rule 面 O 未落位')),
}
