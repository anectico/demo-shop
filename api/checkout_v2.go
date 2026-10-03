package main

// checkoutV2 is the new checkout behind the new-checkout flag. It spreads the
// discount across the physical items so each shipment carries its own share,
// which the warehouse needs for partial returns.
func checkoutV2(req CheckoutRequest, orderID string) (Order, error) {
	lines, subtotal, err := priceLines(req.Lines)
	if err != nil {
		return Order{}, err
	}
	discount := subtotal * discountCodes[req.DiscountCode] / 100

	var physical int64
	for _, line := range lines {
		if !line.product.Digital {
			physical += line.quantity
		}
	}
	var allocated int64
	if discount > 0 {
		perItem := discount / physical
		allocated = perItem * physical
	}
	return Order{ID: orderID, SubtotalCents: subtotal, DiscountCents: allocated, TotalCents: subtotal - allocated}, nil
}
