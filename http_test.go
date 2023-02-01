package gonertia

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsInertiaRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header http.Header
		want   bool
	}{
		{
			"positive",
			http.Header{"X-Inertia": []string{"foo"}},
			true,
		},
		{
			"negative",
			http.Header{},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest("GET", "/", nil)
			r.Header = tt.header

			got := IsInertiaRequest(r)

			if got != tt.want {
				t.Fatalf("got=%#v, want=%#v", got, tt.want)
			}
		})
	}
}
