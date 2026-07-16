package testutil

// MarkerLine returns a drift range-start marker comment line for the given shortcode.
// The literal "D!" appears only in this file, which is excluded from drift
// scanning via drift.ignore.
func MarkerLine(shortcode string) string {
	return "// D! id=" + shortcode + " range-start"
}

// MarkerStart returns a drift range-start marker comment line for the given shortcode.
func MarkerStart(shortcode string) string {
	return "// D! id=" + shortcode + " range-start"
}

// MarkerEnd returns a drift range-end marker comment line for the given shortcode.
func MarkerEnd(shortcode string) string {
	return "// D! id=" + shortcode + " range-end"
}
