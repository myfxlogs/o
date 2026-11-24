package okxpriv

// respawn.go —— O-38013 spawn 前置败（凭据缺件族）退避账：acctLoop
// 早退自清 kids 残行+指数退避锚，syncRoster 拍内重 spawn 覆盖。

import (
	"time"
)

// credMissBackoff spawn 前置败退避——2s 起倍增 60s 顶格（会话环
// backoffMin/backoffMax 同律；顶格后每次败再推 60s=按拍续试不放弃；
// var=桩测覆写锚同 rosterPollEvery 先例）。
var credMissBackoff = func(fails int) time.Duration {
	d := backoffMin
	for i := 1; i < fails && d < backoffMax; i++ {
		d *= 2
	}
	return min(d, backoffMax)
}

// noteSpawnMiss spawn 前置败记账——①kids 残行自清：acctLoop goroutine 已退
// 而 map 内 cancel 仍占位=syncRoster diff 恒见「在册」永不重 spawn 的
// O-38013 病灶根修（cancel 幂等——自收本 goroutine ctx 无害）；
// ②credMiss 计数+retryAt 退避锚。
func (s *Supervisor) noteSpawnMiss(venue string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cancel, ok := s.kids[venue]; ok {
		cancel()
		delete(s.kids, venue)
	}
	s.credMiss[venue]++
	s.retryAt[venue] = time.Now().Add(credMissBackoff(s.credMiss[venue]))
}

// spawnOK spawn 前置过（凭据族齐）——退避账清零（会话环内层 dialLoop 自带
// 退避续试，本账只管 spawn 前置段）。
func (s *Supervisor) spawnOK(venue string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credMiss, venue)
	delete(s.retryAt, venue)
}
