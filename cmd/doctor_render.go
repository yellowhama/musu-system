package cmd

import (
	"fmt"

	"github.com/yellowhama/musu-nurikun/internal/preflight"
)

func renderDoctorResult(result preflight.DoctorResult, project string, configPath string) {
	if result.Report.ConfigExists {
		fmt.Println("✅ Project config exists")
	} else {
		fmt.Printf("⚠️  Project config missing: %s\n", configPath)
		fmt.Printf("   Run: musu-nurikun init --project %s or use doctor --fix\n", project)
	}
	if result.Report.MailboxOK {
		fmt.Println("✅ Mailbox configuration looks complete")
	} else {
		fmt.Println("❌ Mailbox configuration issues:")
		for _, issue := range result.Report.MailboxIssues {
			fmt.Printf("   - %s\n", issue)
		}
	}
	if result.Report.KnowledgeOK {
		fmt.Println("✅ Knowledge source configuration looks complete")
	} else {
		fmt.Println("❌ Knowledge source issues:")
		for _, issue := range result.Report.KnowledgeIssues {
			fmt.Printf("   - %s\n", issue)
		}
	}
	if result.Report.PublicBaseURLOK {
		fmt.Println("✅ public_base_url configured")
	} else {
		fmt.Println("⚠️  public_base_url is empty; campaign unsubscribe/confirm links will degrade.")
	}
	if result.Report.UnsubSecretOK {
		fmt.Println("✅ unsub_secret configured")
	} else {
		fmt.Println("⚠️  unsub_secret is empty; one-click signed links are not protected.")
	}
	if result.Report.AIReachable {
		fmt.Println("✅ AI endpoint reachable")
	} else if result.Report.AIError != "" {
		fmt.Printf("❌ AI endpoint probe failed: %v\n", result.Report.AIError)
	}
}
