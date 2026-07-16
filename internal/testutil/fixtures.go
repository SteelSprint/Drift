package testutil

// MarkerLine returns a drift marker comment line for the given shortcode.
// The literal "D!" appears only in this file, which is excluded from drift
// scanning via drift.ignore.
func MarkerLine(shortcode string) string {
	return "// D! id=" + shortcode
}
