package middleware

import (
	"sync"
	"time"
)

// permCacheEntry 缓存条目
type permCacheEntry struct {
	perms     map[string]bool
	expiresAt time.Time
}

const permCacheTTL = 5 * time.Minute

var (
	permCache sync.Map // map[uint]*permCacheEntry
)

// GetCachedPermissions 获取用户缓存权限集合，未命中返回 nil
func GetCachedPermissions(userID uint) map[string]bool {
	val, ok := permCache.Load(userID)
	if !ok {
		return nil
	}
	entry := val.(*permCacheEntry)
	if time.Now().After(entry.expiresAt) {
		permCache.Delete(userID)
		return nil
	}
	return entry.perms
}

// SetCachedPermissions 设置用户权限缓存
func SetCachedPermissions(userID uint, perms map[string]bool) {
	permCache.Store(userID, &permCacheEntry{
		perms:     perms,
		expiresAt: time.Now().Add(permCacheTTL),
	})
}

// InvalidateUserPermCache 清除指定用户缓存
func InvalidateUserPermCache(userID uint) {
	permCache.Delete(userID)
}

// InvalidateAllPermCache 清除所有用户权限缓存（角色/菜单变更时调用）
func InvalidateAllPermCache() {
	permCache.Range(func(key, _ interface{}) bool {
		permCache.Delete(key)
		return true
	})
}
