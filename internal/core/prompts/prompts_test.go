package prompts

import "testing"

func TestGet(t *testing.T) {
	p, err := Get("answer")
	if err != nil || p.Text == "" || len(p.SHA256) != 64 {
		t.Fatalf("%+v %v", p, err)
	}
	for _, bad := range []string{"../x", "nope", "set/answer", "answer.md"} {
		if _, err := Get(bad); err == nil {
			t.Errorf("Get(%q) should fail", bad)
		}
	}
	all, err := All()
	if err != nil || len(all) == 0 {
		t.Fatalf("All: %v %v", all, err)
	}
	found := false
	for i, p := range all {
		found = found || p.Name == "answer"
		if i > 0 && all[i-1].Name >= p.Name {
			t.Fatalf("All isn't sorted by name: %s before %s", all[i-1].Name, p.Name)
		}
	}
	if !found {
		t.Fatal("All has no answer prompt")
	}
}
