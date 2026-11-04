// Package names classifies place names by script, so names collected in
// mixed lists can be filed as English (Latin script), Sinhala or Tamil.
package names

import "unicode"

// Script is the writing system a name is written in.
type Script int

const (
	ScriptOther Script = iota
	ScriptLatin
	ScriptSinhala
	ScriptTamil
)

// Of returns the script most of the text's letters are written in, or
// ScriptOther for text without letters or with no majority.
func Of(text string) Script {
	var latin, sinhala, tamil, other int
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Sinhala, r):
			sinhala++
		case unicode.Is(unicode.Tamil, r):
			tamil++
		case unicode.Is(unicode.Latin, r):
			latin++
		case unicode.IsLetter(r):
			other++
		}
	}
	best, count, tie := ScriptOther, other, false
	for _, candidate := range []struct {
		script Script
		count  int
	}{{ScriptLatin, latin}, {ScriptSinhala, sinhala}, {ScriptTamil, tamil}} {
		switch {
		case candidate.count > count:
			best, count, tie = candidate.script, candidate.count, false
		case candidate.count == count && count > 0:
			tie = true
		}
	}
	if tie || count == 0 {
		return ScriptOther
	}
	return best
}
