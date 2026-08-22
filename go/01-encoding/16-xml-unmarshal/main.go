package main

import (
	"encoding/xml"
	"fmt"
)

type Server struct {
	Name string `xml:"name"`
	IP   string `xml:"ip"`
}

type Servers struct {
	XMLName xml.Name `xml:"servers"`
	Version int      `xml:"version,attr"`
	Items   []Server `xml:"server"`
}

func main() {
	data := []byte(`
<servers version="1">
  <server>
    <name>Shanghai</name>
    <ip>127.0.0.1</ip>
  </server>
  <server>
    <name>Beijing</name>
    <ip>127.0.0.2</ip>
  </server>
</servers>`)

	var servers Servers
	// 与 json.Unmarshal 相同，目标变量需要传地址。
	if err := xml.Unmarshal(data, &servers); err != nil {
		fmt.Println("unmarshal xml:", err)
		return
	}

	fmt.Printf("根元素=%s，版本=%d，服务器数量=%d\n", servers.XMLName.Local, servers.Version, len(servers.Items))
	for _, server := range servers.Items {
		fmt.Printf("- %s: %s\n", server.Name, server.IP)
	}
}
