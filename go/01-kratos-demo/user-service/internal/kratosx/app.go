package kratosx

import (
	"net/http"
	"os"

	kratos "github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPApp(name, addr string, handler http.Handler) *kratos.App {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"service.name", name,
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)

	server := khttp.NewServer(
		khttp.Address(addr),
		khttp.Middleware(
			recovery.Recovery(),
			logging.Server(logger),
		),
	)
	server.HandlePrefix("/", handler)

	return kratos.New(
		kratos.Name(name),
		kratos.Logger(logger),
		kratos.Server(server),
	)
}
