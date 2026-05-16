package jwks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"user/internal/mock"
	"user/internal/svc"
	"user/internal/types"
	"user/internal/utils"
)

// TestJwksLogic_Jwks 测试 JWKS 端点
func TestJwksLogic_Jwks(t *testing.T) {
	// 创建测试用的 KeyManager
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 创建 ServiceContext
	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	ctx := context.Background()
	logic := NewJwksLogic(ctx, svcCtx)

	resp, err := logic.Jwks()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 验证返回了密钥
	assert.Len(t, resp.Keys, 1)
	assert.Equal(t, keyManager.GetCurrentKeyID(), resp.Keys[0].Kid)
	assert.Equal(t, "RSA", resp.Keys[0].Kty)
	assert.Equal(t, "RS256", resp.Keys[0].Alg)
	assert.Equal(t, "sig", resp.Keys[0].Use)
	assert.NotEmpty(t, resp.Keys[0].N)
	assert.NotEmpty(t, resp.Keys[0].E)
}

// TestJwksLogic_Jwks_WithPrevious 测试包含 previous 密钥的 JWKS
func TestJwksLogic_Jwks_WithPrevious(t *testing.T) {
	// 创建测试用的 KeyManager
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      48 * time.Hour,
		TokenExpireSecs:  3600,
	}

	keyManager, err := utils.NewRSAKeyManagerWithConfig(cfg)
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 执行密钥轮换
	ctx := context.Background()
	err = keyManager.Rotate(ctx)
	require.NoError(t, err)

	// 创建 ServiceContext
	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	logic := NewJwksLogic(ctx, svcCtx)

	resp, err := logic.Jwks()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 验证返回了两个密钥（current + previous）
	assert.Len(t, resp.Keys, 2)

	// 验证第一个是当前密钥
	assert.Equal(t, keyManager.GetCurrentKeyID(), resp.Keys[0].Kid)

	// 验证第二个是 previous 密钥
	keyInfo := keyManager.GetKeyInfo()
	previousKeyID := keyInfo["previous_key_id"]
	if previousKeyID != nil {
		assert.Equal(t, previousKeyID.(string), resp.Keys[1].Kid)
	}
}

// TestJwksLogic_Jwks_Empty 测试没有密钥的情况
func TestJwksLogic_Jwks_Empty(t *testing.T) {
	// 创建一个空的 KeyManager（手动创建，不生成密钥）
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      48 * time.Hour,
		TokenExpireSecs:  3600,
	}

	keyManager, err := utils.NewRSAKeyManagerWithConfig(cfg)
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 销毁密钥，使其为空
	keyManager.Destroy()

	// 创建 ServiceContext
	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	ctx := context.Background()
	logic := NewJwksLogic(ctx, svcCtx)

	resp, err := logic.Jwks()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// 验证返回空密钥列表
	assert.Len(t, resp.Keys, 0)
}

// TestJwksLogic_Jwks_JWKFields 测试 JWK 字段格式
func TestJwksLogic_Jwks_JWKFields(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	ctx := context.Background()
	logic := NewJwksLogic(ctx, svcCtx)

	resp, err := logic.Jwks()
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Keys, 1)

	jwk := resp.Keys[0]

	// 验证所有必需字段
	assert.NotEmpty(t, jwk.Kty, "kty should not be empty")
	assert.NotEmpty(t, jwk.Kid, "kid should not be empty")
	assert.NotEmpty(t, jwk.Use, "use should not be empty")
	assert.NotEmpty(t, jwk.N, "n should not be empty")
	assert.NotEmpty(t, jwk.E, "e should not be empty")
	assert.NotEmpty(t, jwk.Alg, "alg should not be empty")

	// 验证字段值
	assert.Equal(t, "RSA", jwk.Kty)
	assert.Equal(t, "sig", jwk.Use)
	assert.Equal(t, "RS256", jwk.Alg)
}

// TestJwksLogic_Jwks_MultipleCalls 测试多次调用
func TestJwksLogic_Jwks_MultipleCalls(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	ctx := context.Background()
	logic := NewJwksLogic(ctx, svcCtx)

	// 第一次调用
	resp1, err := logic.Jwks()
	require.NoError(t, err)

	// 第二次调用
	resp2, err := logic.Jwks()
	require.NoError(t, err)

	// 验证两次返回相同的结果
	assert.Equal(t, len(resp1.Keys), len(resp2.Keys))
	if len(resp1.Keys) > 0 && len(resp2.Keys) > 0 {
		assert.Equal(t, resp1.Keys[0].Kid, resp2.Keys[0].Kid)
	}
}

// TestJwksLogic_Jwks_AfterRotation 测试密钥轮换后的 JWKS
func TestJwksLogic_Jwks_AfterRotation(t *testing.T) {
	cfg := &utils.KeyManagerConfig{
		Bits:             2048,
		RotationInterval: 24 * time.Hour,
		GracePeriod:      48 * time.Hour,
		TokenExpireSecs:  3600,
	}

	keyManager, err := utils.NewRSAKeyManagerWithConfig(cfg)
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 获取轮换前的密钥 ID
	oldKeyID := keyManager.GetCurrentKeyID()

	svcCtx := &svc.ServiceContext{
		KeyManager: keyManager,
	}

	ctx := context.Background()
	logic := NewJwksLogic(ctx, svcCtx)

	// 轮换前
	resp1, err := logic.Jwks()
	require.NoError(t, err)
	assert.Len(t, resp1.Keys, 1)
	assert.Equal(t, oldKeyID, resp1.Keys[0].Kid)

	// 执行轮换
	err = keyManager.Rotate(ctx)
	require.NoError(t, err)

	newKeyID := keyManager.GetCurrentKeyID()
	assert.NotEqual(t, oldKeyID, newKeyID)

	// 轮换后
	resp2, err := logic.Jwks()
	require.NoError(t, err)
	assert.Len(t, resp2.Keys, 2)
	assert.Equal(t, newKeyID, resp2.Keys[0].Kid)
	assert.Equal(t, oldKeyID, resp2.Keys[1].Kid)
}

// TestConvertJWK 测试 JWK 转换（辅助测试）
func TestConvertJWK(t *testing.T) {
	keyManager, err := mock.NewTestKeyManager()
	require.NoError(t, err)
	defer keyManager.Destroy()

	// 获取内部 JWKS
	internalJWKS := keyManager.ToJWKS()
	require.Len(t, internalJWKS.Keys, 1)

	internalJWK := internalJWKS.Keys[0]

	// 转换为 types.JWK
	convertedJWK := types.JWK{
		Kty: internalJWK.Kty,
		Kid: internalJWK.Kid,
		Use: internalJWK.Use,
		N:   internalJWK.N,
		E:   internalJWK.E,
		Alg: internalJWK.Alg,
	}

	// 验证字段一致
	assert.Equal(t, internalJWK.Kty, convertedJWK.Kty)
	assert.Equal(t, internalJWK.Kid, convertedJWK.Kid)
	assert.Equal(t, internalJWK.Use, convertedJWK.Use)
	assert.Equal(t, internalJWK.N, convertedJWK.N)
	assert.Equal(t, internalJWK.E, convertedJWK.E)
	assert.Equal(t, internalJWK.Alg, convertedJWK.Alg)
}
