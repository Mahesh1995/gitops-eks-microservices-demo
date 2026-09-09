// cart: a small Express service for a per-user shopping cart.
//
// Persistence is OPTIONAL:
//   - If REDIS_ADDR (host:port) is set and reachable, carts are stored in Redis.
//   - Otherwise it falls back to an in-memory Map (fine for local/dev).
//
// Endpoints:
//   GET    /cart/:user            -> { user, items: [{product_id, qty}] }
//   POST   /cart/:user/items      -> add {product_id, qty}; returns the cart
//   DELETE /cart/:user            -> empty the cart
//   GET    /healthz               -> liveness
//   GET    /readyz                -> readiness (store reachable)
//   GET    /metrics               -> Prometheus text-format metrics

const express = require("express");

const PORT = process.env.PORT || 3000;
const REDIS_ADDR = process.env.REDIS_ADDR; // e.g. "redis:6379" — unset = in-memory

const app = express();
app.use(express.json());

let metrics = { requests: 0, items_added: 0 };
app.use((req, _res, next) => { metrics.requests++; next(); });

// ---- storage abstraction -------------------------------------------------
let redis = null;
const memory = new Map(); // user -> [{product_id, qty}]

if (REDIS_ADDR) {
  const IORedis = require("ioredis");
  const [host, port] = REDIS_ADDR.split(":");
  redis = new IORedis({ host, port: Number(port || 6379), lazyConnect: true, maxRetriesPerRequest: 2 });
  redis.connect().then(
    () => console.log(`cart: connected to Redis at ${REDIS_ADDR}`),
    (e) => { console.error(`cart: Redis connect failed (${e.message}); using in-memory`); redis = null; }
  );
} else {
  console.log("cart: REDIS_ADDR not set — using in-memory store");
}

async function getCart(user) {
  if (redis) {
    const raw = await redis.get(`cart:${user}`);
    return raw ? JSON.parse(raw) : [];
  }
  return memory.get(user) || [];
}

async function saveCart(user, items) {
  if (redis) return redis.set(`cart:${user}`, JSON.stringify(items));
  memory.set(user, items);
}

// ---- routes --------------------------------------------------------------
app.get("/cart/:user", async (req, res) => {
  res.json({ user: req.params.user, items: await getCart(req.params.user) });
});

app.post("/cart/:user/items", async (req, res) => {
  const { product_id, qty } = req.body || {};
  if (!product_id || !Number.isInteger(qty) || qty < 1) {
    return res.status(400).json({ error: "body must be {product_id: string, qty: positive int}" });
  }
  const items = await getCart(req.params.user);
  const existing = items.find((i) => i.product_id === product_id);
  if (existing) existing.qty += qty;
  else items.push({ product_id, qty });
  await saveCart(req.params.user, items);
  metrics.items_added += qty;
  res.status(201).json({ user: req.params.user, items });
});

app.delete("/cart/:user", async (req, res) => {
  await saveCart(req.params.user, []);
  res.status(204).end();
});

app.get("/healthz", (_req, res) => res.send("ok"));

app.get("/readyz", async (_req, res) => {
  if (!redis) return res.send("ready (in-memory)");
  try { await redis.ping(); res.send("ready"); }
  catch { res.status(503).send("redis unreachable"); }
});

app.get("/metrics", (_req, res) => {
  res.type("text/plain; version=0.0.4").send(
    "# HELP cart_requests_total Total HTTP requests served.\n" +
    "# TYPE cart_requests_total counter\n" +
    `cart_requests_total ${metrics.requests}\n` +
    "# HELP cart_items_added_total Total cart items added.\n" +
    "# TYPE cart_items_added_total counter\n" +
    `cart_items_added_total ${metrics.items_added}\n`
  );
});

app.listen(PORT, () => console.log(`cart listening on :${PORT}`));
