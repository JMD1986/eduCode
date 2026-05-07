# Boilerplate: systems design — tradeoffs and data model

Discussion notes for the education app (class catalog + enrollment): language and framework choices, suggested packages, and core relational data model rationale. Aligned with the stack in the roadmap (Go API, React + TypeScript, GCP, Postgres).

---

## Languages and platforms

**Go on the backend** fits a classic **API + managed DB + stateless containers** 
shape: binary in a small image, predictable memory, goroutines for concurrent I/O. 
The main tradeoff is **not sharing a language with React** unless you invest in 
OpenAPI/protobuf codegen for types. For **learning Go**, the backend is a good 
place to teach idioms (errors, `context`, interfaces) with real HTTP and SQL.

**TypeScript + React** is a strong default for **rich client UX** 
(filters, forms, accessibility) and a large pool of patterns and 
libraries. The tradeoff is **operational surface**: build tooling,
 client-side routing, CORS, and auth token handling—you own that 
 integration explicitly with a separate API.

**GCP (Cloud Run + Cloud SQL Postgres)** aligns with **12-factor** 
APIs: horizontal scale of stateless services, Postgres for 
**transactions and constraints** around enrollments.
 Alternatives bite in different places:

- **Firestore** simplifies some ops but makes **“exactly capacity 
- enrollments across competing writers”** more awkward than a single 
- Postgres transaction.
- 
- **GKE** buys flexibility at the cost of team Kubernetes expertise you may not need yet.

---

## Framework and package tradeoffs (Go)

Rough spectrum:

| Layer | Minimal / teachable | Productive middle | Heavier abstraction |
| ----- | --------------------- | ----------------- | --------------------- |
| HTTP routing | `net/http` + Go 1.22+ mux | **chi** (small, stdlib-aligned) | Gin / Echo |
| SQL | Raw `database/sql` | **sqlc** (+ migrations) | GORM / Ent |

**Router**

- Pure stdlib is possible and very educational, but middleware composition and parameter 
- parsing are where **chi** (or a thin wrapper) saves repeated boilerplate without 
- hiding how `http.Handler` works.

**DB access**

- **sqlc** is a sweet spot for this domain: you write SQL, generate type-safe Go, 
- and **enrollment queries stay explicit**—good for correctness and for 
- learning SQL + Go together.
- ORMs accelerate CRUD but often obscure **transactions and locking** 
- when you debug “race on last seat.”

**Migrations**

- Treat schema as **versioned** (Atlas, goose, golang-migrate) from day one; 
- Cloud Run should not rely on manual DDL.

**General**

- Avoid pulling in optional packages (tracing, Redis, etc.) until a 
- requirement forces them—instrumentation via **GCP logging/metrics hooks** plus request IDs often suffices early.

---

## Frontend packaging

**Vite + React SPA** behind your API keeps deployment simple (static assets + `/api` to Go).

**Next.js** becomes attractive if you need **SEO-heavy public catalog pages** or a **BFF** in Node; that is a second runtime and deployment path, so it is a deliberate tradeoff—not a default for “API + dashboard” style apps.

---

## Data model and why (systems reasoning)

The hard invariant is: **at no time may active enrollments exceed capacity**, even under concurrent requests.

### `users`

Stable **internal** identity keyed to the IdP subject (for example OIDC `sub`). \
You need foreign-key targets for enrollments and audit; storing only 
JWT claims on the wire is insufficient for relational integrity and lifecycle 
(user deactivated in IdP, historical roster, and so on).

### `classes`

Product-owned metadata and **capacity**, plus **lifecycle** 
(`draft` / `published` / `archived`) and enrollment window. 
Keeps listing and enrollment rules in one authoritative row.

### `class_sessions` (optional early)

Normalize when you care about **per-meeting reminders, 
ICS exports, overlap detection**. Until then, a **schedule 
summary** on `classes` reduces join complexity—but do not 
defer forever if calendars become a wedge feature.

### `enrollments`

- **Unique `(user_id, class_id)`** for double-submit protection.
- **`status`** (for example `active` / `withdrawn`) preserves history without deleting rows.

Correctness relies on **`BEGIN … SELECT class FOR UPDATE … INSERT/COUNT`** 
(or equivalent) inside one transaction—not on “hope the UI only clicks once.”

### `waitlist_entries` (later phase)

Ordering + promotion needs **serialized updates** per class 
(same correctness story as enrollments).

### `audit_events` (later phase)

Append-only record of admin and sensitive reads/writes; 
can start as structured logs and promote to a table when you need queryable compliance.

### Why Postgres

Postgres gives **foreign keys, uniques, and multi-statement transactions** 
that match the enrollment problem; document stores push that complexity 
into application code and multi-step compensating logic.

---

## Summary recommendation for boilerplate

- **Go**: **chi** + **sqlc** + **migrations** + learning comments throughout the backend.
- **DB**: **Postgres** (Cloud SQL) with `users`, `classes`, `enrollments`; 
- add `class_sessions` when calendar features land.
- **Web**: **Vite + React + TypeScript**, talking to Go over HTTPS with a
-  clear auth story (JWT validation or session from your IdP).
- **GCP**: Cloud Run API, Cloud SQL, Secret Manager, CI that runs 
- **migrate + sqlc generate + test**.

**Pedagogical variant:** If you want to bias even more toward 
“pure Go learning,” use hand-written `database/sql` for the first 
iteration, then introduce **sqlc** when enroll queries stabilize.

---

## Related decisions to pin down next

These mostly affect operations and developer experience, not product features:

1. **OIDC bearer tokens vs cookie session** toward the API (CORS, XSS, mobile clients).
2. **Monorepo** (`/backend`, `/web`) vs **split repositories** for API and web.

---

## See also

- Roadmap: **Education app roadmap** plan in your Cursor plans folder (`education_app_roadmap_7917e761.plan.md`) — includes the **Boilerplate — systems design** section that mirrors this document.
