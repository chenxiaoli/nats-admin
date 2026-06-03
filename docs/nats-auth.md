# NATS Auth Model

## 信任链
```
Operator NKey (SO...) → signs → Account JWT  ← 租户身份
Account NKey  (SA...) → signs → User JWT     ← 客户端凭证
User NKey     (SU...) → embedded in .creds
```

## NKey 类型速查

| 类型     | Public Key 前缀 | Seed 前缀 | 用途             |
|----------|----------------|-----------|------------------|
| Operator | `O`            | `SO`      | 签发 Account JWT |
| Account  | `A`            | `SA`      | 签发 User JWT    |
| User     | `U`            | `SU`      | 嵌入 .creds 文件 |

## NATS Server 配置（resolver 模式）

```hcl
operator: "/etc/nats/operator.jwt"
resolver: {
  type:         full
  dir:          "/data/nats/resolver"
  allow_delete: true
  interval:     "2m"
}
system_account: "AXXXXXXXXXX"   # System Account public key
```

## Account JWT 关键 Claims

```json
{
  "limits": { "subs": -1, "conn": -1, "payload": -1 },
  "jetstream": {
    "memory_storage": 10737418240,
    "storage":        107374182400,
    "streams":        100,
    "consumer":       1000,
    "max_bytes_required": false
  }
}
```

## 一次性 Operator Bootstrap（cmd/bootstrap）

```go
okp, _  := nkeys.CreateOperator()
seed, _ := okp.Seed()
pub, _  := okp.PublicKey()

sakp, _  := nkeys.CreateAccount()   // System Account
saPub, _ := sakp.PublicKey()
saSeed, _ := sakp.Seed()

oc := jwt.NewOperatorClaims(pub)
oc.Name = "platform-operator"
oc.SystemAccount = saPub
operatorJWT, _ := oc.Encode(okp)

sac := jwt.NewAccountClaims(saPub)
sac.Name = "SYS"
systemJWT, _ := sac.Encode(okp)

// stdout → 手动写入 .env
// operatorJWT → /etc/nats/operator.jwt
// systemJWT   → push 到 resolver（启动后）
```

## 创建租户（Account）

```go
akp, _      := nkeys.CreateAccount()
pubKey, _   := akp.PublicKey()
seed, _     := akp.Seed()

claims := jwt.NewAccountClaims(pubKey)
claims.Name = req.Name
claims.Limits.JetStreamLimits = jwt.JetStreamLimits{
    MemoryStorage: req.JSMemoryStorage,
    DiskStorage:   req.JSDiskStorage,
    Streams:       int64(req.MaxStreams),
    Consumer:      int64(req.MaxConsumers),
}
accountJWT, _ := claims.Encode(operatorKP)  // Operator 签名
```

## 签发用户凭证

```go
// 1. 解密 Account seed，重建 Account KP
akp, _ := nkeys.FromSeed(decryptedSeed)

// 2. 生成 User NKey
ukp, _    := nkeys.CreateUser()
userPub, _ := ukp.PublicKey()
userSeed, _ := ukp.Seed()

// 3. User JWT 由 Account 签
claims := jwt.NewUserClaims(userPub)
claims.Name = req.Name
claims.IssuerAccount = accountPub
claims.Permissions.Pub.Allow = req.PubAllow  // subject 白名单
userJWT, _ := claims.Encode(akp)

// 4. .creds 格式
creds := fmt.Sprintf(
    "-----BEGIN NATS USER JWT-----\n%s\n------END NATS USER JWT------\n\n"+
    "-----BEGIN USER NKEY SEED-----\n%s\n------END USER NKEY SEED------\n",
    userJWT, userSeed,
)
```

## Push Resolver（Account JWT 变更都要调用）

```go
// $SYS.REQ.CLAIMS.UPDATE — push
msg, err := sysConn.RequestWithContext(ctx, "$SYS.REQ.CLAIMS.UPDATE", []byte(accountJWT))
var resp struct{ Error string `json:"error"` }
json.Unmarshal(msg.Data, &resp)
// resp.Error != "" → push 被拒绝

// $SYS.REQ.CLAIMS.DELETE — 删除账号（立即断开所有连接）
sysConn.RequestWithContext(ctx, "$SYS.REQ.CLAIMS.DELETE", []byte(accountPubKey))
```

## 吊销 User（重签 Account JWT）

```go
claims, _ := jwt.DecodeAccountClaims(currentAccountJWT)
if claims.Revocations == nil {
    claims.Revocations = make(jwt.RevocationList)
}
claims.Revocations.Revoke(userPublicKey, time.Now())
newJWT, _ := claims.Encode(operatorKP)   // 重签
resolver.Push(ctx, newJWT)               // push 到 NATS
db.UpdateAccountJWT(ctx, tenantID, newJWT)
```

## AES-256-GCM Seed 加密

```go
func Encrypt(masterKey, plaintext []byte) (ciphertext, nonce []byte) {
    block, _ := aes.NewCipher(masterKey)   // masterKey: 32 bytes
    gcm, _   := cipher.NewGCM(block)
    nonce = make([]byte, gcm.NonceSize())
    io.ReadFull(rand.Reader, nonce)
    ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
    return
}
```

## 监控：System Account 数据来源

```
# 订阅（服务器周期 publish）
$SYS.SERVER.ACCOUNT.*.STATZ        per-account 连接/消息/字节统计
$SYS.ACCOUNT.*.CONNECT / DISCONNECT

# 主动请求
$SYS.REQ.SERVER.PING               → server 列表
$SYS.REQ.SERVER.<sid>.VARZ         → 单台 server 统计
$SYS.REQ.ACCOUNT.<pubkey>.STATZ    → 指定 account 当前快照
```
