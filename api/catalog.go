package main

// Product is one item in the demo store. Digital items ship by email.
type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PriceUSD int64  `json:"price_cents"`
	Digital  bool   `json:"digital"`
}

var catalog = []Product{
	{ID: "kibble-salmon", Name: "Salmon kibble, 5 kg", PriceUSD: 3499},
	{ID: "harness-trail", Name: "Trail harness", PriceUSD: 4200},
	{ID: "bed-orthopedic", Name: "Orthopedic dog bed", PriceUSD: 8900},
	{ID: "toy-rope", Name: "Rope toy", PriceUSD: 899},
	{ID: "litter-clump", Name: "Clumping litter, 10 kg", PriceUSD: 1899},
	{ID: "gift-25", Name: "E-gift card, $25", PriceUSD: 2500, Digital: true},
	{ID: "gift-50", Name: "E-gift card, $50", PriceUSD: 5000, Digital: true},
}

func productByID(id string) (Product, bool) {
	for _, p := range catalog {
		if p.ID == id {
			return p, true
		}
	}
	return Product{}, false
}

// discountCodes maps a code to its percentage off the order.
var discountCodes = map[string]int64{
	"WELCOME10": 10,
	"PAWS15":    15,
}
