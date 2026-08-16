# API

Base URL: `http://127.0.0.1:8080`

`8080` 是后端 API 端口。

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

## Health Check

```bash
curl http://127.0.0.1:8080/health
```
