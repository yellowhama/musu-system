package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		personaSlug, _ := cmd.Flags().GetString("persona")
		allPersonas, _ := cmd.Flags().GetBool("all-personas")

		fmt.Printf("🌉 Wiki: %s (project: %s) | strict=%v\n", wikiDir, project, strict)
		gen := seo.NewGenerator(aiURL, model, wikiDir, project, minWords)
		dir := filepath.Join("projects", project, outDir)

		// Multi-persona fan-out: one grounded fact base, many honest framings.
		if allPersonas {
			fmt.Printf("🔎 Grounding %q → %d persona variants\n", keyword, len(seo.DefaultPersonas))
			results := gen.GenerateVariants(keyword, seo.DefaultPersonas, strict)
			saved, failed := 0, 0
			for _, v := range results {
				if v.Err != nil {
					failed++
					fmt.Fprintf(os.Stderr, "❌ %s (%s): %v [%d uncited, %d invalid]\n",
						v.Persona.Name, v.Persona.Slug, v.Err,
						len(v.Report.UncitedClaims), len(v.Report.InvalidLabels))
					continue
				}
				path, err := saveArticle(dir, v.Article)
				if err != nil {
					return err
				}
				saved++
				fmt.Printf("✅ %s → %s (%d cited)\n", v.Persona.Name, path, v.Report.CitedClaims)
			}
			fmt.Printf("📦 %d saved, %d blocked by citation gate\n", saved, failed)
			if saved == 0 {
				return fmt.Errorf("no persona variant passed the citation gate")
			}
			return nil
		}

		// Single article: generic, or one named persona.
		var (
			art *seo.Article
			rep seo.CitationReport
			err error
		)
		if personaSlug != "" {
			p, ok := seo.PersonaBySlug(personaSlug)
			if !ok {
				return fmt.Errorf("unknown persona %q (available: %s)",
					personaSlug, strings.Join(seo.PersonaSlugs(), ", "))
			}
			fmt.Printf("🔎 Grounding + outlining %q for persona %s\n", keyword, p.Name)
			art, rep, err = gen.GenerateForPersona(keyword, p, strict)
		} else {
			fmt.Printf("🔎 Grounding + outlining keyword: %q\n", keyword)
			art, rep, err = gen.Generate(keyword, strict)
		}
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
		path, err := saveArticle(dir, art)
		if err != nil {
			return err
		}
		fmt.Printf("📝 Saved: %s\n", path)
		fmt.Printf("   title: %s\n   slug: %s\n", art.Outline.Title, art.Outline.Slug)
		return nil
	},
}

// saveArticle writes a rendered article under dir/<slug>.md and returns the path.
func saveArticle(dir string, art *seo.Article) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create out dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, art.Outline.Slug+".md")
	if err := os.WriteFile(path, []byte(art.Render()), 0o644); err != nil {
		return "", fmt.Errorf("write article: %w", err)
	}
	return path, nil
}

func init() {
	seoCmd.Flags().String("model", defaultLocalModel, "Ollama model for reasoning")
	seoCmd.Flags().Bool("strict", true, "block output if any factual claim is uncited (recommended)")
	seoCmd.Flags().Int("min-words", 900, "minimum target word count for the body")
	seoCmd.Flags().String("out", "blog", "output subdirectory under projects/<project>/")
	seoCmd.Flags().String("persona", "", "generate one audience variant (slug: elderly-owner|heir|returning-farmer|prospective-buyer)")
	seoCmd.Flags().Bool("all-personas", false, "fan out the keyword across all NJD personas")
	rootCmd.AddCommand(seoCmd)
}
