package utils

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

func Ptr[T any](v T) *T {
	return &v
}

func Panicf(format string, args ...any) {
	panic(fmt.Errorf(format, args...))
}

func SortedKeys[T any](m map[string]T) []string {
	var keys []string

	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SyncMap is a [sync.Map], just with typesafe wrapper functions.
type SyncMap[KeyT comparable, ValueT any] struct {
	m    sync.Map
	zero ValueT
}

func (sm *SyncMap[KeyT, ValueT]) Store(key KeyT, value ValueT) {
	sm.m.Store(key, value)
}

func (sm *SyncMap[KeyT, ValueT]) Delete(key KeyT) {
	sm.m.Delete(key)
}

func (sm *SyncMap[KeyT, ValueT]) Load(key KeyT) ValueT {
	v, ok := sm.m.Load(key)

	if !ok {
		return sm.zero
	}

	return v.(ValueT)
}

func Sleep(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

func HostOnly(endpoint string) string {
	return strings.Split(endpoint, ":")[0]
}

func CloseWithLogging[T interface{ Close() error }](label string, closeable T) {
	if err := closeable.Close(); err != nil {
		slog.Warn("closeWithLogging", "type", fmt.Sprintf("%T", closeable), "err", err, "label", label)
	}
}
