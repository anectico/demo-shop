package main

import "testing"

func TestCheckoutV1AppliesTheDiscountToTheOrder(t *testing.T) {
	order, err := checkoutV1(CheckoutRequest{Lines: []CartLine{{ProductID: "gift-50", Quantity: 1}}, DiscountCode: "WELCOME10"}, "o")
	if err != nil || order.TotalCents != 4500 {
		t.Fatalf("v1 = %+v, %v; want a $45.00 total", order, err)
	}
}

func TestCheckoutV2SpreadsTheDiscountOverPhysicalItems(t *testing.T) {
	order, err := checkoutV2(CheckoutRequest{Lines: []CartLine{
		{ProductID: "toy-rope", Quantity: 2}, {ProductID: "gift-25", Quantity: 1},
	}, DiscountCode: "PAWS15"}, "o")
	if err != nil || order.SubtotalCents != 4298 || order.DiscountCents != 644 {
		t.Fatalf("v2 = %+v, %v", order, err)
	}
}
