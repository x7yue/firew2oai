# Repository Guidelines

## Project Structure & Module Organization
`cmd/server/main.go` 是唯一可执行入口。核心代码位于 `internal/`：`config` 负责参数与环境变量，`proxy` 实现 OpenAI 兼容接口，`transport` 处理上游 HTTP 访问，`tokenauth` 和 `whitelist` 负责访问控制。测试文件与被测包同目录，命名为 `*_test.go`。部署相关文件包括 `Dockerfile`、`docker-compose.yml`，构建产物输出到 `bin/`。设计、审计和阶段性说明放入 `docs/`。

## Build, Test, and Development Commands
优先使用 `Makefile` 中的命令：

- `make build`：编译 `./cmd/server`，生成 `bin/firew2oai`。
- `make run`：先构建，再启动本地服务。
- `make test`：执行 `go test -v -race ./...`。
- `make lint`：执行 `golangci-lint run ./...`。
- `make build-all`：交叉编译 Linux、macOS、Windows 产物。
- `make docker-up`：通过 Docker Compose 启动服务；`make docker-down` 停止服务。

本地快速确认可在 `make build` 后运行 `./bin/firew2oai --help`。

## Coding Style & Naming Conventions
遵循标准 Go 风格，提交前使用 `gofmt`，保持导入有序。包名使用简短小写形式，例如 `proxy`、`transport`。导出标识符使用 `CamelCase`，非导出函数和变量使用 `camelCase`。处理器和中间件应保持单一职责，配置应显式传入，避免隐藏默认行为。日志继续使用现有 `log/slog` 结构化风格。

## Testing Guidelines
新增测试应放在被测包同目录，文件名使用 `*_test.go`，测试函数使用 `TestXxx`，基准测试使用 `BenchmarkXxx`。认证、代理转换、上游传输和 IP 过滤逻辑应覆盖成功路径与失败路径。提交 PR 前至少运行 `make test`；涉及行为或接口变化时，同时运行 `make lint`。

## Commit & Pull Request Guidelines
现有提交历史采用简短的约定式标题，例如 `feat: ...`、`docs: ...`。每个提交只处理一个明确变更。PR 描述应说明用户可见影响、列出已运行的验证命令，并明确配置项或 API 行为变化。若修改请求或响应格式，应附上示例 `curl` 或关键响应片段。

## Security & Configuration Tips
不要提交真实 API Key、令牌 JSON 或本地私密配置。开发环境优先使用 `API_KEY`、`PORT`、`CORS_ORIGINS`、`IP_WHITELIST` 等环境变量。若为测试临时放宽 CORS 或 IP 白名单，必须在 PR 中说明，便于审查时确认不会误带到生产配置。
