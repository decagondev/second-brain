package canonjson

import "testing"

// A signed document's amounts and rates hash as decimal strings; integers stay numbers,
// and Encode still refuses them.
func TestEncodeDocument(t *testing.T) {
	got, err := EncodeDocument([]byte(`{"b":0.60,"a":[12.75,3,4.0,1e2]}`))
	if err != nil || string(got) != `{"a":["12.75",3,4,100],"b":"0.6"}` {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := Encode([]byte(`{"b":0.6}`)); err == nil {
		t.Fatal("Encode took a float")
	}
	if _, err := EncodeDocument([]byte(`{} {}`)); err == nil {
		t.Fatal("trailing data")
	}
}
