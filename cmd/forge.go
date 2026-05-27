package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/agent"
	"github.com/yellowhama/musu-nurikun/internal/db"
	"github.com/yellowhama/musu-nurikun/internal/identity"
	"github.com/yellowhama/musu-nurikun/internal/utils"
)

var forgeCmd = &cobra.Command{
	Use:   "forge [name]",
	Short: "Forge a new digital identity with a REAL email and a Soul",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		project := viper.GetString("project")
		dbPath := viper.GetString("db_path")
		model, _ := cmd.Flags().GetString("model")

		if dbPath == "" {
			dbPath = filepath.Join("projects", project, "data", "nurikun.db")
		}

		fmt.Printf("🎭 Forging REAL identity '%s' for project '%s'...\n", name, project)

		// 1. Create REAL Email via Mail.tm
		emailProv := identity.NewMailTMProvider()
		domain, err := emailProv.GetDomain()
		if err != nil {
			fmt.Printf("❌ Failed to get email domain: %v\n", err)
			return
		}

		emailPassword := "Nurikun!777"
		address := fmt.Sprintf("%s_%d@%s", name, time.Now().Unix()%1000, domain)
		
		fmt.Printf("📧 Requesting real email: %s...\n", address)
		email, err := emailProv.CreateRealEmail(address, emailPassword)
		if err != nil {
			fmt.Printf("❌ Failed to create real email: %v\n", err)
			return
		}
		fmt.Printf("✅ Real email account created: %s\n", email)

		// 2. Mock Phone
		phoneProv := &identity.MockProvider{}
		phone, _, _ := phoneProv.RequestPhone("reddit")
		fmt.Printf("📱 Phone reserved: %s\n", phone)

		// 3. Generate Soul (Backstory, Tone, Interests)
		fmt.Println("🧠 Phase 3: Generating a deep soul for the citizen...")
		forger := agent.NewSoulForger(model)
		soul, err := forger.GenerateSoul(name)
		if err != nil {
			fmt.Printf("   ⚠️  Soul generation failed: %v. Using placeholder.\n", err)
			soul = &agent.SoulProfile{Job: "Freelancer", Backstory: "A digital nomad."}
		} else {
			fmt.Printf("   👤 [Soul] Job: %s\n", soul.Job)
			fmt.Printf("   📝 [Soul] Tone: %s\n", soul.Tone)
		}

		// 4. Generate Avatar
		fmt.Println("🖼️  Phase 4: Generating personal avatar...")
		avatarDir := filepath.Join("projects", project, "profiles", "avatars")
		avatarPath, err := utils.GenerateAvatar(name, avatarDir)
		if err != nil {
			fmt.Printf("   ⚠️  Avatar generation failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Avatar ready: %s\n", avatarPath)
		}

		// 5. Store in DB
		store, err := db.NewStore(dbPath)
		if err != nil {
			fmt.Printf("❌ DB Error: %v\n", err)
			return
		}

		meta := db.IdentityMetadata{
			EmailPassword: emailPassword,
			Provider:      "mail.tm",
			Job:           soul.Job,
			Backstory:     soul.Backstory,
			Tone:          soul.Tone,
			Interests:     soul.Interests,
			BioShort:      soul.BioShort,
			AvatarPath:    avatarPath,
		}
		metadataJSON, _ := json.Marshal(meta)

		id, err := store.SaveIdentity(name, email, phone, string(metadataJSON))
		if err != nil {
			fmt.Printf("❌ Failed to save identity: %v\n", err)
			return
		}

		fmt.Printf("\n✨ Identity forged with REAL email and Soul (ID: %d)!\n", id)
		fmt.Printf("👉 Ready for autonomous signup.\n")
	},
}

func init() {
	forgeCmd.Flags().String("model", "llama3", "Ollama model for soul generation")
	rootCmd.AddCommand(forgeCmd)
}
