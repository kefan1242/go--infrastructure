// Package page provides standard request/response types for paginated
// endpoints. Keep API shapes consistent across services without locking
// callers into a particular ORM or data layer.
//
// Usage on the request side:
//
//	type ListUsersReq struct {
//	    page.Param
//	    NameLike string `json:"name_like"`
//	}
//
//	func (h *Handler) List(ctx context.Context, req *ListUsersReq) (*page.Result[UserDTO], error) {
//	    p := req.Param.Normalize()
//	    users, total, err := h.repo.List(ctx, req.NameLike, p.Offset(), p.Limit())
//	    if err != nil { return nil, err }
//	    return ptr(page.New(users, total, p.PageSize)), nil
//	}
package page

// Defaults applied when Param fields are zero / out of range. Override in
// your handler if your endpoint warrants a different policy.
const (
	DefaultPageSize int64 = 20
	MaxPageSize     int64 = 1000
)

// Param is the standard request shape for paginated endpoints.
// PageNo is 1-based. Zero / negative values normalize to defaults.
type Param struct {
	PageNo   int64 `json:"page_no"   form:"page_no"`
	PageSize int64 `json:"page_size" form:"page_size"`
}

// Normalize returns a copy of p with PageNo>=1 and PageSize in
// [1, MaxPageSize]. Call this in handlers before deriving offsets.
func (p Param) Normalize() Param {
	if p.PageNo < 1 {
		p.PageNo = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
	return p
}

// Offset returns the SQL OFFSET (0-based) implied by p, using normalized
// values. Safe to call on an un-normalized Param.
func (p Param) Offset() int64 {
	n := p.Normalize()
	return (n.PageNo - 1) * n.PageSize
}

// Limit returns the SQL LIMIT implied by p, using normalized values.
func (p Param) Limit() int64 {
	return p.Normalize().PageSize
}

// Result is the standard response shape for paginated endpoints.
type Result[T any] struct {
	Total int64 `json:"total"`
	Pages int64 `json:"pages"`
	List  []T   `json:"list"`
}

// New builds a Result. pageSize is the page size that was actually used to
// fetch (so pages reflects what the client asked for, not the slice length).
// A nil list becomes an empty slice for stable JSON output.
func New[T any](list []T, total, pageSize int64) Result[T] {
	if pageSize <= 0 {
		pageSize = 1
	}
	pages := total / pageSize
	if total%pageSize != 0 {
		pages++
	}
	if list == nil {
		list = []T{}
	}
	return Result[T]{Total: total, Pages: pages, List: list}
}

// Empty returns a zero-content paginated result. Use when a guard short-
// circuits the query (e.g. unauthorized scope) — keeps the response shape
// consistent so callers don't need a special branch.
func Empty[T any]() Result[T] {
	return Result[T]{List: []T{}}
}

// Map transforms each element of r.List via fn and returns a new Result
// carrying the same Total/Pages. Useful when the repo returns DB entities
// but the API needs DTOs.
//
//	dto := page.Map(entityResult, func(e *Entity) *DTO { return e.toDTO() })
func Map[S, T any](r Result[S], fn func(S) T) Result[T] {
	out := make([]T, len(r.List))
	for i, s := range r.List {
		out[i] = fn(s)
	}
	return Result[T]{Total: r.Total, Pages: r.Pages, List: out}
}
