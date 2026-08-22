package store

import (
	"context"
	"errors"
	"testing"
)

// 这些测试不依赖 MySQL，用于固定 Demo 的输入边界。
func TestValidateProductInput(t *testing.T) {
	tests := []struct {
		name  string
		input CreateProductInput
		want  error
	}{
		{"valid", CreateProductInput{SKU: "sku-1", Name: "keyboard", PriceCents: 39900, Stock: 10}, nil},
		{"empty sku", CreateProductInput{Name: "keyboard", PriceCents: 1, Stock: 1}, ErrInvalidProduct},
		{"negative stock", CreateProductInput{SKU: "sku-1", Name: "keyboard", PriceCents: 1, Stock: -1}, ErrInvalidProduct},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateProductInput(tt.input); !errors.Is(got, tt.want) {
				t.Fatalf("validateProductInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateOrderInput(t *testing.T) {
	if err := validateOrderInput(PlaceOrderInput{CustomerID: 1, ProductID: 2, Quantity: 1}); err != nil {
		t.Fatalf("valid order rejected: %v", err)
	}
	if err := validateOrderInput(PlaceOrderInput{CustomerID: 1, ProductID: 2, Quantity: 0}); !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("zero quantity error = %v, want ErrInvalidOrder", err)
	}
}

// 更新和删除在访问数据库前应先拒绝明显无效的参数，
// 因此这组测试不需要启动 MySQL，也不会使用传入的 nil DB。
func TestProductMutationInput(t *testing.T) {
	ctx := context.Background()

	if _, err := UpdateProductName(ctx, nil, 0, "keyboard"); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("invalid update error = %v, want ErrInvalidProduct", err)
	}
	if _, err := UpdateProductNameSQLX(ctx, nil, 1, ""); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("empty-name update error = %v, want ErrInvalidProduct", err)
	}
	if _, err := DeleteProduct(ctx, nil, 0); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("invalid delete error = %v, want ErrInvalidProduct", err)
	}
	if _, err := DeleteProductSQLX(ctx, nil, -1); !errors.Is(err, ErrInvalidProduct) {
		t.Fatalf("invalid sqlx delete error = %v, want ErrInvalidProduct", err)
	}
}
