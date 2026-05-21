# Data layering

Whether to separate "DB row" / "domain object" / "API payload" depends on the
service. This doc lays out the three patterns we see in real Go codebases,
when each fits, and the one rule that always applies.

## TL;DR

| Pattern | When | What it costs |
|---|---|---|
| **A. single struct** | DB ↔ API are 1:1, no sensitive fields, small service | ORM tags pollute API docs; field type changes ripple |
| **B. PO + domain** | sensitive fields or many ORM annotations | one extra struct + a converter |
| **C. PO + domain + API DTO** | multiple API versions, multiple transports (gRPC + HTTP + internal RPC) | three structs + two converters; usually overkill until you actually need it |

`go-infrastructure/pkg` is **not opinionated** about which you pick — it
ships transport, observability, and config; the layer below the wire is
yours. This file documents the trade-offs so your team picks deliberately.

## Pattern A — one struct does double duty

```go
type User struct {
    ID       int64     `gorm:"primaryKey" json:"id"`
    Name     string    `gorm:"size:64"    json:"name"`
    Email    string    `gorm:"uniqueIndex" json:"email"`
    Password string    `gorm:"size:128"   json:"-"` // hidden from JSON
    Created  time.Time `json:"created"`
}
```

Used by: most internal tools, simple CRUD admin panels, the GORM and `ent`
documentation examples, plenty of production services where the API surface
mirrors the table.

**Drawbacks** to know going in:

- `gorm:"size:64"` shows up in Swagger / OpenAPI generators that scan struct tags.
- Schema changes are API changes. Renaming a column is a breaking client change.
- The `json:"-"` line for `Password` is the only thing keeping the hash off the wire. **One missed tag and you leak a credential**.
- Joins / computed fields force `gorm:"-"` everywhere.

Use it when the trade-offs above don't bite for your service. Don't apologize
for it — for half the services, this is the right call.

## Pattern B — DB row + domain object

```go
// internal/data/user.go     — DB shape, ORM tags live here
type userRow struct {
    ID       int64
    Name     string
    Email    string
    PwdHash  []byte
    Created  time.Time
}

func (r *userRow) toDomain() *biz.User {
    return &biz.User{ID: r.ID, Name: r.Name, Email: r.Email, Created: r.Created}
    // PwdHash deliberately not copied — domain layer can't accidentally expose it.
}
```

```go
// internal/biz/user/user.go — domain shape, no tags, no ORM imports
type User struct {
    ID      int64
    Name    string
    Email   string
    Created time.Time
}
```

The HTTP / gRPC handler returns `*biz.User` directly (or a thin wrapper).
The domain struct is the API contract.

Used by: most kratos-template services, hexagonal-style codebases,
anywhere the team wants `internal/biz` to compile without importing GORM.

This is the **sweet spot for medium-complexity services**. You pay one extra
struct + one converter per entity to get:
- ORM imports stay below the `internal/biz` boundary
- Sensitive fields physically cannot reach an API handler
- DB schema can evolve (add columns, denormalize) without touching the API

## Pattern C — DB row + domain + API DTO

```
internal/data/po/user.go        // GORM/ent struct, 1:1 with table
internal/biz/user/user.go       // pure domain, no tags
api/user/v1/user.pb.go          // proto-generated; the v1 API shape
api/user/v2/user.pb.go          // when v2 exists
```

Two converters per entity (`po ⇄ domain`, `domain ⇄ pb`).

Worth the cost when:
- You expose **multiple API versions** concurrently and they diverge.
- The same domain object goes out over **multiple transports** (e.g. gRPC for
  internal calls + REST for the web client + a Kafka event for downstream
  pipelines), each needing a different shape.
- You're large enough that contributors casually adding a column to the API
  struct is a real risk you want the compiler to catch.

For most services, C is over-engineered. The `internal/biz` struct doubling as
the API DTO (pattern B) is usually fine until proven otherwise.

## The one hard rule

> **Sensitive fields never share a struct with API output.**

Password hashes, refresh tokens, internal user IDs, billing details — these
either live on a struct that the API layer cannot import, or they get
**redacted at the type level**, not at serialization time. `json:"-"` is a
last line of defense, not a first one.

In pattern A this means *separate the sensitive struct out*. Even if 90% of
your code uses one struct, the table holding `password_hash` gets the B/C
treatment.

## Conversion conventions

When you do separate, two readable patterns:

```go
// Method on the source — fine when the destination is a simple downstream type.
func (r *userRow) toDomain() *biz.User { ... }

// Function in the target package — fine when conversion needs target-package state.
func FromRow(r *data.UserRow) *User { ... }
```

Pick one per direction and stick with it. The thing to avoid is `ConvertToDTO`,
`MapEntity`, `Build`, `Make`, `New`, `Of`, all coexisting in the same module.

## How pagination fits

`pkg/page.Result[T]` is generic over whatever you return. The pattern adapts:

```go
// Pattern A — one struct
return page.New(users, total, p.PageSize), nil   // users []*User

// Pattern B — repo returns rows, handler converts
rows, total, _ := repo.List(...)
domain := convertAll(rows)                       // []*biz.User
return page.New(domain, total, p.PageSize), nil
// or: return page.Map(rowResult, (*userRow).toDomain), nil

// Pattern C — repo → domain → DTO
return page.Map(domainResult, toAPIv1), nil
```

`page.Map` is exactly the spot where conversion happens cleanly; the metadata
(total, pages) carries through without you re-implementing it.

## Recommended starting point

If you don't know which pattern fits, **start at A**. Move to B the first time
either of the following happens:

1. You catch a sensitive field about to ship over the wire.
2. ORM struct tags start showing up in generated API docs and bother reviewers.

Don't preemptively go to C. Multi-version APIs and multi-transport fanout are
real reasons, but they're real reasons you'll know when you have them.
