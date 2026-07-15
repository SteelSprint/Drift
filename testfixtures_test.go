// testfixtures_test.go — Test fixture strings containing drift marker syntax (#F).
//
// This file is listed in drift.ignore so the drift scanner does not treat
// these test-only marker strings as real code markers. By isolating the #F
// string construction here, the remaining *_test.go files no longer contain
// #F literals and can be tracked by the drift scanner like any other source file.
//
// HOW TO USE:
//   Call markerLine("shortcode") to get a string like "// #F shortcode".
//   Concatenate it with the rest of the fixture code:
//
//       writeCodeFile(t, dir, "main.go", markerLine("abc123")+`
//   func handleRequest() {
//       doSomething()
//   }
//   `)
//
// DO NOT write "#F" directly in other *_test.go files. Always go through
// markerLine so the literal stays confined to this file.
package driftpin

// markerLine returns a drift marker comment line for the given shortcode.
// The literal "#F" appears only in this file, which is excluded from drift
// scanning via drift.ignore.
func markerLine(shortcode string) string {
	return "// #F " + shortcode
}
