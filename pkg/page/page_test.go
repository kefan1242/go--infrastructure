package page_test

import (
	"encoding/json"
	"testing"

	"github.com/kris/go-infrastructure/pkg/page"
)

func TestParam_NormalizeDefaults(t *testing.T) {
	got := page.Param{}.Normalize()
	if got.PageNo != 1 {
		t.Errorf("PageNo: want 1, got %d", got.PageNo)
	}
	if got.PageSize != page.DefaultPageSize {
		t.Errorf("PageSize: want %d, got %d", page.DefaultPageSize, got.PageSize)
	}
}

func TestParam_NormalizeClampsMax(t *testing.T) {
	got := page.Param{PageNo: 2, PageSize: page.MaxPageSize + 500}.Normalize()
	if got.PageSize != page.MaxPageSize {
		t.Errorf("PageSize: want %d, got %d", page.MaxPageSize, got.PageSize)
	}
}

func TestParam_NormalizeNegativePageNo(t *testing.T) {
	got := page.Param{PageNo: -5, PageSize: 10}.Normalize()
	if got.PageNo != 1 {
		t.Errorf("PageNo: want 1, got %d", got.PageNo)
	}
	if got.PageSize != 10 {
		t.Errorf("PageSize: should not be touched, got %d", got.PageSize)
	}
}

func TestParam_OffsetAndLimit(t *testing.T) {
	cases := []struct {
		name       string
		p          page.Param
		wantOffset int64
		wantLimit  int64
	}{
		{"defaults", page.Param{}, 0, page.DefaultPageSize},
		{"page1", page.Param{PageNo: 1, PageSize: 10}, 0, 10},
		{"page2", page.Param{PageNo: 2, PageSize: 10}, 10, 10},
		{"page5_size50", page.Param{PageNo: 5, PageSize: 50}, 200, 50},
		{"clamped_size", page.Param{PageNo: 1, PageSize: 99999}, 0, page.MaxPageSize},
		{"neg_pageno", page.Param{PageNo: -3, PageSize: 10}, 0, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.Offset(); got != tc.wantOffset {
				t.Errorf("Offset: want %d, got %d", tc.wantOffset, got)
			}
			if got := tc.p.Limit(); got != tc.wantLimit {
				t.Errorf("Limit: want %d, got %d", tc.wantLimit, got)
			}
		})
	}
}

func TestNew_ComputesPages(t *testing.T) {
	cases := []struct {
		name      string
		listLen   int
		total     int64
		pageSize  int64
		wantPages int64
	}{
		{"exact_fit", 2, 10, 5, 2},
		{"partial_last", 1, 11, 5, 3},
		{"single_page", 3, 3, 10, 1},
		{"empty", 0, 0, 10, 0},
		{"oversize_total", 5, 100, 10, 10},
		{"pageSize_one", 1, 7, 1, 7},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := make([]int, tc.listLen)
			got := page.New(list, tc.total, tc.pageSize)
			if got.Total != tc.total {
				t.Errorf("Total: want %d, got %d", tc.total, got.Total)
			}
			if got.Pages != tc.wantPages {
				t.Errorf("Pages: want %d, got %d", tc.wantPages, got.Pages)
			}
		})
	}
}

func TestNew_NilListBecomesEmptySlice(t *testing.T) {
	got := page.New[int](nil, 0, 10)
	if got.List == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(got.List) != 0 {
		t.Errorf("expected empty, got len=%d", len(got.List))
	}
	// JSON should serialize as `[]` not `null` — stable response shape.
	b, _ := json.Marshal(got)
	if want := `"list":[]`; !contains(string(b), want) {
		t.Errorf("expected %q in JSON, got %s", want, string(b))
	}
}

func TestNew_ZeroPageSizeDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New panicked on pageSize=0: %v", r)
		}
	}()
	got := page.New([]int{1, 2, 3}, 3, 0)
	if got.Pages == 0 {
		t.Error("expected non-zero pages even on bad pageSize")
	}
}

func TestEmpty_HasZeroTotalAndEmptyList(t *testing.T) {
	r := page.Empty[string]()
	if r.Total != 0 || r.Pages != 0 {
		t.Errorf("Total/Pages: want 0/0, got %d/%d", r.Total, r.Pages)
	}
	if r.List == nil || len(r.List) != 0 {
		t.Errorf("List should be empty non-nil, got %v", r.List)
	}
}

func TestMap_TransformsListPreservesMeta(t *testing.T) {
	src := page.Result[int]{Total: 42, Pages: 5, List: []int{1, 2, 3}}
	dst := page.Map(src, func(i int) string { return strconv(i) })

	if dst.Total != 42 || dst.Pages != 5 {
		t.Errorf("meta: want total=42 pages=5, got %d/%d", dst.Total, dst.Pages)
	}
	want := []string{"1", "2", "3"}
	if len(dst.List) != len(want) {
		t.Fatalf("list len: want %d, got %d", len(want), len(dst.List))
	}
	for i, v := range dst.List {
		if v != want[i] {
			t.Errorf("list[%d]: want %q, got %q", i, want[i], v)
		}
	}
}

func TestMap_EmptyListProducesEmptyList(t *testing.T) {
	src := page.Empty[int]()
	dst := page.Map(src, func(int) string { return "x" })
	if len(dst.List) != 0 {
		t.Errorf("expected empty, got %v", dst.List)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// strconv is a tiny inline int->string to avoid pulling strconv into the test.
func strconv(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
