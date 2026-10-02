package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const runtimePriceVersionTTL = 30 * time.Second

type runtimePriceVersionSnapshot struct {
	version  string
	loadedAt time.Time
}

var runtimePriceVersionCache sync.Map

// RuntimeConfigVersion 在请求开始时固化配置和价格版本。
// 这里只保存内容指纹，不读取请求体，也不会把设置值或价格表写入日志。
func RuntimeConfigVersion(settingService *service.SettingService, pricingFiles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()
		if settingService != nil {
			version := settingService.RuntimeSettingsVersion(ctx).Version
			ctx = context.WithValue(ctx, ctxkey.RuntimeConfigVersion, version)
		}
		ctx = context.WithValue(ctx, ctxkey.RuntimePriceVersion, resolveRuntimePriceVersion(pricingFiles))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// resolveRuntimePriceVersion 对主价格目录、回退文件和覆盖文件生成联合指纹。
// 结果短暂缓存，避免每个网关请求都读取磁盘。
func resolveRuntimePriceVersion(pricingFiles []string) string {
	cacheKey := strings.Join(pricingFiles, "\x00")
	if cached, ok := runtimePriceVersionCache.Load(cacheKey); ok {
		snapshot := cached.(runtimePriceVersionSnapshot)
		if time.Since(snapshot.loadedAt) < runtimePriceVersionTTL {
			return snapshot.version
		}
	}

	version := computeRuntimePriceVersion(pricingFiles)
	runtimePriceVersionCache.Store(cacheKey, runtimePriceVersionSnapshot{version: version, loadedAt: time.Now()})
	return version
}

func computeRuntimePriceVersion(pricingFiles []string) string {
	if len(pricingFiles) == 0 {
		return "fallback"
	}

	hash := sha256.New()
	found := false
	for index, configuredPath := range pricingFiles {
		configuredPath = strings.TrimSpace(configuredPath)
		if configuredPath == "" {
			continue
		}
		paths := []string{configuredPath}
		if index == 0 {
			paths = []string{
				filepath.Join(configuredPath, "model_pricing.sha256"),
				filepath.Join(configuredPath, "model_pricing.json"),
			}
		}
		for fileIndex, path := range paths {
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			found = true
			// 用文件角色而不是绝对路径参与哈希，保证不同部署目录的实例可比较。
			_, _ = hash.Write([]byte{byte(index), byte(fileIndex)})
			_, _ = hash.Write([]byte{0})
			_, _ = hash.Write(content)
			_, _ = hash.Write([]byte{0})
			// 主目录存在 sha256 文件时，它已经代表价格目录内容，无需再读取大 JSON 文件。
			if index == 0 && fileIndex == 0 {
				break
			}
		}
	}
	if !found {
		return "fallback"
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
