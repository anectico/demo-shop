import { initApp, type App } from '@anectico/sdk/app'

interface Product {
  id: string
  name: string
  price_cents: number
  digital: boolean
}

interface BrowserConfig {
  endpoint: string
  app_endpoint?: string
  release: string
  keys: { ingest: string; capture: string; replay?: string; flags?: string }
}

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T
const money = (cents: number) => `$${(cents / 100).toFixed(2)}`
const cart = new Map<string, number>()
let products: Product[] = []
let anectico: App

async function start() {
  const config = (await (await fetch('/api/config')).json()) as BrowserConfig
  anectico = await initApp({
    endpoint: config.endpoint,
    appEndpoint: config.app_endpoint,
    serviceName: 'shop-web',
    environment: 'demo',
    release: config.release,
    keys: config.keys,
    // This store exists only to produce the play.anectico.com demo; every
    // visitor is simulated, so all categories are granted.
    consent: { diagnostics: true, performance: true, product: true, replay: true },
    replay: { maskAllInputs: true },
  })
  products = ((await (await fetch('/api/products')).json()) as { products: Product[] }).products
  renderCatalog()
  renderCart()
  $<HTMLFormElement>('signin').addEventListener('submit', (event) => {
    event.preventDefault()
    void signIn($<HTMLInputElement>('email').value.trim().toLowerCase())
  })
  $<HTMLButtonElement>('checkout').addEventListener('click', () => void checkout())
  anectico.page('Store')
}

async function signIn(email: string) {
  if (!email) return
  const company = email.split('@')[1] ?? 'personal'
  anectico.identify(email, { email, plan: company.endsWith('.example') ? 'business' : 'personal' })
  anectico.group('company', company, { name: company })
  await anectico.flags?.reload(email).catch(() => undefined)
  $<HTMLFormElement>('signin').hidden = true
  const who = $<HTMLSpanElement>('who')
  who.textContent = `Signed in as ${email}`
  who.hidden = false
  anectico.capture('signed_in')
}

function renderCatalog() {
  const section = $<HTMLElement>('catalog')
  section.replaceChildren(
    ...products.map((product) => {
      const card = document.createElement('article')
      card.innerHTML = `<h3></h3><p></p><button type="button">Add to cart</button>`
      card.querySelector('h3')!.textContent = product.name
      card.querySelector('p')!.textContent = money(product.price_cents)
      const button = card.querySelector('button')!
      button.dataset.product = product.id
      button.addEventListener('click', () => {
        cart.set(product.id, (cart.get(product.id) ?? 0) + 1)
        anectico.capture('added_to_cart', { product_id: product.id, digital: product.digital })
        renderCart()
      })
      return card
    }),
  )
}

function renderCart() {
  const lines = [...cart.entries()].map(([id, quantity]) => {
    const item = document.createElement('li')
    const product = products.find((p) => p.id === id)!
    item.textContent = `${quantity} × ${product.name}`
    return item
  })
  $<HTMLUListElement>('lines').replaceChildren(...lines)
  $<HTMLButtonElement>('checkout').disabled = cart.size === 0
}

async function checkout() {
  const newCheckout = anectico.flags?.isFeatureEnabled('new-checkout') ?? false
  const discountCode = $<HTMLInputElement>('code').value.trim().toUpperCase()
  const body = {
    lines: [...cart.entries()].map(([product_id, quantity]) => ({ product_id, quantity })),
    discount_code: discountCode || undefined,
  }
  anectico.capture('checkout_started', { new_checkout: newCheckout, has_discount: !!discountCode })
  const result = $<HTMLParagraphElement>('result')
  const response = await fetch(newCheckout ? '/api/v2/checkout' : '/api/checkout', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    result.textContent = 'Something went wrong placing your order. Please try again.'
    anectico.capture('checkout_failed', { status: response.status, new_checkout: newCheckout })
    return
  }
  const order = (await response.json()) as { id: string; total_cents: number }
  result.textContent = `Order placed: ${money(order.total_cents)}`
  anectico.capture('order_completed', { order_id: order.id, total_cents: order.total_cents })
  cart.clear()
  renderCart()
}

void start()
