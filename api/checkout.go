package main

import (
	"errors"
	"fmt"
)

// CartLine is one product and quantity in a checkout request.
type CartLine struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// CheckoutRequest is what the storefront sends to either checkout version.
type CheckoutRequest struct {
	Lines        []CartLine `json:"lines"`
	DiscountCode string     `json:"discount_code,omitempty"`
}

// Order is a priced checkout.
type Order struct {
	ID            string `json:"id"`
	SubtotalCents int64  `json:"subtotal_cents"`
	DiscountCents int64  `json:"discount_cents"`
	TotalCents    int64  `json:"total_cents"`
}

var errEmptyCart = errors.New("cart is empty")

type pricedLine struct {
	product  Product
	quantity int64
}

func priceLines(lines []CartLine) ([]pricedLine, int64, error) {
	if len(lines) == 0 {
		return nil, 0, errEmptyCart
	}
	priced := make([]pricedLine, 0, len(lines))
	var subtotal int64
	for _, line := range lines {
		product, ok := productByID(line.ProductID)
		if !ok || line.Quantity <= 0 {
			return nil, 0, fmt.Errorf("unknown product or quantity: %q", line.ProductID)
		}
		priced = append(priced, pricedLine{product: product, quantity: line.Quantity})
		subtotal += product.PriceUSD * line.Quantity
	}
	return priced, subtotal, nil
}

// checkoutV1 is the current checkout: the discount applies to the order total.
func checkoutV1(req CheckoutRequest, orderID string) (Order, error) {
	_, subtotal, err := priceLines(req.Lines)
	if err != nil {
		return Order{}, err
	}
	discount := subtotal * discountCodes[req.DiscountCode] / 100
	return Order{ID: orderID, SubtotalCents: subtotal, DiscountCents: discount, TotalCents: subtotal - discount}, nil
}
