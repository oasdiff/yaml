package yaml

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type MarshalTest struct {
	A string
	B int64
	C float32
	D float64
}

func TestMarshal(t *testing.T) {
	f32String := strconv.FormatFloat(math.MaxFloat32, 'g', -1, 32)
	f64String := strconv.FormatFloat(math.MaxFloat64, 'g', -1, 64)
	s := MarshalTest{"a", math.MaxInt64, math.MaxFloat32, math.MaxFloat64}
	e := []byte(fmt.Sprintf("A: a\nB: %d\nC: %s\nD: %s\n", int64(math.MaxInt64), f32String, f64String))

	y, err := Marshal(s)
	if err != nil {
		t.Errorf("error marshaling YAML: %v", err)
	}

	if !reflect.DeepEqual(y, e) {
		t.Errorf("marshal YAML was unsuccessful, expected: %#v, got: %#v",
			string(e), string(y))
	}
}

type UnmarshalString struct {
	A string
	B string
}

type UnmarshalStringMap struct {
	A map[string]string
}

type UnmarshalNestedString struct {
	A NestedString
}

type NestedString struct {
	A string
}

type UnmarshalSlice struct {
	A []NestedSlice
}

type NestedSlice struct {
	B string
	C *string
}

func TestUnmarshal(t *testing.T) {
	y := []byte(``)
	s1 := UnmarshalString{}
	e1 := UnmarshalString{}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte(`{}`)
	s1 = UnmarshalString{}
	e1 = UnmarshalString{}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte("a: 1")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "1"}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte(`a: "1"`)
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "1"}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte("a: true")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "true"}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte("a: 1")
	s1 = UnmarshalString{}
	e1 = UnmarshalString{A: "1"}
	unmarshalEqual(t, y, &s1, &e1)

	y = []byte("a:\n  a: 1")
	s2 := UnmarshalNestedString{}
	e2 := UnmarshalNestedString{NestedString{"1"}}
	unmarshalEqual(t, y, &s2, &e2)

	y = []byte("a:\n  - b: abc\n    c: def\n  - b: 123\n    c: 456\n")
	s3 := UnmarshalSlice{}
	e3 := UnmarshalSlice{[]NestedSlice{{"abc", strPtr("def")}, {"123", strPtr("456")}}}
	unmarshalEqual(t, y, &s3, &e3)

	y = []byte("a:\n  b: 1")
	s4 := UnmarshalStringMap{}
	e4 := UnmarshalStringMap{map[string]string{"b": "1"}}
	unmarshalEqual(t, y, &s4, &e4)

	y = []byte(`
a:
  name: TestA
b:
  name: TestB
`)
	type NamedThing struct {
		Name string `json:"name"`
	}
	s5 := map[string]*NamedThing{}
	e5 := map[string]*NamedThing{
		"a": {Name: "TestA"},
		"b": {Name: "TestB"},
	}
	unmarshalEqual(t, y, &s5, &e5)
}

// TestUnmarshalNonStrict tests that we parse ambiguous YAML without error.
func TestUnmarshalNonStrict(t *testing.T) {
	for _, tc := range []struct {
		yaml []byte
		want UnmarshalString
	}{
		{
			yaml: []byte("a: 1"),
			want: UnmarshalString{A: "1"},
		},
		{
			// Order does not matter.
			yaml: []byte("b: 1\na: 2"),
			want: UnmarshalString{A: "2", B: "1"},
		},
		{
			// Unknown field get ignored.
			yaml: []byte("a: 1\nunknownField: 2"),
			want: UnmarshalString{A: "1"},
		},
		{
			// Unknown fields get ignored.
			yaml: []byte("unknownOne: 2\na: 1\nunknownTwo: 2"),
			want: UnmarshalString{A: "1"},
		},
		{
			// In YAML, `YES` is no longer Boolean true.
			yaml: []byte("a: YES"),
			want: UnmarshalString{A: "YES"},
		},
	} {
		s := UnmarshalString{}
		unmarshalEqual(t, tc.yaml, &s, &tc.want)
	}
}

// prettyFunctionName converts a slice of JSONOpt function pointers to a human
// readable string representation.
func prettyFunctionName(opts []JSONOpt) []string {
	var r []string
	for _, o := range opts {
		r = append(r, runtime.FuncForPC(reflect.ValueOf(o).Pointer()).Name())
	}
	return r
}

func unmarshalEqual(t *testing.T, y []byte, s, e interface{}, opts ...JSONOpt) { //nolint:unparam
	t.Helper()
	_, err := Unmarshal(y, s, DecodeOpts{}, opts...)
	if err != nil {
		t.Errorf("Unmarshal(%#q, s, %v) = %v", string(y), prettyFunctionName(opts), err)
		return
	}

	if !reflect.DeepEqual(s, e) {
		t.Errorf("Unmarshal(%#q, s, %v) = %+#v; want %+#v", string(y), prettyFunctionName(opts), s, e)
	}
}

// TestUnmarshalErrors tests that we return an error on ambiguous YAML.
func TestUnmarshalErrors(t *testing.T) {
	for _, tc := range []struct {
		yaml    []byte
		wantErr string
	}{
		{
			// Declaring `a` twice produces an error.
			yaml:    []byte("a: 1\na: 2"),
			wantErr: `key "a" already defined`,
		},
		{
			// Not ignoring first declaration of A with wrong type.
			yaml:    []byte("a: [1,2,3]\na: value-of-a"),
			wantErr: `key "a" already defined`,
		},
		{
			// Declaring field `true` twice.
			yaml:    []byte("true: string-value-of-yes\ntrue: 1"),
			wantErr: `key "true" already defined`,
		},
	} {
		s := UnmarshalString{}
		_, err := Unmarshal(tc.yaml, &s, DecodeOpts{})
		if tc.wantErr != "" && err == nil {
			t.Errorf("Unmarshal(%#q, &s) = nil; want error", string(tc.yaml))
			continue
		}
		if tc.wantErr == "" && err != nil {
			t.Errorf("Unmarshal(%#q, &s) = %v; want no error", string(tc.yaml), err)
			continue
		}
		// We only expect errors during unmarshalling YAML.
		if tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr) {
			t.Errorf("Unmarshal(%#q, &s) = %v; want err contains %#q", string(tc.yaml), err, tc.wantErr)
		}
	}
}

type Case struct {
	input  string
	output string
	// By default we test that reversing the output == input. But if there is a
	// difference in the reversed output, you can optionally specify it here.
	reverse *string
}

type RunType int

const (
	RunTypeJSONToYAML RunType = iota
	RunTypeYAMLToJSON
)

func TestJSONToYAML(t *testing.T) {
	cases := []Case{
		{
			`{"t":"a"}`,
			"t: a\n",
			nil,
		}, {
			`{"t":null}`,
			"t: null\n",
			nil,
		},
	}

	runCases(t, RunTypeJSONToYAML, cases)
}

func TestYAMLToJSON(t *testing.T) {
	cases := []Case{
		{
			"t: a\n",
			`{"t":"a"}`,
			nil,
		}, {
			"t: \n",
			`{"t":null}`,
			strPtr("t: null\n"),
		}, {
			"t: null\n",
			`{"t":null}`,
			nil,
		},
		{
			"true: yes\n",
			`{"true":"yes"}`,
			strPtr("\"true\": \"yes\"\n"),
		},
		{
			"false: yes\n",
			`{"false":"yes"}`,
			strPtr("\"false\": \"yes\"\n"),
		},
		{
			"1: a\n",
			`{"1":"a"}`,
			strPtr("\"1\": a\n"),
		}, {
			"1000000000000000000000000000000000000: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"1e+36: a\n",
			`{"1e+36":"a"}`,
			strPtr("\"1e+36\": a\n"),
		}, {
			"\"1e+36\": a\n",
			`{"1e+36":"a"}`,
			nil,
		}, {
			"\"1.2\": a\n",
			`{"1.2":"a"}`,
			nil,
		}, {
			"- t: a\n",
			`[{"t":"a"}]`,
			nil,
		}, {
			"- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n",
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			nil,
		}, {
			`[{t: a}, {t: {b: 1, c: 2}}]`,
			`[{"t":"a"},{"t":{"b":1,"c":2}}]`,
			strPtr("- t: a\n" +
				"- t:\n" +
				"    b: 1\n" +
				"    c: 2\n"),
		}, {
			"- t: \n",
			`[{"t":null}]`,
			strPtr("- t: null\n"),
		}, {
			"- t: null\n",
			`[{"t":null}]`,
			nil,
		},
	}

	// Cases that should produce errors.
	_ = []Case{
		{
			"~: a",
			`{"null":"a"}`,
			nil,
		}, {
			"a: !!binary gIGC\n",
			"{\"a\":\"\x80\x81\x82\"}",
			nil,
		},
	}

	runCases(t, RunTypeYAMLToJSON, cases)
}

func runCases(t *testing.T, runType RunType, cases []Case) {
	var f func([]byte) ([]byte, error)
	var invF func([]byte) ([]byte, error)
	var msg string
	var invMsg string
	if runType == RunTypeJSONToYAML {
		f = JSONToYAML
		invF = YAMLToJSON
		msg = "JSON to YAML"
		invMsg = "YAML back to JSON"
	} else {
		f = YAMLToJSON
		invF = JSONToYAML
		msg = "YAML to JSON"
		invMsg = "JSON back to YAML"
	}

	for _, c := range cases {
		// Convert the string.
		t.Logf("converting %s\n", c.input)
		output, err := f([]byte(c.input))
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", msg, c.input, err)
		}

		// Check it against the expected output.
		if string(output) != c.output {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				msg, c.input, c.output, string(output))
		}

		// Set the string that we will compare the reversed output to.
		reverse := c.input
		// If a special reverse string was specified, use that instead.
		if c.reverse != nil {
			reverse = *c.reverse
		}

		// Reverse the output.
		input, err := invF(output)
		if err != nil {
			t.Errorf("Failed to convert %s, input: `%s`, err: %v", invMsg, string(output), err)
		}

		// Check the reverse is equal to the input (or to *c.reverse).
		if string(input) != reverse {
			t.Errorf("Failed to convert %s, input: `%s`, expected `%s`, got `%s`",
				invMsg, string(output), reverse, string(input))
		}
	}

}

// To be able to easily fill in the *Case.reverse string above.
func strPtr(s string) *string {
	return &s
}

func TestYAMLToJSONDuplicateFields(t *testing.T) {
	data := []byte("foo: bar\nfoo: baz\n")
	if _, err := YAMLToJSON(data); err == nil {
		t.Error("expected YAMLtoJSON to fail on duplicate field names")
	}
}

func TestExtractOrigins_Map(t *testing.T) {
	input := map[string]any{
		"__origin__": map[string]any{"key": "root"},
		"info": map[string]any{
			"__origin__": map[string]any{"key": "info"},
			"title":      "Test",
		},
	}

	tree := extractOrigins(input, "")
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}

	// Root origin extracted
	if tree.Origin == nil {
		t.Fatal("expected root origin")
	}
	if tree.Origin.(map[string]any)["key"] != "root" {
		t.Error("wrong root origin")
	}

	// __origin__ removed from map
	if _, ok := input["__origin__"]; ok {
		t.Error("__origin__ not removed from root map")
	}
	infoMap := input["info"].(map[string]any)
	if _, ok := infoMap["__origin__"]; ok {
		t.Error("__origin__ not removed from info map")
	}

	// Child tree extracted
	infoTree := tree.Fields["info"]
	if infoTree == nil {
		t.Fatal("expected info child tree")
	}
	if infoTree.Origin.(map[string]any)["key"] != "info" {
		t.Error("wrong info origin")
	}
}

func TestExtractOrigins_Slice(t *testing.T) {
	input := map[string]any{
		"items": []any{
			map[string]any{
				"__origin__": map[string]any{"key": "item0"},
				"name":       "first",
			},
			"scalar",
			map[string]any{
				"__origin__": map[string]any{"key": "item2"},
				"name":       "third",
			},
		},
	}

	tree := extractOrigins(input, "")
	if tree == nil {
		t.Fatal("expected non-nil tree")
	}

	itemsTree := tree.Fields["items"]
	if itemsTree == nil {
		t.Fatal("expected items child tree")
	}
	if len(itemsTree.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(itemsTree.Items))
	}
	if itemsTree.Items[0] == nil || itemsTree.Items[0].Origin == nil {
		t.Error("expected origin for item 0")
	}
	if itemsTree.Items[1] != nil {
		t.Error("expected nil tree for scalar item 1")
	}
	if itemsTree.Items[2] == nil || itemsTree.Items[2].Origin == nil {
		t.Error("expected origin for item 2")
	}

	// __origin__ removed from slice elements
	item0 := input["items"].([]any)[0].(map[string]any)
	if _, ok := item0["__origin__"]; ok {
		t.Error("__origin__ not removed from item 0")
	}
}

func TestExtractOrigins_Nil(t *testing.T) {
	tree := extractOrigins("scalar", "")
	if tree != nil {
		t.Error("expected nil tree for scalar")
	}

	tree = extractOrigins(map[string]any{"a": "b"}, "")
	if tree != nil {
		t.Error("expected nil tree for map without __origin__")
	}
}

type OriginTreeTestStruct struct {
	Info struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
}

func TestUnmarshal_OriginTracking(t *testing.T) {
	yamlData := []byte("info:\n  title: Test\n  version: v1\n")

	var out OriginTreeTestStruct
	tree, err := Unmarshal(yamlData, &out, DecodeOpts{Origin: OriginOpt{Enabled: true, File: "test.yaml"}})
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Struct populated correctly
	if out.Info.Title != "Test" {
		t.Errorf("expected title Test, got %s", out.Info.Title)
	}

	// Origin tree returned — root mapping now gets __origin__ too (yaml3 document() fix)
	if tree == nil {
		t.Fatal("expected non-nil origin tree")
	}

	// Root mapping must have origin set
	if tree.Origin == nil {
		t.Fatal("expected root origin to be set (yaml3 document() now injects __origin__ for root mappings)")
	}

	// Info subtree with origin
	infoTree := tree.Fields["info"]
	if infoTree == nil {
		t.Fatal("expected info subtree")
	}
	if infoTree.Origin == nil {
		t.Fatal("expected info origin")
	}
	// Compact format: []any{file, key_name, key_line, key_col, nf, ...}
	originSeq, ok := infoTree.Origin.([]any)
	if !ok {
		t.Fatalf("expected []any origin, got %T", infoTree.Origin)
	}
	if len(originSeq) < 5 {
		t.Errorf("expected at least 5 elements in origin sequence, got %d", len(originSeq))
	}
	if originSeq[0] != "test.yaml" {
		t.Errorf("expected file 'test.yaml' at index 0, got %v", originSeq[0])
	}
	if originSeq[1] != "info" {
		t.Errorf("expected key_name 'info' at index 1, got %v", originSeq[1])
	}
}

// TestUnmarshal_OriginTracking_StandaloneSchema verifies that a YAML document
// that IS itself a schema (no wrapping key — the $ref pattern) produces a
// non-nil root Origin in the OriginTree. This exercises the yaml3 fix that
// injects __origin__ for the root mapping in document().
func TestUnmarshal_OriginTracking_StandaloneSchema(t *testing.T) {
	// Simulates data/ref-chain-example/base/schemas/pet.yaml
	yamlData := []byte("type: object\nrequired:\n  - id\n  - name\n")

	var out map[string]any
	tree, err := Unmarshal(yamlData, &out, DecodeOpts{Origin: OriginOpt{Enabled: true, File: "pet.yaml"}})
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected non-nil origin tree")
	}
	// Root schema Origin must now be set (yaml3 document() fix)
	if tree.Origin == nil {
		t.Fatal("expected root origin to be set for standalone schema ($ref-root fix)")
	}
	// Compact format: [file, key_name, key_line, key_col, nf, fields..., ns, seqs...]
	originSeq, ok := tree.Origin.([]any)
	if !ok {
		t.Fatalf("expected []any origin, got %T", tree.Origin)
	}
	if len(originSeq) < 6 {
		t.Errorf("expected at least 6 elements, got %d", len(originSeq))
	}
	if originSeq[0] != "pet.yaml" {
		t.Errorf("expected file 'pet.yaml', got %v", originSeq[0])
	}
	// key_name must be empty string (synthetic root key has no name)
	if originSeq[1] != "" {
		t.Errorf("expected empty key_name for root, got %v", originSeq[1])
	}
	// 'required' sequence items must be in the tree for source tracking
	if tree.Fields != nil {
		t.Errorf("expected no sub-fields in a standalone scalar-only schema, but got: %v", tree.Fields)
	}
}

func TestUnmarshal_OriginTracking_Disabled(t *testing.T) {
	yamlData := []byte("info:\n  title: Test\n")

	var out OriginTreeTestStruct
	tree, err := Unmarshal(yamlData, &out, DecodeOpts{Origin: OriginOpt{Enabled: false}})
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if tree != nil {
		t.Error("expected nil tree when origin is disabled")
	}
	if out.Info.Title != "Test" {
		t.Errorf("expected title Test, got %s", out.Info.Title)
	}
}

func TestDecodeOpts_DisableTimestamps(t *testing.T) {
	y := []byte("1344-08-22: hello\n")

	// Without DisableTimestamps the underlying YAML decoder resolves the
	// date-shaped map key to a time.Time, which the JSON-conversion step
	// then rejects as an unsupported map key type. Both claims are pinned
	// by asserting on the error message itself: it names the type
	// (time.Time) and the rejection reason (unsupported map key).
	_, err := Unmarshal(y, &map[string]string{}, DecodeOpts{})
	if err == nil {
		t.Fatalf("expected default Unmarshal to fail on a date-shaped map key, got nil error")
	}
	if !strings.Contains(err.Error(), "time.Time") {
		t.Fatalf("expected error to mention time.Time (the resolved key type), got: %v", err)
	}
	if !strings.Contains(err.Error(), "unsupported map key") {
		t.Fatalf("expected error to mention unsupported map key (the JSON-conversion rejection), got: %v", err)
	}

	// With DisableTimestamps=true, the same input parses cleanly: the key
	// stays a string and round-trips through to the target map.
	out := map[string]string{}
	tree, err := Unmarshal(y, &out, DecodeOpts{DisableTimestamps: true})
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if tree != nil {
		t.Fatalf("expected nil OriginTree (Origin disabled), got %+v", tree)
	}
	if got := out["1344-08-22"]; got != "hello" {
		t.Fatalf("expected map[\"1344-08-22\"]=\"hello\", got map=%v", out)
	}
}
