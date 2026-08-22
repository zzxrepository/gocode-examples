# encoding 示例

本目录对应文档 [`03-encoding.md`](../../../gocode/src/backend/go/advanced/01-standard-library/03-encoding.md)。

每个子目录都是一个可独立运行的 `main` 包。进入对应目录后执行：

```bash
go run .
```

HTTP 示例启动后，再按程序日志中的 `curl` 命令发送请求。它们使用不同端口，可以分别调试。

| 目录 | 对应知识点 |
| --- | --- |
| `01-marshal` | `json.Marshal`：结构体编码为紧凑 JSON |
| `02-marshal-indent` | `json.MarshalIndent`：格式化 JSON |
| `03-unmarshal` | `json.Unmarshal`：JSON 解码到结构体和切片 |
| `04-exported-fields` | 只有导出字段能被 JSON 处理 |
| `05-struct-tags` | 字段名映射、无 tag 与 snake_case 的区别 |
| `06-http-json-tags` | 在 HTTP handler 中使用 JSON tag |
| `07-omitempty` | `omitempty` 与指针字段的零值语义 |
| `08-ignore-field-and-dto` | `json:"-"` 与响应 DTO |
| `09-http-encoder` | `json.Encoder` 写 HTTP JSON 响应 |
| `10-http-decoder` | `json.Decoder` 读 HTTP JSON 请求 |
| `11-disallow-unknown-fields` | 拒绝未知 JSON 字段 |
| `12-limit-body-and-single-value` | 限制请求体并拒绝多个 JSON 值 |
| `13-dynamic-json` | `map[string]any` 的类型断言与数字默认类型 |
| `14-use-number` | `Decoder.UseNumber` 保留大整数精度 |
| `15-xml-marshal` | `xml.MarshalIndent` 与 XML tag |
| `16-xml-unmarshal` | `xml.Unmarshal` 与 XML tag |

文档中 `Marshal`、`Unmarshal`、`Decoder`、`UseNumber`、XML `typeInfo` 等源码片段是 Go 标准库的内部实现摘录，不能作为独立业务程序复制运行；相应的外部行为已由上面的 demo 覆盖。
