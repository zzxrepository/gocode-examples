# API

Base URL: `http://127.0.0.1:8080`

`8080` 是 gateway 端口。前端只访问 gateway，不直接访问 `user-service` 和 `post-service`。

## Register

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"123456"}'
```

## Login

```bash
curl -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"123456"}'
```

Copy `data.token` from the response:

```bash
export TOKEN='replace-with-token'
```

## Create Post

```bash
curl -X POST http://127.0.0.1:8080/api/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first post","content":"hello gin"}'
```

## List Posts

```bash
curl http://127.0.0.1:8080/api/v1/posts
```

## Get Post

```bash
curl http://127.0.0.1:8080/api/v1/posts/1
```

## Update Post

```bash
curl -X PUT http://127.0.0.1:8080/api/v1/posts/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"updated title","content":"updated content"}'
```

## Delete Post

```bash
curl -X DELETE http://127.0.0.1:8080/api/v1/posts/1 \
  -H "Authorization: Bearer $TOKEN"
```

## Internal Service Health Check

```bash
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8081/health
curl http://127.0.0.1:8082/health
```

## Rate Limit Check

临时调小 `gateway/configs/local.yaml` 里的 gateway 限流参数：

```yaml
rate_limit:
  rps: 2
  burst: 2
```

然后启动 gateway：

```bash
make run-gateway
```

快速请求几次：

```bash
for i in $(seq 1 10); do
  curl -i http://127.0.0.1:8080/health
done
```

超过限流后会返回：

```text
HTTP/1.1 429 Too Many Requests
```
