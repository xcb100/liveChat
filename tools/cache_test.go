package tools

import (
	"testing"
	"time"
)

// TestNewMultiLevelCacheFallsBackToMemory 输入测试上下文，输出为降级断言结果，目的在于验证 Redis 缺失时多级缓存会退化到内存实现。
func TestNewMultiLevelCacheFallsBackToMemory(t *testing.T) {
	cache := NewMultiLevelCache(nil, 50*time.Millisecond, 50*time.Millisecond)
	if cache == nil {
		t.Fatalf("期望得到非空缓存实例")
	}
	if !cache.useMemoryCache {
		t.Fatalf("期望 Redis 不可用时自动启用内存缓存")
	}

	if setError := cache.Set(Ctx, "reply:test", "value", time.Minute); setError != nil {
		t.Fatalf("写入内存缓存失败: %v", setError)
	}
	value, getError := cache.Get(Ctx, "reply:test")
	if getError != nil {
		t.Fatalf("读取内存缓存失败: %v", getError)
	}
	if value != "value" {
		t.Fatalf("期望读取到 value，实际得到 %s", value)
	}
}
