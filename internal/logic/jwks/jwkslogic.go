// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package jwks

import (
	"context"

	"user/internal/svc"
	"user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type JwksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewJwksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *JwksLogic {
	return &JwksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *JwksLogic) Jwks() (resp *types.JWKSResp, err error) {
	// 从 KeyManager 获取 JWKS（包含 current + previous 两个公钥）
	jwks := l.svcCtx.KeyManager.ToJWKS()

	// 转换为 types.JWKSResp
	var jwksKeys []types.JWK
	for _, key := range jwks.Keys {
		jwksKeys = append(jwksKeys, types.JWK{
			Kty: key.Kty,
			Kid: key.Kid,
			Use: key.Use,
			N:   key.N,
			E:   key.E,
			Alg: key.Alg,
		})
	}

	return &types.JWKSResp{
		Keys: jwksKeys,
	}, nil
}
