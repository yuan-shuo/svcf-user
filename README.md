## redis 键命名

```go
// account:register:verify:3695@qq.com - 服务:子功能:功能类型:参数
// account:register:limit:3695@qq.com
baseKey := fmt.Sprintf("%s:%s", l.svcCtx.Config.Register.SendCodeConfig.RedisKeyPrefix, l.svcCtx.Config.Register.SendCodeConfig.ReceiveType)
// redis缓存验证码数据
redisKey := fmt.Sprintf("%s:verify:%s", baseKey, req.Email)
// 设置限流验证码键
limitKey := fmt.Sprintf("%s:limit:%s", baseKey, req.Email)
```