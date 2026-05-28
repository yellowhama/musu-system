package cmd

import (
	"fmt"

	"github.com/yellowhama/musu-marketer/internal/preflight"
)

func renderDoctorResult(result preflight.DoctorResult, wikiDir string, topic string) {
	fmt.Printf("Project Dir  : %s\n", result.Report.ProjectDir)
	if result.Report.WikiExists {
		fmt.Printf("✅ Wiki directory exists (%d markdown files)\n", result.Report.WikiMarkdownCount)
	} else {
		fmt.Printf("❌ Wiki directory not found. Expected a musu-crawl-ai wiki at %s\n", wikiDir)
	}
	if result.Report.ProjectExists {
		fmt.Println("✅ Project directory exists")
	}
	if result.Report.DBExists {
		fmt.Println("✅ Project database exists")
	}
	if result.Report.AIReachable {
		fmt.Println("✅ AI endpoint reachable")
	} else if result.Report.AIError != "" {
		fmt.Printf("❌ AI endpoint probe failed: %v\n", result.Report.AIError)
	}
	if topic != "" {
		if result.Report.TopicReady {
			fmt.Printf("✅ Topic %q matched %d wiki source(s)\n", topic, result.Report.TopicSourceCount)
		} else if result.Report.TopicError != "" {
			fmt.Printf("❌ Topic lookup failed: %s\n", result.Report.TopicError)
		} else {
			fmt.Printf("⚠️  No wiki sources matched topic %q\n", topic)
		}
	}
}
