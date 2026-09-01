package api

import "testing"

func TestClientPageQueryUsesDeskLink149CurrentParameter(t *testing.T) {
	query := ClientPageQuery{Page: 9, Current: 3, PageSize: 100}
	if got := query.PageNumber(); got != 3 {
		t.Fatalf("PageNumber() = %d, want current=3", got)
	}
	if got := query.Limit(); got != 100 {
		t.Fatalf("Limit() = %d, want 100", got)
	}
}

func TestClientPageQueryKeepsLegacyPageFallback(t *testing.T) {
	query := ClientPageQuery{Page: 2}
	if got := query.PageNumber(); got != 2 {
		t.Fatalf("PageNumber() = %d, want page=2", got)
	}
	if got := query.Limit(); got != 100 {
		t.Fatalf("Limit() = %d, want default 100", got)
	}
}
