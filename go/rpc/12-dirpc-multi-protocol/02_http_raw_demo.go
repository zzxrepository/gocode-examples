//go:build ignore

// 订单服务调用没有生成 IDL 的优惠券 HTTP 服务，并把上游 trace 和鉴权 header 继续传下去。
package main

import (
	"context"
	"fmt"
)

type Header map[string][]string
type httpHeaderKey struct{}

func SetHTTPHeader(ctx context.Context, header Header) context.Context {
	// WithValue 返回新 ctx；调用方必须接住返回值，透传才会生效。
	return context.WithValue(ctx, httpHeaderKey{}, header)
}

type HTTPClient struct{ serviceName string }

func NewHTTPClient(serviceName string) *HTTPClient { return &HTTPClient{serviceName: serviceName} }

func (c *HTTPClient) Post(ctx context.Context, path, contentType string, body []byte) ([]byte, error) {
	header, _ := ctx.Value(httpHeaderKey{}).(Header)
	fmt.Printf("[raw HTTP] %s POST %s, content-type=%s, body=%q, header=%v\n", c.serviceName, path, contentType, body, header)
	return []byte(`{"errno":0,"data":{"discount_cents":1200}}`), nil
}

var couponClient = NewHTTPClient("disf!book-coupon")

func validateCoupon(ctx context.Context, upstreamHeader Header, couponCode, orderID string) ([]byte, error) {
	// 原生转发没有生成客户端替你处理 header，因此显式放入 ctx。
	ctx = SetHTTPHeader(ctx, upstreamHeader)
	body := []byte("coupon_code=" + couponCode + "&order_id=" + orderID)
	return couponClient.Post(ctx, "/v1/coupons/validate", "application/x-www-form-urlencoded", body)
}

func main() {
	resp, err := validateCoupon(context.Background(), Header{
		"Trace-Id":      {"trace-order-1001"},
		"Authorization": {"Bearer example-token"},
	}, "READMORE", "order-1001")
	if err != nil {
		panic(err)
	}
	fmt.Println("response:", string(resp))
}
