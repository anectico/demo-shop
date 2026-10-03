# Pawprint — the Anectico demo shop

The source of the small pet store behind Anectico's public demo. It is instrumented with the
Anectico SDKs, and simulated customers use it around the clock, so the demo always shows the
product working on live data.

- `api/` — Go backend (Gin) that serves the storefront and the checkout.
- `web/` — the storefront, using `initApp()` from `@anectico/sdk/app`.

The new checkout (`api/checkout_v2.go`, behind the `new-checkout` flag) contains a real bug on
purpose. Open the demo to see who it hits, watch their session, and find the commit that caused it.

This repository mirrors the demo's source for reading; the demo is built from Anectico's main
repository.
