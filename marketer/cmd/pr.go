package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-system/marketer/internal/pr"
)

var prCmd = &cobra.Command{
	Use:   "pr [keyword]",
	Short: "Generate a gated press pitch DRAFT for an agricultural outlet (human sends it)",
	Long: `Generate a press pitch draft grounded ONLY in verified wiki sources.

Every factual/numeric/legal claim in the body must carry a [S#] citation; in
--strict mode (default) the pitch is BLOCKED if any claim is uncited or cites a
non-existent source — you never hand a journalist a fabricated statistic.

The pitch is a DRAFT written to disk for review. It is NEVER sent automatically;
outbound stays human-in-the-loop.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := args[0]
		project := viper.GetString("project")
		wikiDir := viper.GetString("wiki_dir")
		aiURL := viper.GetString("ai_url")
		model, _ := cmd.Flags().GetString("model")
		strict, _ := cmd.Flags().GetBool("strict")
		outDir, _ := cmd.Flags().GetString("out")
		outletSlug, _ := cmd.Flags().GetString("outlet")
		allOutlets, _ := cmd.Flags().GetBool("all-outlets")

		fmt.Printf("🌉 Wiki: %s (project: %s) | strict=%v | draft-only (발송 안 함)\n", wikiDir, project, strict)
		gen := pr.NewPitchGenerator(aiURL, model, wikiDir, project)
		dir := filepath.Join("projects", project, outDir)

		var outlets []pr.Outlet
		if allOutlets {
			outlets = pr.DefaultOutlets
		} else {
			o, ok := pr.OutletBySlug(outletSlug)
			if !ok {
				return fmt.Errorf("unknown outlet %q (available: %s)",
					outletSlug, strings.Join(pr.OutletSlugs(), ", "))
			}
			outlets = []pr.Outlet{o}
		}

		saved, failed := 0, 0
		for _, o := range outlets {
			fmt.Printf("🔎 Drafting pitch for %s …\n", o.Name)
			pitch, rep, err := gen.Generate(keyword, o, strict)
			if err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "❌ %s: %v [%d uncited, %d invalid]\n",
					o.Name, err, len(rep.UncitedClaims), len(rep.InvalidLabels))
				continue
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create out dir %s: %w", dir, err)
			}
			path := filepath.Join(dir, pitch.Slug()+".md")
			if err := os.WriteFile(path, []byte(pitch.Render()), 0o644); err != nil {
				return fmt.Errorf("write pitch: %w", err)
			}
			saved++
			fmt.Printf("✅ %s → %s (%d cited)\n", o.Name, path, rep.CitedClaims)
		}
		fmt.Printf("📦 %d draft(s) saved, %d blocked by citation gate\n", saved, failed)
		if saved == 0 {
			return fmt.Errorf("no pitch passed the citation gate")
		}
		return nil
	},
}

func init() {
	prCmd.Flags().String("model", defaultLocalModel, "Ollama model for reasoning")
	prCmd.Flags().Bool("strict", true, "block output if any factual claim is uncited (recommended)")
	prCmd.Flags().String("out", "pr", "output subdirectory under projects/<project>/")
	prCmd.Flags().String("outlet", "nongmin", "target outlet slug (nongmin|aflnews|aflnnews)")
	prCmd.Flags().Bool("all-outlets", false, "draft a pitch for every default outlet")
	rootCmd.AddCommand(prCmd)
}
