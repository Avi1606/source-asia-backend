# source-asia-backend

## 1. How to Run

### Prerequisites

- Go 1.22+

### Start the server

```sh
go run main.go
```

The server listens on `http://localhost:8080`.

### Curl examples

Create a rate-limited request:

```sh
curl -i -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","payload":{"action":"sync","value":42}}'
```

Read rate-limit stats:

```sh
curl -i http://localhost:8080/stats
```

Create a product:

```sh
curl -i -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Wireless Keyboard","sku":"KB-001","image_urls":["https://cdn.example.com/kb-front.jpg"],"video_urls":["https://cdn.example.com/kb-demo.mp4"]}'
```

List products:

```sh
curl -i "http://localhost:8080/products?limit=20&offset=0"
```

Get one product by ID:

```sh
curl -i http://localhost:8080/products/REPLACE_WITH_PRODUCT_ID
```

Add media to a product:

```sh
curl -i -X POST http://localhost:8080/products/REPLACE_WITH_PRODUCT_ID/media \
  -H "Content-Type: application/json" \
  -d '{"image_urls":["https://cdn.example.com/kb-side.jpg"],"video_urls":["https://cdn.example.com/kb-unboxing.mp4"]}'
```

## 2. Part 1 Design Decisions

The rate limiter uses a fixed 1-minute window per `user_id` with a maximum of 5 accepted requests per window. A fixed window is simple, fast, and easy to reason about for an in-memory assignment service. A sliding window can smooth traffic more precisely, but it needs more bookkeeping and is unnecessary for this scope.

`rejected_total` is cumulative across all windows for each user. When a user exceeds the current window limit, that counter increments and is not reset when the next window starts.

### GET /stats response schema

```json
{
  "users": {
    "user-123": {
      "accepted_current_window": 3,
      "rejected_total": 1
    }
  },
  "global": {
    "total_accepted": 3,
    "total_rejected": 1
  }
}
```

`accepted_current_window` resets to 0 when the window expires.
`rejected_total` is cumulative and never resets.

The `429 Too Many Requests` response does not include a `Retry-After` header or window reset time. In production, this would be added for client backoff.

`POST /request` returns `201 Created` because the server accepts and records a new request event in memory. Even though there is no persistent database row, the accepted request is a newly created in-memory event for the purpose of rate-limit accounting.

Production limitations:

- This implementation is single-instance only.
- Restarting the process loses all rate-limit and catalog state.
- A multi-instance deployment needs Redis or another shared store for consistent rate-limit decisions.
- There is no persistence layer, so product data is also lost on restart.

## 3. Part 2 Design Decisions

Products and media are stored in separate maps. The `products` map contains compact product metadata, while the `media` map stores the larger image and video URL arrays.

`GET /products` reads only the `Product` structs and does not access the media map, so list responses avoid loading or serializing large media arrays. `GET /products/{id}` loads both the product and its media for the full detail response.

Pagination uses `limit` and `offset` over an ordered `[]string` slice of product IDs. The slice preserves insertion order, and each page resolves those IDs back to product metadata.

Pagination defaults: `limit=20`, `offset=0`. Maximum `limit` is capped at 100.

URL validation rules:

- URLs must start with `http://` or `https://`.
- URLs must be no longer than 2048 characters.
- Each URL array may contain at most 20 URLs per request.
- Empty string entries are rejected.

Duplicate SKUs return `409 Conflict` with `{ "error": "sku already exists" }`, which clearly separates uniqueness conflicts from normal validation errors.

Production limitations:

- In-memory storage means all product and media data is lost on restart.
- No full-text search; filtering and sorting would require PostgreSQL or Elasticsearch.
- UUID generation uses `crypto/rand`; safe for single instance but a distributed ID strategy such as ULIDs would be better for multi-node sorted pagination.

Production upgrade path:

- Store products in PostgreSQL.
- Store media rows in a separate product media table.
- Store actual files in object storage behind a CDN.
- Add a unique index on the `sku` column.

## 4. AI Usage

Codex / AI tooling was used to scaffold and implement this project.
