package yaml_test

import (
	"testing"

	"github.com/oasdiff/yaml"
)

type dupSpec struct {
	Info struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
}

const dupSrc = "info:\n  title: a\n  title: b\n  version: \"1\"\n"

func TestDecodeOpts_DuplicateKeys(t *testing.T) {
	var strict dupSpec
	if _, err := yaml.Unmarshal([]byte(dupSrc), &strict, yaml.DecodeOpts{}); err == nil {
		t.Fatal("default should reject a repeated key")
	}

	var lenient dupSpec
	if _, err := yaml.Unmarshal([]byte(dupSrc), &lenient,
		yaml.DecodeOpts{AllowDuplicateKeys: true}); err != nil {
		t.Fatalf("AllowDuplicateKeys should accept: %v", err)
	}
	// The last occurrence wins, as encoding/json does. Asserting the value,
	// not just the absence of an error.
	if lenient.Info.Title != "b" {
		t.Fatalf("title = %q, want %q", lenient.Info.Title, "b")
	}
	if lenient.Info.Version != "1" {
		t.Fatalf("sibling key clobbered: version = %q", lenient.Info.Version)
	}
}

// The option must survive the JSON round trip this package performs, and must
// compose with origin tracking rather than disabling it.
func TestDecodeOpts_DuplicateKeysWithOrigin(t *testing.T) {
	var v dupSpec
	tree, err := yaml.Unmarshal([]byte(dupSrc), &v, yaml.DecodeOpts{
		AllowDuplicateKeys: true,
		Origin:             yaml.OriginOpt{Enabled: true, File: "spec.yaml"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Info.Title != "b" {
		t.Fatalf("title = %q, want %q", v.Info.Title, "b")
	}
	if tree == nil {
		t.Fatal("origin tree should still be produced")
	}
}
