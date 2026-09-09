"""product-catalog: a tiny Flask service that serves a static product list.

Endpoints:
    GET /products        -> {"products": [...]}
    GET /products/<id>   -> a single product (404 if unknown)
    GET /healthz         -> liveness
    GET /readyz          -> readiness (catalog loaded)
    GET /metrics         -> Prometheus text-format metrics

The catalog is loaded once from products.json. Kept stateless on purpose.
"""
import json
import os
import pathlib

from flask import Flask, jsonify, abort, Response

app = Flask(__name__)

CATALOG_PATH = pathlib.Path(__file__).parent / "products.json"
_PRODUCTS = json.loads(CATALOG_PATH.read_text())
_REQUESTS = {"count": 0}


@app.before_request
def _count():
    _REQUESTS["count"] += 1


@app.get("/products")
def list_products():
    return jsonify(products=_PRODUCTS)


@app.get("/products/<product_id>")
def get_product(product_id):
    for p in _PRODUCTS:
        if p["id"] == product_id:
            return jsonify(p)
    abort(404, description=f"product {product_id} not found")


@app.get("/healthz")
def healthz():
    return "ok", 200


@app.get("/readyz")
def readyz():
    # Ready only if the catalog actually loaded.
    return ("ready", 200) if _PRODUCTS else ("no catalog", 503)


@app.get("/metrics")
def metrics():
    body = (
        "# HELP product_catalog_requests_total Total HTTP requests served.\n"
        "# TYPE product_catalog_requests_total counter\n"
        f"product_catalog_requests_total {_REQUESTS['count']}\n"
        "# HELP product_catalog_products Number of products in the catalog.\n"
        "# TYPE product_catalog_products gauge\n"
        f"product_catalog_products {len(_PRODUCTS)}\n"
    )
    return Response(body, mimetype="text/plain; version=0.0.4")


if __name__ == "__main__":
    port = int(os.getenv("PORT", "5000"))
    # For learning/local use. In-cluster the Dockerfile runs gunicorn instead.
    app.run(host="0.0.0.0", port=port)
