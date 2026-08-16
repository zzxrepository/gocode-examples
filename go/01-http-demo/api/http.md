# API 示例

Base URL：`http://127.0.0.1:8081`

## Register

```bash
curl -X POST http://127.0.0.1:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"123456"}'
```

## Login

```bash
curl -X POST http://127.0.0.1:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"123456"}'
```

将返回值中的 `data.token` 保存到环境变量：

```bash
export TOKEN='replace-with-token'
```

## Create Post

```bash
curl -X POST http://127.0.0.1:8081/api/v1/posts \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first post","content":"hello net/http"}'
```

## List Posts

```bash
curl http://127.0.0.1:8081/api/v1/posts
```

## Get / Update / Delete

```bash
curl http://127.0.0.1:8081/api/v1/posts/1

curl -X PUT http://127.0.0.1:8081/api/v1/posts/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"updated title","content":"updated content"}'

curl -X DELETE http://127.0.0.1:8081/api/v1/posts/1 \
  -H "Authorization: Bearer $TOKEN"
```
