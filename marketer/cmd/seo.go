package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/marketer/internal/seo"
)

var seoCmd = &cobra.Command{
	Use:   "seo [keyword]",
	Short: "Generate an SEO longform blog post grounded in the wiki, with a strict citation gate",
	Long: `Generate a publishable Korean SEO blog post for a keyword.

The post is grounded ONLY in verified wiki sources. Every legal/factual claim
must carry a [S#] citation; in --strict mode (default) the post is BLOCKED from
being written if any factual claim is uncited or cites a non-existent source.
This is the "no hallucinated law" gate — a v2 healthy-marketing guarantee.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		project := viper.GetString("project")
		wikiDir := viper.GetString("wiki_dir")
		aiURL := viper.GetString("ai_url")
		model, _ := cmd.Flags().GetString("model")
		strict, _ := cmd.Flags().GetBool("strict")
		minWords, _ := cmd.Flags().GetInt("min-words")
		outDir, _ := cmd.Flags().GetString("out")

		fmt.Printf("🌉 Wiki: %s (project: %s) | strict=%v\n", wikiDir, project, strict)
		gen := seo.NewGenerator(aiURL, model, wikiDir, project, minWords)

		fmt.Printf("🔎 Grounding + outlining keyword: %q\n", keyword)
		art, rep, err := gen.Generate(keyword, strict)
		if err != nil {
			// In strict mode a citation failure lands here. Surface the report so
			// the operator sees exactly which claims were uncited, then exit non-zero.
			if art != nil {
				fmt.Fprintf(os.Stderr, "\n📋 Citation report: %d cited, %d uncited, %d invalid labels\n",
					rep.CitedClaims, len(rep.UncitedClaims), len(rep.InvalidLabels))
			}
			return err
		}

		fmt.Printf("✅ Citation gate passed: %d cited claims, %d sources\n",
			rep.CitedClaims, len(art.Pack.Sources))

		dir := filepath.Join("projects", project, outDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create out dir %s: %w", dir, err)
		}
		path := filepath.Join(dir, art.Outline.Slug+".md")
		if err := os.WriteFile(path, []byte(art.Render()), 0o644); err != nil {
			return fmt.Errorf("write article: %w", err)
		}

		fmt.Printf("📝 Saved: %s\n", path)
		fmt.Printf("   title: %s\n   slug: %s\n", art.Outline.Title, art.Outline.Slug)
		return nil
	},
}

func init() {
	seoCmd.Flags().String("model", "llama3", "Ollama model for reasoning")
	seoCmd.Flags().Bool("strict", true, "block output if any factual claim is uncited (recommended)")
	seoCmd.Flags().Int("min-words", 900, "minimum target word count for the body")
	seoCmd.Flags().String("out", "blog", "output subdirectory under projects/<project>/")
	rootCmd.AddCommand(seoCmd)
}
