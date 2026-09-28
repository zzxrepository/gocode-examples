package router

import (
	"net/http"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/controller"
)

func New(syncOrders, asyncOrders, callbackOrders *controller.OrderController, transactions *controller.TransactionController) http.Handler {
	mux := http.NewServeMux()
	registerOrders(mux, "/demo/sync/orders", syncOrders)
	registerOrders(mux, "/demo/async/orders", asyncOrders)
	registerOrders(mux, "/demo/callback/orders", callbackOrders)
	mux.HandleFunc("POST /demo/transaction/commit", transactions.Commit)
	mux.HandleFunc("POST /demo/transaction/abort", transactions.Abort)
	return mux
}

func registerOrders(mux *http.ServeMux, path string, orders *controller.OrderController) {
	mux.HandleFunc("POST "+path, orders.Create)
	mux.HandleFunc("GET "+path+"/{id}", orders.Get)
	mux.HandleFunc("POST "+path+"/{id}/pay", orders.Pay)
	mux.HandleFunc("POST "+path+"/{id}/cancel", orders.Cancel)
}
