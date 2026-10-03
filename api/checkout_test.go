package main

import "testing"

func TestCheckoutV1AppliesTheDiscountToTheOrder(t *testing.T) {
	order, err := checkoutV1(CheckoutRequest{Lines: []CartLine{{ProductID: "gift-50", Quantity: 1}}, DiscountCode: "WELCOME10"}, "o")
	if err != nil || order.TotalCents != 4500 {
		t.Fatalf("v1 = %+v, %v; want a $45.00 total", order, err)
	}
}

func TestCheckoutV2MatchesV1(t *testing.T) {
	order, err := checkoutV2(CheckoutRequest{Lines: []CartLine{{ProductID: "toy-rope", Quantity: 2}}, DiscountCode: "PAWS15"}, "o")
	if err != nil || order.SubtotalCents != 1798 || order.DiscountCents != 269 {
		t.Fatalf("v2 = %+v, %v", order, err)
	}
}
