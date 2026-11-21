package main

// venue_dead.go —— venue 结构死分类哨兵（工单#26999-root：mt5Live
// OLD_VERSION vendor 死 19h+→Money 读件缺行→navBy 数 miss 拒写
// md_equity_curve——闸行为正确缺口=不分类暂态读败 vs 结构死 venue）。
// VenueDead=venue 侧结构死（协议版本拒/端点断族），暂态读败（超时/限流/
// 断连）非同类——NAV 闸对死腿剔除照写、对暂态腿照 miss 照挡。

import (
	"errors"
	"fmt"
	"strings"
)

// ErrVenueDead venue 结构死哨兵（errors.Is 判定——包装链保留原文可溯）。
var ErrVenueDead = errors.New("venue dead（结构死 venue——暂态读败非同类）")

// venueDeadMark MoneyRow.Note 死腿标记词（navBy/money_card 同源字面量
// 单点——UI 读件缺行显形不动，死因注记随行可见）。
const venueDeadMark = "venue死"

// IsVenueDead 判定 err 链是否结构死 venue（哨兵优先；旧径裸错兜底——
// OLD_VERSION 串=mtapi ErrorCode 1010 vendor 死签名，生产实证在案）。
func IsVenueDead(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrVenueDead) {
		return true
	}
	return strings.Contains(strings.ToUpper(err.Error()), "OLD_VERSION")
}

// venueDeadWrap err→ErrVenueDead 包装（原文留 %v 可溯——判词不丢因）。
func venueDeadWrap(err error) error {
	return fmt.Errorf("%w: %v", ErrVenueDead, err)
}
