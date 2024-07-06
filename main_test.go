package main

import (
	"bytes"
	"testing"
)

func TestFragmentWrite_noWriter(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		frag := fragment{}
		if err := frag.write(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("map only", func(t *testing.T) {
		frag := fragment{
			m: map[string][]string{
				"1.1.1.1": {"example.com"},
			},
		}
		if err := frag.write(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFragmentWrite(t *testing.T) {
	tt := map[string]struct {
		frag fragment
		want string
	}{
		"empty": {
			frag: fragment{
				out: &bytes.Buffer{},
			},
			want: "",
		},
		"valid format": {
			frag: fragment{
				out: &bytes.Buffer{},
				m: map[string][]string{
					"2.2.2.2": {"example.com"},
					"1.1.1.1": {"example.com", "example.org"},
				},
			},
			want: "1.1.1.1 example.com,example.org\n2.2.2.2 example.com\n",
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			if err := tc.frag.write(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			got := tc.frag.out.(*bytes.Buffer).String()
			if got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
