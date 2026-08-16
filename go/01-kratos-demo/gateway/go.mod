module github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway

go 1.26.0

toolchain go1.26.5

require (
	github.com/go-kratos/kratos/v2 v2.8.4
	github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service v0.0.0
	github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service v0.0.0
	golang.org/x/time v0.5.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service => ../post-service

replace github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service => ../user-service

require (
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto v0.0.0-20231212172506-995d672761c0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/grpc v1.64.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
