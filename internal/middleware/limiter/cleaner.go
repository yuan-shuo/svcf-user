// 缓存清理器
package limiter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CacheEntry 带时间戳的缓存条目
type CacheEntry struct {
	Value      interface{}
	lastAccess int64 // 使用原子操作存储时间戳
	// done 用于通知等待的协程创建完成
	done chan struct{}
	// err 存储创建过程中的错误
	err error
}

// LoadLastAccess 原子加载最后访问时间
func (e *CacheEntry) LoadLastAccess() time.Time {
	unix := atomic.LoadInt64(&e.lastAccess)
	return time.Unix(0, unix)
}

// StoreLastAccess 原子存储最后访问时间
func (e *CacheEntry) StoreLastAccess(t time.Time) {
	atomic.StoreInt64(&e.lastAccess, t.UnixNano())
}

// IsReady 检查条目是否已准备好（值已设置或出错）
func (e *CacheEntry) IsReady() bool {
	return e.Value != nil || e.err != nil
}

// WaitForReady 等待条目准备就绪，使用 context 避免内存泄漏
func (e *CacheEntry) WaitForReady(ctx context.Context) bool {
	select {
	case <-e.done:
		return true
	case <-ctx.Done():
		return false
	}
}

// CleanableCache 可清理的缓存
type CleanableCache struct {
	cache    sync.Map
	ttl      time.Duration
	stopChan chan struct{}
	once     sync.Once
	stopOnce sync.Once
}

func NewCleanableCache(ttl time.Duration) *CleanableCache {
	return &CleanableCache{
		ttl:      ttl,
		stopChan: make(chan struct{}),
	}
}

func (c *CleanableCache) Load(key string) (interface{}, bool) {
	val, ok := c.cache.Load(key)
	if !ok {
		return nil, false
	}
	entry := val.(*CacheEntry)
	entry.StoreLastAccess(time.Now())
	return entry.Value, true
}

func (c *CleanableCache) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	// 启动清理协程（只启动一次）
	c.once.Do(func() {
		go c.cleanupLoop()
	})

	entry := &CacheEntry{
		Value:      value,
		lastAccess: time.Now().UnixNano(),
		done:       make(chan struct{}),
	}
	close(entry.done) // 立即关闭，表示已就绪

	actual, loaded := c.cache.LoadOrStore(key, entry)
	if loaded {
		actualEntry := actual.(*CacheEntry)
		actualEntry.StoreLastAccess(time.Now())
		return actualEntry.Value, true
	}
	return value, false
}

// ComputeIfAbsent 如果不存在则计算并存储，确保只创建一次
// 使用 sync.Map 的 LoadOrStore 原子操作避免竞态条件
// 使用 channel 通知机制避免忙等待
// 使用 defer/recover 防止 panic 导致死锁
func (c *CleanableCache) ComputeIfAbsent(key string, compute func() interface{}) (interface{}, error) {
	// 启动清理协程（只启动一次）
	c.once.Do(func() {
		go c.cleanupLoop()
	})

	for {
		// 先尝试加载已存在的条目
		if val, ok := c.cache.Load(key); ok {
			entry := val.(*CacheEntry)
			if entry.IsReady() {
				entry.StoreLastAccess(time.Now())
				return entry.Value, entry.err
			}
			// 条目正在创建中，等待完成（使用 context 避免内存泄漏）
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			ready := entry.WaitForReady(ctx)
			cancel()
			if ready {
				entry.StoreLastAccess(time.Now())
				return entry.Value, entry.err
			}
			// 超时，继续循环重试
			continue
		}

		// 尝试创建新条目
		placeholder := &CacheEntry{
			Value:      nil,
			lastAccess: time.Now().UnixNano(),
			done:       make(chan struct{}),
		}

		_, loaded := c.cache.LoadOrStore(key, placeholder)
		if loaded {
			// 其他协程已创建，继续循环等待
			continue
		}

		// 我们获得了创建权，执行创建（带 panic 保护）
		func() {
			defer func() {
				if r := recover(); r != nil {
					placeholder.err = &computePanicError{recovered: r}
				}
				// 无论成功还是失败，都关闭 done channel 通知等待的协程
				close(placeholder.done)
			}()
			placeholder.Value = compute()
		}()

		return placeholder.Value, placeholder.err
	}
}

// computePanicError 表示 compute 函数 panic 的错误
type computePanicError struct {
	recovered interface{}
}

func (e *computePanicError) Error() string {
	return "compute function panicked"
}

func (c *CleanableCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2) // 每半个 TTL 清理一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

func (c *CleanableCache) cleanup() {
	now := time.Now()
	c.cache.Range(func(key, value interface{}) bool {
		entry := value.(*CacheEntry)
		// 只清理已就绪的条目，避免删除正在创建的条目
		if entry.IsReady() && now.Sub(entry.LoadLastAccess()) > c.ttl {
			c.cache.Delete(key)
		}
		return true
	})
}

func (c *CleanableCache) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
}
