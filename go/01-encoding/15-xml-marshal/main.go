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
	servers := Servers{
		Version: 1,
		Items: []Server{
			{Name: "Shanghai", IP: "127.0.0.1"},
			{Name: "Beijing", IP: "127.0.0.2"},
		},
	}

	data, err := xml.MarshalIndent(servers, "", "  ")
	if err != nil {
		fmt.Println("marshal xml:", err)
		return
	}

	fmt.Println(xml.Header + string(data))
}
