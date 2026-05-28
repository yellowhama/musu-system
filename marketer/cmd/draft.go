package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-marketer/internal/agent"
	"github.com/yellowhama/musu-marketer/internal/bridge"
	"github.com/yellowhama/musu-marketer/internal/db"
)

var draftCmd = &cobra.Command{
	Use:   "draft [topic]",
	Short: "Draft a viral campaign using the multi-agent Strategic Crew",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		topic := args[0]
		project := viper.GetString("project")
		wikiDir := viper.GetString("wiki_dir")
		
		dbPath := filepath.Join("projects", project, "data", "marketer.db")
		model, _ := cmd.Flags().GetString("model")
		persona, _ := cmd.Flags().GetString("persona")

		ExecuteDraft(topic, wikiDir, dbPath, model, persona, project)
	},
}

func ExecuteDraft(topic, wikiDir, dbPath, model, persona, project string) error {
	fmt.Printf("🌉 Connecting to Wiki at: %s (Project Scope: %s)\n", wikiDir, project)
	b := bridge.NewWikiBridge(wikiDir)
	
	sources, err := b.FindByTopic(topic)
	if err != nil || len(sources) == 0 {
		return fmt.Errorf("no verified knowledge found for topic: %s", topic)
	}

	fmt.Printf("📚 Found %d knowledge sources. Starting Strategic Crew...\n", len(sources))
	
	var context strings.Builder
	for _, s := range sources {
		context.WriteString(fmt.Sprintf("--- Source: %s ---\n%s\n\n", s.Title, s.Content))
	}

	// 1. Generate Strategy Brief
	fmt.Println("🧠 Phase 1: Strategist is analyzing the market...")
	
	// Fetch Social Memory
	var history strings.Builder
	store, dbErr := db.NewStore(dbPath)
	if dbErr == nil {
		defer store.Close()
		past, _ := store.GetRecentPublishedCampaigns(3)
		for _, p := range past {
			history.WriteString(fmt.Sprintf("- Topic: %s | Brief: %s\n", p.Topic, p.Brief))
		}
	}

	aiURL := viper.GetString("ai_url")
	strategist := agent.NewStrategist(aiURL, model, wikiDir, project)
	brief, err := strategist.CreateBrief(context.String(), history.String())
	if err != nil {
		fmt.Printf("   ⚠️  Strategy phase failed: %v. Using default approach.\n", err)
		brief = &agent.MarketingBrief{ValueProp: "General info", Target: "General audience", Framework: "AIDA"}
	} else {
		fmt.Printf("   🎯 [Brief] Target: %s | Goal: %s | Framework: %s\n", brief.Target, brief.Goal, brief.Framework)
	}

	// 2. Collaborative Drafting Loop (Copywriter + Critic)
	fmt.Println("✍️  Phase 2: Collaborative Drafting Loop starting...")
	projectPath := filepath.Join("projects", project)
	copywriter := agent.NewCopywriter(aiURL, model, persona, projectPath, wikiDir, project)
	critic := agent.NewCritic(aiURL, model, wikiDir, project)
	
	var finalCampaign string
	var currentFeedback string
	maxRetries := 3

	for i := 1; i <= maxRetries; i++ {
		fmt.Printf("   [Attempt %d/%d] Copywriter is drafting...\n", i, maxRetries)
		draft, err := copywriter.GenerateCampaign(topic, context.String(), brief, currentFeedback)
		if err != nil {
			return fmt.Errorf("copywriter failed: %v", err)
		}

		fmt.Printf("   [Attempt %d/%d] Critic is auditing the draft...\n", i, maxRetries)
		eval, err := critic.Evaluate(brief, copywriter.PersonaContent, draft)
		if err != nil {
			fmt.Printf("   ⚠️  Critic audit failed: %v. Proceeding with draft.\n", err)
			finalCampaign = draft
			break
		}

		if eval.Approved {
			fmt.Println("   ✅ Critic approved the draft!")
			finalCampaign = draft
			break
		}

		fmt.Printf("   ❌ Critic rejected: %s\n", eval.Feedback)
		currentFeedback = eval.Feedback
		finalCampaign = draft // Store as fallback
		if i == maxRetries {
			fmt.Println("   ⚠️  Maximum retries reached. Using the last iteration.")
		}
	}

	// 3. Save to Database
	// Fixed: reuse 'store' from above or re-open correctly
	if store == nil && dbErr != nil {
		store, err = db.NewStore(dbPath)
		if err == nil && store != nil {
			defer store.Close()
		}
	}
	
	if err == nil && store != nil {
		briefJSON, _ := json.Marshal(brief)
		id, _ := store.SaveCampaign(topic, finalCampaign, string(briefJSON), persona)
		fmt.Printf("\n✅ Strategic Campaign saved to database (ID: %d)\n", id)
	}

	fmt.Println("\n🏁 --- FINAL STRATEGIC CAMPAIGN --- 🏁")
	fmt.Println(finalCampaign)
	return nil
}

func init() {
	draftCmd.Flags().String("model", "llama3", "Ollama model for reasoning")
	draftCmd.Flags().String("persona", "default", "Persona to use for this draft")
	rootCmd.AddCommand(draftCmd)
}
