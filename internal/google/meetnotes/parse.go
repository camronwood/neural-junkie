package meetnotes

import "regexp"

var docIDRegex = regexp.MustCompile(`https://docs\.google\.com/document/d/([a-zA-Z0-9_-]+)`)

// ExtractDocID returns the Google Doc ID from email or HTML body content.
func ExtractDocID(body string) string {
	m := docIDRegex.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// DocLink builds a canonical edit URL for a document ID.
func DocLink(docID string) string {
	if docID == "" {
		return ""
	}
	return "https://docs.google.com/document/d/" + docID + "/edit"
}
