package store

import (
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
