// Package httpread —— 外呼 HTTP 响应读面共享件（T-0440③b）：容量帽+截断
// 显式错一体。病灶原型：io.ReadAll(io.LimitReader(r, N)) 截断静默——读满
// 帽不报错，下游 JSON parse 以「unexpected end of JSON input」伪装成解析
// 病而非容量病（T-0440 实证：OKX SPOT instruments 1.31MB>1MB 帽→SPOT 族
// fail-closed 每拍让位）。新 venue 客户端消费本件继承同构防御——禁各抄
// LimitReader（抄=病跟着抄）。
package httpread

import (
	"errors"
	"fmt"
	"io"
)

// PublicCap 公开行情读面容量帽（T-0440①钦定 8MB 起步）。实证锚：OKX
// /api/v5/public/instruments?instType=SPOT 2026-11-07 实测 1,306,838B≈1.31MB
// 且增长中（旧 1MB 帽必截断）；帽保留防失控体不写死无限。
const PublicCap int64 = 1 << 23

// ErrOverCap 响应超帽（errors.Is 判位——T-0440②截断显式化：帽处显式
// 「响应超帽」错，替代「读满帽静默→下游 JSON 语法错」伪装形态）。
var ErrOverCap = errors.New("响应超帽")

// Body 读全响应体（帽+截断显式错一体）：LimitReader(cap+1) 多读一字节探测
// 截断——len>cap 即 ErrOverCap 显式错；帽内原样返回零改写。
func Body(r io.Reader, capBytes int64) ([]byte, error) {
	if capBytes < 1 {
		return nil, fmt.Errorf("httpread: 帽非法 %d（<1）", capBytes)
	}
	b, err := io.ReadAll(io.LimitReader(r, capBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > capBytes {
		return nil, fmt.Errorf("%w：读得 %dB>帽 %dB——响应截断于帽位（显式错非 JSON 语法错伪装）",
			ErrOverCap, len(b), capBytes)
	}
	return b, nil
}
