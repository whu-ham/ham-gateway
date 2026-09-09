# HAM Gateway

HAM OpenAPI Gateway 是一个专门用于管理和转发 HAM OpenAPI 请求的服务组件。

## 架构

```
客户端 ──HTTP/JSON──→ Gateway ──gRPC/mTLS──→ HAM API
(Backend/CF Worker)   (Gin Server)          (open-api.ham.nowcent.cn:4443)
```

- **客户端 → Gateway**: RESTful HTTP/JSON API
- **Gateway → HAM API**: gRPC with mTLS authentication

## 核心功能

- **RESTful API**: 提供标准的 HTTP/JSON 接口给客户端
- **gRPC 客户端**: 使用 gRPC 与 HAM API 通信
- **mTLS 认证**: 使用客户端证书进行安全认证
- **请求转换**: 将 HTTP 请求转换为 gRPC 调用
- **访问控制**: 基于 token 的身份验证

## API 端点

| Method | Path | Query Params | Description |
|--------|------|--------------|-------------|
| GET | `/health` | - | 健康检查 |
| GET | `/api/v1/external/ham/course/search` | `keyword`, `keyword_type` | 搜索课程 |
| GET | `/api/v1/external/ham/score/stat` | `course_name`, `instructor` | 获取课程成绩统计 |
| GET | `/api/v1/external/ham/score/by-course` | `course_name`, `page_num`, `page_size` | 获取课程名对应的所有教学团队成绩统计 |

## 环境变量

| Variable | Default | Description |
|----------|---------|-------------|
| `HAM_API_BASE_URL` | `open-api.ham.nowcent.cn:4443` | HAM gRPC API 地址 |
| `HAM_OPEN_APP_ID` | - | HAM API 应用 ID |
| `HAM_OPEN_APP_SECRET` | - | HAM API 应用密钥 |
| `HAM_GRPC_MTLS_CLIENT_CRT` | - | mTLS 客户端证书路径 |
| `HAM_GRPC_MTLS_CLIENT_KEY` | - | mTLS 客户端密钥路径 |
| `GATEWAY_AUTH_TOKEN` | - | Gateway 认证 token |
| `PORT` | `8080` | 服务监听端口 |

## 开发

### Proto 代码生成

本项目使用 `proto/` submodule (HAM OpenAPI 定义) 生成 Go 代码。

**更新 submodule:**

```bash
git submodule update --init --recursive
```

**生成代码 (跨平台):**

```bash
go generate ./...
```

这会自动运行 `protoc` 生成以下文件：

- `gen/proto/common.pb.go`
- `gen/proto/course.pb.go`
- `gen/proto/course_grpc.pb.go`
- `gen/proto/score.pb.go`
- `gen/proto/score_grpc.pb.go`

**前置要求:**

- 安装 `protoc`: <https://grpc.io/docs/protoc-installation/>
- 安装 Go 插件:

  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```

### 构建

```bash
go build -o gateway ./cmd/gateway
```

### 运行

```bash
# 设置环境变量
export HAM_OPEN_APP_ID="your-app-id"
export HAM_OPEN_APP_SECRET="your-app-secret"
export HAM_GRPC_MTLS_CLIENT_CRT="/path/to/client.crt"
export HAM_GRPC_MTLS_CLIENT_KEY="/path/to/client.key"
export GATEWAY_AUTH_TOKEN="your-token"

# 启动服务
./gateway
```

## 测试

```bash
# 健康检查
curl http://localhost:8080/health

# 搜索课程
curl -H "X-Gateway-Token: your-token" \
  "http://localhost:8080/api/v1/external/ham/course/search?keyword=数学&keyword_type=1"

# 获取课程统计
curl -H "X-Gateway-Token: your-token" \
  "http://localhost:8080/api/v1/external/ham/score/stat?course_name=高等数学&instructor=张三"
```

## 项目结构

```
.
├── cmd/gateway/        # 主程序入口
├── internal/
│   ├── ham/           # HAM API gRPC 客户端
│   ├── handlers/      # HTTP 请求处理器
│   ├── middleware/    # HTTP 中间件
│   └── generate.go    # go:generate 指令
├── gen/proto/         # 生成的 proto 代码
├── proto/             # HAM OpenAPI proto 定义 (submodule)
└── README.md
```
