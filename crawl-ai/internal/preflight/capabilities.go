package preflight

import "strings"

type SourceCapability struct {
	Source           string `json:"source"`
	Supported        bool   `json:"supported"`
	AuthRequired     bool   `json:"auth_required"`
	CapabilityStatic bool   `json:"capability_static"`
	RecommendedMode  string `json:"recommended_mode,omitempty"`
}

func SourceCapabilities(sources []string) []SourceCapability {
	var report []SourceCapability
	for _, source := range sources {
		key := strings.ToLower(strings.TrimSpace(source))
		switch key {
		case "web", "yt", "gh", "arxiv", "reddit", "hf", "x":
			report = append(report, SourceCapability{
				Source:           key,
				Supported:        true,
				AuthRequired:     false,
				CapabilityStatic: true,
				RecommendedMode:  "public-fetch",
			})
		default:
			report = append(report, SourceCapability{
				Source:           key,
				Supported:        false,
				AuthRequired:     false,
				CapabilityStatic: true,
			})
		}
	}
	return report
}
