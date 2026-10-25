// 本地 interface 等价件（T-0113）。
import type { Timestamp } from "@bufbuild/protobuf/wkt";

export interface HedgeFlowItem {
  source: string; ts: Timestamp; kind: string;
  amountValue: number; amountUnit: string;
  counterparty: string; txId: string;
}
export interface ListHedgeFlowRequest {
  sourceFilter: string; limit: number; offset: number;
}
export interface ListHedgeFlowResponse { txs: HedgeFlowItem[]; total: number }
