package update

import "testing"

func TestCompareVersionsHandlesPrereleaseOrdering(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "beta sequence", a: "0.10.0-beta.1", b: "0.10.0-beta.2", want: -1},
		{name: "beta to stable", a: "0.10.0-beta.2", b: "0.10.0", want: -1},
		{name: "stable to next beta", a: "0.10.0", b: "0.10.1-beta.1", want: -1},
		{name: "stable beats same prerelease", a: "1.0.0", b: "1.0.0-beta.9", want: 1},
		{name: "prefix ignored", a: "v1.2.3-beta.1", b: "1.2.3-beta.2", want: -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compareVersions(tc.a, tc.b); got != tc.want {
				t.Fatalf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
