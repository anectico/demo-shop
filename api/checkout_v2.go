package main

// checkoutV2 is the new checkout behind the new-checkout flag. It prices the
// cart the same way as the current checkout while the new flow is rolled out.
func checkoutV2(req CheckoutRequest, orderID string) (Order, error) {
	_, subtotal, err := priceLines(req.Lines)
	if err != nil {
		return Order{}, err
	}
	discount := subtotal * discountCodes[req.DiscountCode] / 100
	return Order{ID: orderID, SubtotalCents: subtotal, DiscountCents: discount, TotalCents: subtotal - discount}, nil
}
