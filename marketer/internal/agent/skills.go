package agent

import (
	_ "embed"
)

//go:embed skills/MARKETING_BIBLE.md
var marketingBible string

// LoadMarketingBible returns the embedded high-resolution marketing knowledge base.
func LoadMarketingBible() (string, error) {
	return marketingBible, nil
}
