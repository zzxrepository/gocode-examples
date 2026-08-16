//go:build ignore

// 订单服务调用历史会员服务：Thrift 生成客户端常把查询 key 和 trace 作为专用参数。
package main

import (
	"context"
	"fmt"
)

type MemberFeatureClient struct{ serviceName string }

func NewMemberFeatureClient(serviceName string) (*MemberFeatureClient, error) {
	return &MemberFeatureClient{serviceName: serviceName}, nil
}

type FeatureKey struct {
	domain string
	keys   []string
}
type Trace struct {
	TraceID string
	Caller  string
}
type FeatureResponse struct {
	Errno  int
	Errmsg string
	Values map[string]string
}

func (c *MemberFeatureClient) BuildFeatureKey(domain string, keys []string, params map[string]string) (*FeatureKey, error) {
	return &FeatureKey{domain: domain, keys: keys}, nil
}

func (c *MemberFeatureClient) AppendKeys(featureKeys ...*FeatureKey) []string {
	features := make([]string, 0)
	for _, key := range featureKeys {
		for _, name := range key.keys {
			features = append(features, key.domain+"."+name)
		}
	}
	return features
}

func (c *MemberFeatureClient) Mget(ctx context.Context, features []string, trace *Trace) (*FeatureResponse, error) {
	fmt.Printf("[Thrift] %s.Mget: features=%v, caller=%s\n", c.serviceName, features, trace.Caller)
	return &FeatureResponse{Values: map[string]string{
		"member.tier": "gold", "member.points": "820",
	}}, nil
}

func main() {
	client, err := NewMemberFeatureClient("disf!legacy-member-profile")
	if err != nil {
		panic(err)
	}
	key, err := client.BuildFeatureKey("member", []string{"tier", "points"}, map[string]string{"member_id": "m-1001"})
	if err != nil {
		panic(err)
	}
	resp, err := client.Mget(context.Background(), client.AppendKeys(key), &Trace{TraceID: "trace-order-1001", Caller: "book-order"})
	if err != nil {
		panic(err)
	}
	if resp.Errno != 0 {
		panic(resp.Errmsg)
	}
	fmt.Println("member features:", resp.Values)
}
