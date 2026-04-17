# Verification 2026-04-17

## Commands

```bash
gofmt -w internal/proxy/responses.go internal/proxy/response_state.go internal/proxy/responses_test.go
go test ./internal/proxy
go test ./...
make test
curl -H "Authorization: Bearer <token>" http://127.0.0.1:39530/v1/models
POST /v1/chat/completions (qwen3-8b, stream=false/true)
POST /v1/responses
POST /v1/responses with previous_response_id
GET /v1/responses/{id}
GET /v1/responses/{id}/input_items
POST https://chat.fireworks.ai/chat/single for minimax-m2p1, kimi-k2-thinking, kimi-k2-instruct-0905, cogito-671b-v2-p1
```

## Results

- `gofmt` completed for `internal/proxy/responses.go`, `internal/proxy/response_state.go`, `internal/proxy/responses_test.go`.
- `go test ./internal/proxy` passed on 2026-04-17.
- `go test ./...` passed on 2026-04-17.
- `make test` passed on 2026-04-17.
- Docker image `firew2oai:fixed` was rebuilt and `firew2oai-smoke` was replaced on `39529 -> 39527`.
- Local rebuilt instance on `:39530` returned 12 models from `/v1/models`.
- `qwen3-8b` no longer leaked `<think>` tags to OpenAI-compatible output.
- `/v1/responses` now preserved multi-turn context through `previous_response_id`.
- `GET /v1/responses/{id}` and `GET /v1/responses/{id}/input_items` both returned stored state successfully.
- `/v1/responses` now stores raw request items and raw history items, not only flattened chat text.
- `response.completed.response.usage` now returns non-zero local estimates for `input_tokens`, `output_tokens`, and `total_tokens`.
- Package tests now cover streamed/non-streamed `usage`, undeclared tool rejection, and `previous_response_id + function_call_output` prompt reconstruction.
- Direct upstream checks confirmed `minimax-m2p1`, `kimi-k2-thinking`, `kimi-k2-instruct-0905`, and `cogito-671b-v2-p1` all returned Fireworks `type=error` with upstream `404 Not Found`.

## Limits

- End-to-end `new-api -> firew2oai -> Codex` inference was not executed in this pass because the local New API management credentials were not configured in this environment.
