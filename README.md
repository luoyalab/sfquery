# 顺丰路由查询服务

基于 Go 标准库封装顺丰丰桥 `EXP_RECE_SEARCH_ROUTES` 路由查询接口，提供 Web 查询页面。

## 项目结构

```
sfquery/
├── cmd/
│   └── server/
│       └── main.go          # 入口：HTTP 服务、静态文件嵌入、优雅关闭
├── internal/
│   ├── sfclient/
│   │   └── client.go        # 顺丰客户端：签名算法、接口调用、结构体定义
│   └── handler/
│       └── handler.go       # HTTP Handler：参数解析、响应组装
├── web/
│   └── static/
│       └── index.html       # 前端查询页面（嵌入到二进制）
├── Dockerfile
├── go.mod
└── README.md
```

## 快速启动

```bash
# 直接运行（使用内置凭据）
go run ./cmd/server

# 自定义凭据
go run ./cmd/server \
  -addr :8080 \
  -partner-id YOUR_PARTNER_ID \
  -check-word YOUR_CHECK_WORD

# 沙箱环境测试
go run ./cmd/server -sandbox

# 环境变量方式（适合生产部署）
export SF_PARTNER_ID=XMWL43MUNA8
export SF_CHECK_WORD=PWc8G05FTdp28Oi4ZgrCGUP9SFmgKphS
go run ./cmd/server
```

访问 http://localhost:8080 打开查询页面。

## API

### POST /api/route/query

**请求体**（JSON）：
```json
{
  "trackingType": 1,
  "trackingNumber": "SF1234567890",
  "checkPhoneNo": "1234"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| trackingType | int | 是 | 1=运单号，2=订单号 |
| trackingNumber | string | 是 | 快递单号 |
| checkPhoneNo | string | 否 | 收件人手机后四位 |

**成功响应**：
```json
{
  "code": 0,
  "message": "ok",
  "data": [{
    "mailNo": "SF1234567890",
    "status": "已签收",
    "statusCode": "delivered",
    "total": 8,
    "routes": [
      {
        "opCode": "80",
        "time": "2024-01-15 14:32:00",
        "address": "厦门",
        "remark": "快件已签收，感谢使用顺丰",
        "isFirst": true
      }
    ]
  }]
}
```

### GET /health

```json
{ "code": 0, "message": "ok", "data": { "status": "ok", "time": "..." } }
```

## Docker 部署

```bash
docker build -t sfquery .
docker run -p 8080:8080 \
  -e SF_PARTNER_ID=XMWL43MUNA8 \
  -e SF_CHECK_WORD=PWc8G05FTdp28Oi4ZgrCGUP9SFmgKphS \
  sfquery
```

## 签名算法

顺丰丰桥签名规则：
```
msgDigest = Base64( MD5( msgData + timestamp + checkWord ) )
```
实现见 `internal/sfclient/client.go` → `msgDigest()`
