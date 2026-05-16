package utils

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"user/internal/logger"
)

// 错误定义
var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrInvalidConfig    = errors.New("invalid key manager config")
	ErrRotationFailed   = errors.New("key rotation failed")
	ErrKeyAlreadyExists = errors.New("key rotation already running")
)

// KeyManagerConfig RSA 密钥管理器配置
type KeyManagerConfig struct {
	Bits             int           // 密钥位数
	RotationInterval time.Duration // 轮换周期
	GracePeriod      time.Duration // 旧密钥保留期
	TokenExpireSecs  int64         // token 有效期
}

// Validate 验证配置有效性
func (cfg *KeyManagerConfig) Validate() error {
	if cfg.Bits < 2048 {
		return fmt.Errorf("%w: bits must >= 2048", ErrInvalidConfig)
	}
	if cfg.RotationInterval < time.Hour {
		return fmt.Errorf("%w: rotation_interval must >= 1h", ErrInvalidConfig)
	}
	if cfg.GracePeriod < time.Duration(cfg.TokenExpireSecs)*time.Second {
		return fmt.Errorf("%w: grace_period (%v) must >= token_expire_secs (%v)",
			ErrInvalidConfig, cfg.GracePeriod, time.Duration(cfg.TokenExpireSecs)*time.Second)
	}
	return nil
}

// KeyInfo 密钥信息，包含过期时间
type KeyInfo struct {
	KeyPair    *RSAKeyPair
	ExpireTime time.Time // 密钥过期时间，超过此时间后密钥不再有效
}

// RSAKeyManager RSA 密钥管理器，支持密钥轮换
type RSAKeyManager struct {
	current   *KeyInfo      // 当前使用的密钥
	previous  *KeyInfo      // 上一个密钥（带过期时间）
	mu        sync.RWMutex  // 读写锁
	stopChan  chan struct{} // 停止信号通道
	isRunning bool          // 标记自动轮换是否正在运行
	config    *KeyManagerConfig
}

// NewRSAKeyManagerWithConfig 使用配置创建 RSA 密钥管理器
func NewRSAKeyManagerWithConfig(cfg *KeyManagerConfig) (*RSAKeyManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 生成初始密钥对
	keyPair, err := GenerateRSAKeyPair(cfg.Bits)
	if err != nil {
		return nil, fmt.Errorf("generate initial key pair failed: %w", err)
	}

	return &RSAKeyManager{
		current: &KeyInfo{
			KeyPair:    keyPair,
			ExpireTime: time.Now().Add(time.Duration(cfg.TokenExpireSecs) * time.Second),
		},
		previous:  nil,
		stopChan:  make(chan struct{}),
		isRunning: false,
		config:    cfg,
	}, nil
}

// NewRSAKeyManagerFromPEM 从 PEM 格式私钥创建密钥管理器
func NewRSAKeyManagerFromPEM(privateKeyPEM, keyID string, cfg *KeyManagerConfig) (*RSAKeyManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	privateKey, err := ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key from PEM failed: %w", err)
	}

	keyPair := &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      keyID,
	}

	return &RSAKeyManager{
		current: &KeyInfo{
			KeyPair:    keyPair,
			ExpireTime: time.Now().Add(time.Duration(cfg.TokenExpireSecs) * time.Second),
		},
		previous:  nil,
		stopChan:  make(chan struct{}),
		isRunning: false,
		config:    cfg,
	}, nil
}

// Rotate 执行密钥轮换
// current → previous，生成新的 current
func (m *RSAKeyManager) Rotate(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成新的密钥对
	newKeyPair, err := GenerateRSAKeyPair(m.config.Bits)
	if err != nil {
		logger.L(ctx, "密钥轮换失败").WErrorMsg(err.Error()).Errors()
		return fmt.Errorf("%w: %v", ErrRotationFailed, err)
	}

	oldKeyID := ""
	if m.current != nil {
		oldKeyID = m.current.KeyPair.KeyID
	}

	// 轮换：current 变为 previous（设置过期时间）
	if m.current != nil {
		m.previous = &KeyInfo{
			KeyPair:    m.current.KeyPair,
			ExpireTime: time.Now().Add(m.config.GracePeriod),
		}
	}

	// 新密钥成为 current
	m.current = &KeyInfo{
		KeyPair:    newKeyPair,
		ExpireTime: time.Now().Add(time.Duration(m.config.TokenExpireSecs) * time.Second),
	}

	logger.L(ctx, "密钥轮换成功").
		WOldKeyId(oldKeyID).
		WNewKeyId(newKeyPair.KeyID).
		Infos()

	return nil
}

// GetCurrentPrivateKey 获取当前私钥（用于签发 token）
func (m *RSAKeyManager) GetCurrentPrivateKey() *rsa.PrivateKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return nil
	}
	return m.current.KeyPair.PrivateKey
}

// GetCurrentKeyID 获取当前密钥 ID
func (m *RSAKeyManager) GetCurrentKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return ""
	}
	return m.current.KeyPair.KeyID
}

// GetConfig 获取密钥管理器配置
func (m *RSAKeyManager) GetConfig() *KeyManagerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetPublicKeyByID 根据 KeyID 获取公钥（用于验证 token）
// 支持 current 和 previous 两个密钥
// 如果密钥不存在或已过期，返回错误
func (m *RSAKeyManager) GetPublicKeyByID(keyID string) (*rsa.PublicKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()

	// 先检查 current
	if m.current != nil && m.current.KeyPair.KeyID == keyID {
		if now.After(m.current.ExpireTime) {
			return nil, fmt.Errorf("%w: current key expired", ErrKeyNotFound)
		}
		return m.current.KeyPair.PublicKey, nil
	}

	// 再检查 previous
	if m.previous != nil && m.previous.KeyPair.KeyID == keyID {
		if now.After(m.previous.ExpireTime) {
			return nil, fmt.Errorf("%w: previous key expired", ErrKeyNotFound)
		}
		return m.previous.KeyPair.PublicKey, nil
	}

	return nil, ErrKeyNotFound
}

// ToJWKS 生成 JWKS，包含 current 和 previous 两个公钥（只返回未过期的）
func (m *RSAKeyManager) ToJWKS() JWKS {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []JWK
	now := time.Now()

	// 添加 current 密钥（优先）
	if m.current != nil && now.Before(m.current.ExpireTime) {
		keys = append(keys, m.current.KeyPair.ToJWK())
	}

	// 添加 previous 密钥（如果未过期）
	if m.previous != nil && now.Before(m.previous.ExpireTime) {
		keys = append(keys, m.previous.KeyPair.ToJWK())
	}

	return JWKS{Keys: keys}
}

// ToJWKSJSON 返回 JWKS 的 JSON 字符串
func (m *RSAKeyManager) ToJWKSJSON() (string, error) {
	jwks := m.ToJWKS()
	return jwks.ToJSON()
}

// GetKeyInfo 获取当前密钥信息（用于日志或监控）
func (m *RSAKeyManager) GetKeyInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	info := map[string]interface{}{
		"current_key_id":   nil,
		"previous_key_id":  nil,
		"has_current":      m.current != nil,
		"has_previous":     m.previous != nil,
		"current_expired":  false,
		"previous_expired": false,
	}

	if m.current != nil {
		info["current_key_id"] = m.current.KeyPair.KeyID
		info["current_expired"] = now.After(m.current.ExpireTime)
		info["current_expire_time"] = m.current.ExpireTime
	}
	if m.previous != nil {
		info["previous_key_id"] = m.previous.KeyPair.KeyID
		info["previous_expired"] = now.After(m.previous.ExpireTime)
		info["previous_expire_time"] = m.previous.ExpireTime
	}

	return info
}

// AutoRotate 自动密钥轮换（可配合定时任务使用）
// 返回停止函数，调用可停止自动轮换
func (m *RSAKeyManager) AutoRotate(ctx context.Context) (func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已经在运行，返回错误
	if m.isRunning {
		return nil, ErrKeyAlreadyExists
	}

	m.isRunning = true
	// 创建新的停止通道（确保之前的通道已被关闭）
	m.stopChan = make(chan struct{})

	// 启动定时轮换
	ticker := time.NewTicker(m.config.RotationInterval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 执行轮换
				if err := m.Rotate(ctx); err != nil {
					logger.L(ctx, "自动密钥轮换失败").WErrorMsg(err.Error()).Errors()
					continue
				}

				// 延迟清理旧密钥（gracePeriod 后）
				go func() {
					time.Sleep(m.config.GracePeriod)
					m.clearPreviousIfNotChanged()
				}()
			case <-m.stopChan:
				return
			}
		}
	}()

	logger.L(ctx, "自动密钥轮换已启动").
		WRotationInterval(m.config.RotationInterval.String()).
		WGracePeriod(m.config.GracePeriod.String()).
		Infos()

	// 返回停止函数
	return func() {
		m.mu.Lock()
		if m.isRunning {
			close(m.stopChan)
			m.isRunning = false
		}
		m.mu.Unlock()
		logger.L(ctx, "自动密钥轮换已停止").Infos()
	}, nil
}

// clearPreviousIfNotChanged 安全清理上一个密钥
// 只有当 previous 没有被更新时才清理（避免误删新密钥）
func (m *RSAKeyManager) clearPreviousIfNotChanged() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查 previous 是否已过期
	if m.previous != nil && time.Now().After(m.previous.ExpireTime) {
		m.previous = nil
	}
}

// Destroy 安全销毁密钥（内存清理）
// 会停止自动轮换 goroutine
func (m *RSAKeyManager) Destroy() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止自动轮换
	if m.isRunning {
		close(m.stopChan)
		m.isRunning = false
	}

	// 清理密钥
	m.current = nil
	m.previous = nil
}
