package explain

import "testing"

func TestFirstDivergentStreetName(t *testing.T) {
	cases := []struct {
		name              string
		chosen, reference []string
		want              string
	}{
		{
			name:      "diverges at named street",
			chosen:    []string{"A1", "Local Road", "Main St"},
			reference: []string{"A1", "A1", "A1"},
			want:      "Local Road",
		},
		{
			name:      "diverges at unnamed maneuver",
			chosen:    []string{"A1", ""},
			reference: []string{"A1", "A1"},
			want:      "jednoj deonici puta",
		},
		{
			name:      "identical - no divergence found within overlap",
			chosen:    []string{"A1", "A1"},
			reference: []string{"A1", "A1"},
			want:      "jednoj deonici puta",
		},
		{
			name:      "empty inputs",
			chosen:    nil,
			reference: nil,
			want:      "jednoj deonici puta",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := firstDivergentStreetName(c.chosen, c.reference)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
