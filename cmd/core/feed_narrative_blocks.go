// feed_narrative_blocks.go —— O·动叙事渲染主径（工单O-36963）：
// pending 批→窗态两相消化（Phase A 库查补解/超时兜底/闭段 GC；Phase B
// 行序开闭窗/窗员抑/余逐行）→混排块序列。窗态读写/闭标判别在
// feed_narrative.go，成文纯件在 feed_narrative_compose.go。
package main

import (
	"context"
	"sort"
	"time"
)

// narrativeBlocks 叙事模式渲染主径——pending 批→窗态消化→混排块序列。
// wm=当前水位（超时兜底/闭段 GC 上界）。内部先存窗态后返块（发前账律）。
func (s *feedPush) narrativeBlocks(ctx context.Context, rows []feedRow, wm int64) ([]feedNarrBlock, error) {
	st := s.loadNarrState(ctx)
	var blocks []feedNarrBlock

	// ── Phase A：库查补解（feed 剔集闭标）+超时兜底+闭段 GC ──
	// 两遍制：先全量分类定生死（liveRanges=仍占行段/closedNow=本轮新闭段），
	// 再渲染——他窗段 others 须含本轮全部判定结果才不漏嵌套窗归属。
	// O-37510①：剔集闭段闭标延至水位实证（st.Pending）——SendText 败=窗保开
	// 下拍重发叙事，非发前即闭+成员行被抑的整段静默（原径窗内容凭空消失）。
	type paItem struct {
		b         feedBracket
		cid       int64
		feedClose bool
		emit      int64 // >0=本拍出叙事块的闭标行 id（新决闭/发败重发）
		timeout   bool  // 超时兜底逐行退
	}
	items := make([]paItem, 0, len(st.Open))
	for _, b := range st.Open {
		cid, fc := s.narrDBCloseID(ctx, b.Unit, b.OpenID)
		items = append(items, paItem{b: b, cid: cid, feedClose: fc})
	}
	pendOf := map[int64]feedPendingClose{}
	for _, p := range st.Pending {
		pendOf[p.OpenID] = p
	}
	batchHi := int64(0) // 本批行 id 上界=送成后水位——pending 实证锚
	if len(rows) > 0 {
		batchHi = rows[len(rows)-1].id
	}
	var kept []feedBracket
	var liveRanges, closedNow []narrRange
	var pendNext []feedPendingClose
	matched := map[int64]bool{}
	cidOf := map[string]int64{} // unit→已知闭行（feedClose 待解窗——Phase B others 界）
	for i := range items {
		it := &items[i]
		p, hasP := pendOf[it.b.OpenID]
		switch {
		case hasP && wm >= p.BatchHi:
			// 上批载叙事段水位实证=SendText 成——闭段转正（不重发）
			matched[it.b.OpenID] = true
			st.Closed = append(st.Closed, feedClosedRange{Unit: p.Unit, OpenID: p.OpenID, CloseID: p.CloseID})
			closedNow = append(closedNow, narrRange{p.OpenID, p.CloseID})
		case hasP:
			// 发败未实证——窗保开重发叙事，pending 刷新本批界待下证
			matched[it.b.OpenID] = true
			it.emit = p.CloseID
			kept = append(kept, it.b)
			liveRanges = append(liveRanges, narrRange{it.b.OpenID, p.CloseID})
			cidOf[it.b.Unit] = p.CloseID
			pendNext = append(pendNext, feedPendingClose{Unit: p.Unit, OpenID: p.OpenID, CloseID: p.CloseID, BatchHi: batchHi})
		case it.cid > 0 && !it.feedClose:
			// 新决剔集闭段——出叙事块+登记待实证；窗暂留 Open（成员行续抑），
			// 水位实证后转正 Closed
			it.emit = it.cid
			kept = append(kept, it.b)
			liveRanges = append(liveRanges, narrRange{it.b.OpenID, it.cid})
			cidOf[it.b.Unit] = it.cid
			pendNext = append(pendNext, feedPendingClose{Unit: it.b.Unit, OpenID: it.b.OpenID, CloseID: it.cid, BatchHi: batchHi})
		case it.cid > 0:
			kept = append(kept, it.b) // 首闭=会话收口 feed 可见——留 Phase B 行序解
			// （闭标已在库界定窗界 hi=cid——行到即叙事，龄再长不走超时）
			liveRanges = append(liveRanges, narrRange{it.b.OpenID, it.cid})
			cidOf[it.b.Unit] = it.cid
		case time.Since(it.b.OpenTS) > feedNarrTimeout:
			it.timeout = true
			closedNow = append(closedNow, narrRange{it.b.OpenID, wm}) // 兜底段界=已耗水位
		default:
			kept = append(kept, it.b)
			liveRanges = append(liveRanges, narrRange{it.b.OpenID, 0})
		}
	}
	// pending 孤儿（窗态外——理论不可达，存了就结：实证转正/未证续等）
	for _, p := range st.Pending {
		if matched[p.OpenID] {
			continue
		}
		if wm >= p.BatchHi {
			st.Closed = append(st.Closed, feedClosedRange{Unit: p.Unit, OpenID: p.OpenID, CloseID: p.CloseID})
			closedNow = append(closedNow, narrRange{p.OpenID, p.CloseID})
		} else {
			pendNext = append(pendNext, p)
		}
	}
	st.Pending = pendNext
	st.Open = kept
	// 闭段 GC——closeID≤wm=段内行全消费，戒区使命完成
	cl := st.Closed[:0]
	for _, c := range st.Closed {
		if c.CloseID > wm {
			cl = append(cl, c)
		}
	}
	st.Closed = cl
	// 他窗段全集=存活段+本轮新闭段+已闭持久段（self 段按 lo=OpenID 剔——
	// OpenID 唯一锚）。
	persistClosed := make([]narrRange, 0, len(st.Closed))
	for _, c := range st.Closed {
		persistClosed = append(persistClosed, narrRange{c.OpenID, c.CloseID})
	}
	othersOf := func(self feedBracket) []narrRange {
		var rs []narrRange
		for _, r := range liveRanges {
			if r.lo != self.OpenID {
				rs = append(rs, r)
			}
		}
		for _, r := range closedNow {
			if r.lo != self.OpenID {
				rs = append(rs, r)
			}
		}
		return append(rs, persistClosed...)
	}
	// 渲染段——分类已定，逐件出块/兜底
	for i := range items {
		it := &items[i]
		switch {
		case it.emit > 0:
			txt, err := s.narrText(ctx, it.b, it.emit, othersOf(it.b))
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, feedNarrBlock{key: it.emit, text: txt})
		case it.timeout:
			// 超时兜底——窗内已抑行按逐行退不丢行（id≤wm 段；>wm 段本批
			// 行序自然逐行——窗已除不再抑）
			members, err := s.narrBracketRows(ctx, it.b, 0, wm, othersOf(it.b))
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				blocks = append(blocks, feedNarrBlock{key: 0, text: feedRender(m, s.ticketSubjFn(ctx))})
			}
		}
	}

	// ── Phase B：pending 行序消化——开窗/闭窗(会话收口)/窗员抑/余逐行 ──
	openNow := map[string]*feedBracket{}
	for i := range st.Open {
		openNow[st.Open[i].Unit] = &st.Open[i]
	}
	inClosed := func(r feedRow) bool { // 闭段窗员抑（叙事已含）
		if !feedNarrAbsorb[r.event] {
			return false
		}
		u := feedUnitOf(r.note)
		for _, c := range st.Closed {
			if r.id <= c.OpenID || r.id > c.CloseID {
				continue
			}
			if u == "" || u == c.Unit {
				return true
			}
		}
		return false
	}
	inOpen := func(r feedRow) *feedBracket { // 开窗窗员——异标/他窗不归
		if !feedNarrAbsorb[r.event] {
			return nil
		}
		u := feedUnitOf(r.note)
		var best *feedBracket
		for _, b := range openNow {
			if r.id <= b.OpenID {
				continue
			}
			if u != "" && u != b.Unit {
				continue // 异 unit 标行归他窗
			}
			if hi := cidOf[b.Unit]; hi > 0 && r.id > hi {
				continue // 已决闭标界外行不归本窗（O-37510——界外成员不吞逐行）
			}
			if best == nil || b.OpenID > best.OpenID { // 无标行归最近开窗（内层）
				best = b
			}
		}
		return best
	}
	// Phase B 他窗段=持久闭段+他开窗（hi=cidOf 已知闭标界/0 未闭）+本批已决段
	pbClosed := append([]narrRange{}, persistClosed...)
	othersB := func(self feedBracket) []narrRange {
		var rs []narrRange
		for u, b := range openNow {
			if b.OpenID == self.OpenID {
				continue
			}
			rs = append(rs, narrRange{b.OpenID, cidOf[u]})
		}
		for _, r := range pbClosed {
			if r.lo != self.OpenID {
				rs = append(rs, r)
			}
		}
		return rs
	}
	for _, r := range rows {
		u := feedUnitOf(r.note)
		switch {
		case inClosed(r):
			continue // 闭段窗员——叙事已含不二泄
		case r.event == "会话收口" && u != "" && openNow[u] != nil && (cidOf[u] == 0 || r.id <= cidOf[u]):
			// 会话收口闭合——行位解叙事（闭标行本身入窗员集）；cidOf[u]>0
			// 且本行在界外=前序闭标（唤起果/败 pending）已决——不重闭（O-37510）
			b := *openNow[u]
			txt, err := s.narrTextRows(ctx, b, r, othersB(b))
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, feedNarrBlock{key: r.id, text: txt})
			pbClosed = append(pbClosed, narrRange{b.OpenID, r.id})
			delete(openNow, u)
		case r.event == "在办" && u != "" && openNow[u] == nil:
			// 新窗开启——开口行入窗员集（叙事含「醒了」）
			nb := feedBracket{Unit: u, OpenID: r.id, OpenTS: r.ts}
			openNow[u] = &nb
			st.Open = append(st.Open, nb)
			continue // 开口行不逐行——归叙事
		case inOpen(r) != nil:
			continue // 窗员抑——待闭合叙事/超时兜底
		default:
			blocks = append(blocks, feedNarrBlock{key: r.id, text: feedRender(r, s.ticketSubjFn(ctx))})
		}
	}
	// openNow 指针与 st.Open 元素需同步——重建 st.Open 保序（Phase A 存活
	// +Phase B 新开皆在；已闭 delete 者不入）。
	fin := make([]feedBracket, 0, len(st.Open))
	for _, b := range st.Open {
		if p, ok := openNow[b.Unit]; ok && p.OpenID == b.OpenID {
			fin = append(fin, *p)
		}
	}
	st.Open = fin

	if err := s.saveNarrState(ctx, st); err != nil {
		return nil, err // 发前账——存败不发（下拍全量重试不丢行）
	}
	sort.SliceStable(blocks, func(i, j int) bool { return blocks[i].key < blocks[j].key })
	return blocks, nil
}

// ticketSubjFn 票题解析闭包（feedRender 注题用——ctx 绑定版）。
func (s *feedPush) ticketSubjFn(ctx context.Context) func(int64) string {
	return func(id int64) string { return s.ticketSubj(ctx, id) }
}

// narrText 窗叙事成文——闭合段全要素（开行/闭行/窗员）库查拼装。
// cid=闭标行 id（唤起果/唤起败径——会话收口径走 narrTextRows 行内闭行）。
// others=他窗占用段（嵌套窗内层归属——见 narrRange 注）。
func (s *feedPush) narrText(ctx context.Context, b feedBracket, cid int64, others []narrRange) (string, error) {
	open, err := s.narrRowByID(ctx, b.OpenID)
	if err != nil {
		return "", err
	}
	close, err := s.narrRowByID(ctx, cid)
	if err != nil {
		return "", err
	}
	members, err := s.narrBracketRows(ctx, b, cid, 0, others)
	if err != nil {
		return "", err
	}
	return feedNarrCompose(b.Unit, open, close, members), nil
}

// narrTextRows 会话收口行位解——闭行=批内行本体免回取。
func (s *feedPush) narrTextRows(ctx context.Context, b feedBracket, closeRow feedRow, others []narrRange) (string, error) {
	open, err := s.narrRowByID(ctx, b.OpenID)
	if err != nil {
		return "", err
	}
	members, err := s.narrBracketRows(ctx, b, closeRow.id, 0, others)
	if err != nil {
		return "", err
	}
	return feedNarrCompose(b.Unit, open, closeRow, members), nil
}
