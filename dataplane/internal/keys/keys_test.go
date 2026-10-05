package keys

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCrossLanguageVectors(t *testing.T) {
	raw, err := os.ReadFile("../../../proto/testdata/keys.json")
	if err != nil {
		t.Fatal(err)
	}
	var vs []struct{ Private, Public string }
	if err := json.Unmarshal(raw, &vs); err != nil {
		t.Fatal(err)
	}
	for _, v := range vs {
		priv, err := Parse(v.Private)
		if err != nil {
			t.Fatal(err)
		}
		if got := priv.Public().String(); got != v.Public {
			t.Fatalf("public mismatch: go=%s python=%s", got, v.Public)
		}

	}
}
