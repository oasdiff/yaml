package yaml

import (
	"fmt"
	"strings"
	"testing"
)

func TestRejectCollidingJSONMapKeys(t *testing.T) {
	for _, test := range []struct {
		name, input, key string
	}{
		{"fuzz input", "  0:\n  7:\n  .0:\n   02", "0"},
		{"integer and string", "0x1: integer\n'1': string\n", "1"},
		{"boolean and string", "TRUE: boolean\n\"true\": string\n", "true"},
		{"nested mapping", "nested: {1.0: number, '1': string}\n", "1"},
		{"sequence mapping", "- {1.0: number, '1': string}\n", "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			want := fmt.Sprintf("multiple YAML keys map to JSON key %q", test.key)
			for _, origin := range []bool{false, true} {
				t.Run(fmt.Sprintf("origin=%t", origin), func(t *testing.T) {
					var out any
					_, err := Unmarshal([]byte(test.input), &out, DecodeOpts{Origin: OriginOpt{Enabled: origin}})
					if err == nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("Unmarshal error = %v, want %q", err, want)
					}
				})
			}
			if _, err := YAMLToJSON([]byte(test.input)); err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("YAMLToJSON error = %v, want %q", err, want)
			}
		})
	}
}

func TestDistinctJSONMapKeys(t *testing.T) {
	for _, test := range []struct {
		name, input, want string
	}{
		{"numeric keys", "1: one\n2: two\n", `{"1":"one","2":"two"}`},
		{"distinct strings", "'0': one\n'0.0': two\n", `{"0":"one","0.0":"two"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, origin := range []bool{false, true} {
				var out any
				if _, err := Unmarshal([]byte(test.input), &out, DecodeOpts{Origin: OriginOpt{Enabled: origin}}); err != nil {
					t.Fatalf("Unmarshal with origin=%t: %v", origin, err)
				}
			}
			out, err := YAMLToJSON([]byte(test.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != test.want {
				t.Fatalf("YAMLToJSON = %s, want %s", out, test.want)
			}
		})
	}
}
