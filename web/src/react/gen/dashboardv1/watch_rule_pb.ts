// 本地 interface 等价件（T-0113）。
import type { Timestamp } from "@bufbuild/protobuf/wkt";

export interface WatchRule {
  id: bigint; kind: string; subject: string; venueKey: string;
  op: string; threshold: number; fireMode: string; cooldownSec: number;
  note: string; enabled: boolean; lastFiredTs: number;
  stale: boolean; level: string; message: string; sourceMatched: boolean;
}
export interface ListWatchRulesRequest {}
export interface ListWatchRulesResponse { rules: WatchRule[] }
export interface UpsertWatchRuleRequest { rule: WatchRule }
export interface UpsertWatchRuleResponse { rule: WatchRule }
export interface DeleteWatchRuleRequest { id: number }
export interface DeleteWatchRuleResponse {}
