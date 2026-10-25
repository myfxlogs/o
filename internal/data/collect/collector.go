// Package collect —— 采集器接口件（T-0108 nox 移植最小面：Poll 统一签名）。
package collect

import (
	"context"

	"o/internal/data/fact"
)

// Collector 采集器（Poll=单次真读——节拍由装配层管）。
type Collector interface {
	Kind() string
	Poll(ctx context.Context) ([]fact.Fact, error)
}
