package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Manage marketing personas",
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available personas for the current project",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		personaDir := filepath.Join("projects", project, "personas")
		
		files, err := os.ReadDir(personaDir)
		if err != nil {
			fmt.Printf("❌ Failed to read personas directory: %v\n", err)
			return
		}

		fmt.Printf("\n🎭 --- AVAILABLE PERSONAS (Project: %s) --- 🎭\n", project)
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".md" {
				fmt.Printf("* %s\n", strings.TrimSuffix(f.Name(), ".md"))
			}
		}
	},
}

var personaShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Display the profile of a specific persona",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		path := filepath.Join("projects", project, "personas", args[0]+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ Persona '%s' not found in project '%s'.\n", args[0], project)
			return
		}
		fmt.Printf("\n📄 --- PERSONA PROFILE: %s (Project: %s) --- 📄\n", args[0], project)
		fmt.Println(string(data))
	},
}

var personaCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new persona using an interactive wizard",
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		reader := bufio.NewReader(os.Stdin)

		fmt.Printf("\n✨ --- PERSONA CREATION WIZARD (Project: %s) --- ✨\n", project)

		fmt.Print("1. Persona Name (e.g. tech-guru): ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)

		fmt.Print("2. Role (e.g. Senior Cloud Architect): ")
		role, _ := reader.ReadString('\n')
		role = strings.TrimSpace(role)

		fmt.Print("3. Tone of Voice (e.g. Sarcastic, Professional): ")
		tone, _ := reader.ReadString('\n')
		tone = strings.TrimSpace(tone)

		fmt.Print("4. Preferred Framework (AIDA or PAS): ")
		framework, _ := reader.ReadString('\n')
		framework = strings.ToUpper(strings.TrimSpace(framework))

		fmt.Print("5. Target Audience (e.g. Startup Founders): ")
		audience, _ := reader.ReadString('\n')
		audience = strings.TrimSpace(audience)

		content := fmt.Sprintf(`# Role: %s
# Tone: %s
# Framework: %s
# Audience: %s

## System Instructions
- Adapt all technical content to appeal to the %s.
- Maintain a %s tone throughout.
- Use the %s logic to drive engagement.
`, role, tone, framework, audience, audience, tone, framework)

		path := filepath.Join("projects", project, "personas", name+".md")
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			fmt.Printf("❌ Failed to save persona: %v\n", err)
			return
		}

		fmt.Printf("\n✅ Persona '%s' created successfully at %s!\n", name, path)
	},
}

var personaDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete an existing persona",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		project := viper.GetString("project")
		path := filepath.Join("projects", project, "personas", args[0]+".md")
		if err := os.Remove(path); err != nil {
			fmt.Printf("❌ Failed to delete persona '%s' from project '%s': %v\n", args[0], project, err)
			return
		}
		fmt.Printf("✅ Persona '%s' deleted from project '%s'.\n", args[0], project)
	},
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaShowCmd)
	personaCmd.AddCommand(personaCreateCmd)
	personaCmd.AddCommand(personaDeleteCmd)
	rootCmd.AddCommand(personaCmd)
}
