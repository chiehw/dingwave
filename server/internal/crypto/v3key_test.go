package crypto

import "testing"

func TestSaltFromUserConfigJSON(t *testing.T) {
	// 模拟 user_config Base64 解码后的 JSON
	raw := []byte(`{"salt":"abc123","other":1}`)
	s, err := saltFromUserConfigJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s != "abc123" {
		t.Fatalf("salt = %q", s)
	}
}

func TestSaltFromUserConfigJSONSlt(t *testing.T) {
	raw := []byte(`{"slt":"xyz"}`)
	s, err := saltFromUserConfigJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s != "xyz" {
		t.Fatalf("slt = %q", s)
	}
}
