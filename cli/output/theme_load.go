package output

// D! id=otload range-start

import (
	"encoding/xml"
	"fmt"

	"drift/internal/fileio"
)

type themeXML struct {
	XMLName  xml.Name     `xml:"theme"`
	Elements []elementXML `xml:"element"`
}

type elementXML struct {
	ID    string `xml:"id,attr"`
	Color string `xml:"color,attr"`
	Bold  bool   `xml:"bold,attr"`
	Dim   bool   `xml:"dim,attr"`
}

// ParseCustomTheme parses the bytes of a theme.xml file and returns a Theme
// built from the 18 element entries. All 18 element IDs must be present (full
// override — no inheritance from built-in themes). Returns a descriptive
// error if the XML is malformed or any element is missing.
func ParseCustomTheme(data []byte) (Theme, error) {
	var raw themeXML
	if err := xml.Unmarshal(data, &raw); err != nil {
		return Theme{}, fmt.Errorf("theme.xml: %s", err)
	}

	// Build a map of element ID → Style from the parsed XML.
	styleMap := make(map[string]Style, len(raw.Elements))
	for _, e := range raw.Elements {
		styleMap[e.ID] = Style{Color: e.Color, Bold: e.Bold, Dim: e.Dim}
	}

	// Validate all 18 elements are present.
	for _, id := range AllElementIDs {
		if _, ok := styleMap[id]; !ok {
			return Theme{}, fmt.Errorf("theme.xml: missing element %q (all 18 elements are required)", id)
		}
	}

	return Theme{
		Name:          "custom",
		MarkerID:      styleMap["marker_id"],
		SpecID:        styleMap["spec_id"],
		Filepath:      styleMap["filepath"],
		LineNumber:    styleMap["line_number"],
		Hash:          styleMap["hash"],
		StatusOK:      styleMap["status_ok"],
		StatusWarn:    styleMap["status_warn"],
		StatusError:   styleMap["status_error"],
		SectionHeader: styleMap["section_header"],
		Command:       styleMap["command"],
		Hint:          styleMap["hint"],
		DiffAdd:       styleMap["diff_add"],
		DiffRemove:    styleMap["diff_remove"],
		DiffHunk:      styleMap["diff_hunk"],
		CodeComment:   styleMap["code_comment"],
		CodeString:    styleMap["code_string"],
		CodeKeyword:   styleMap["code_keyword"],
		CodeNumber:    styleMap["code_number"],
	}, nil
}

// LoadCustomTheme reads .drift/theme.xml via the Session and parses it with
// ParseCustomTheme. Returns the Session's not-exist error if the file does
// not exist (callers treat this as "no custom theme, use built-in").
func LoadCustomTheme(sess *fileio.Session) (Theme, error) {
	data, err := sess.Read("theme.xml")
	if err != nil {
		return Theme{}, err
	}
	return ParseCustomTheme(data)
}

// D! id=otload range-end
