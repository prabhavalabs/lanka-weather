package names

import "testing"

func TestOf(t *testing.T) {
	cases := map[string]Script{
		"Kandy":  ScriptLatin,
		"මහනුවර": ScriptSinhala,
		"ශ්‍රී ජයවර්ධනපුර": ScriptSinhala,
		"கண்டி":           ScriptTamil,
		"Kandy கண்டி கண்": ScriptTamil,
		"8/1":             ScriptOther,
		"":                ScriptOther,
		"Москва":          ScriptOther,
	}
	for text, want := range cases {
		if got := Of(text); got != want {
			t.Errorf("Of(%q) = %v, want %v", text, got, want)
		}
	}
}
