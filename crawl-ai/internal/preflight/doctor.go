package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corepreflight "github.com/yellowhama/musu-core/preflight"
)

type DoctorOptions struct {
	Out               string
	Project           string
	AIProvider        string
	AIURL             string
	CapabilitySources []string
	AutoFix           bool
	FixScaffold       func() error
}

type DoctorReport struct {
	WikiDir            string             `json:"wiki_dir"`
	Project            string             `json:"project"`
	AIProvider         string             `json:"ai_provider"`
	AIURL              string             `json:"ai_url"`
	WikiExists         bool               `json:"wiki_exists"`
	WikiMarkdownCount  int                `json:"wiki_markdown_count"`
	IndexExists        bool               `json:"index_exists"`
	ProjectExists      bool               `json:"project_exists"`
	AIReachable        bool               `json:"ai_reachable"`
	SourceCapabilities []SourceCapability `json:"source_capabilities,omitempty"`

	AIError      string `json:"ai_error,omitempty"`
	WikiFixError string `json:"wiki_fix_error,omitempty"`
}

type DoctorResult struct {
	Report        DoctorReport `json:"report"`
	Blocking      bool         `json:"blocking"`
	ActionableFix string       `json:"actionable_fix"`
}

func EvaluateDoctor(opts DoctorOptions) DoctorResult {
	report := DoctorReport{
		WikiDir:           opts.Out,
		Project:           opts.Project,
		AIProvider:        opts.AIProvider,
		AIURL:             opts.AIURL,
		WikiExists:        false,
		WikiMarkdownCount: 0,
		IndexExists:       false,
		ProjectExists:     false,
		AIReachable:       false,
	}
	if len(opts.CapabilitySources) > 0 {
		report.SourceCapabilities = SourceCapabilities(opts.CapabilitySources)
	}
	result := DoctorResult{
		Report:        report,
		ActionableFix: "Initialize the wiki/project scaffold or start the configured AI endpoint. Treat source_capabilities as static setup metadata only.",
	}

	if info, err := os.Stat(opts.Out); err == nil && info.IsDir() {
		result.Report.WikiExists = true
		result.Report.WikiMarkdownCount = countMarkdownFiles(opts.Out)
	} else if opts.AutoFix && opts.FixScaffold != nil {
		if fixErr := opts.FixScaffold(); fixErr == nil {
			result.Report.WikiExists = true
			result.Report.ProjectExists = true
		} else {
			result.Report.WikiFixError = fixErr.Error()
			result.Blocking = true
		}
	} else {
		result.Blocking = true
	}

	blevePath := filepath.Join(opts.Out, "musu.bleve")
	if _, err := os.Stat(blevePath); err == nil {
		result.Report.IndexExists = true
	}

	if opts.Project != "" && opts.Project != "all" && opts.Project != "default" {
		projectDir := filepath.Join(opts.Out, "projects", opts.Project)
		if _, err := os.Stat(projectDir); err == nil {
			result.Report.ProjectExists = true
		} else {
			result.Blocking = true
		}
	}

	if err := corepreflight.Probe(opts.AIURL); err != nil {
		result.Report.AIError = err.Error()
		result.Blocking = true
	} else {
		result.Report.AIReachable = true
	}

	result.ActionableFix = buildActionableFix(result.Report)

	return result
}

func buildActionableFix(report DoctorReport) string {
	var fixes []string
	if !report.WikiExists {
		fixes = append(fixes, "run init or doctor --fix to create the wiki scaffold")
	}
	if report.Project != "" && report.Project != "all" && report.Project != "default" && !report.ProjectExists {
		fixes = append(fixes, fmt.Sprintf("create or sync wiki/projects/%s before project-scoped runs", report.Project))
	}
	if !report.AIReachable {
		fixes = append(fixes, "start the configured AI endpoint or pass a reachable --ai-url")
	}
	if len(report.SourceCapabilities) > 0 {
		fixes = append(fixes, "treat source_capabilities as static setup metadata only")
	}
	if len(fixes) == 0 {
		return "No action required."
	}
	return strings.Join(fixes, "; ") + "."
}

func countMarkdownFiles(root string) int {
	count := 0
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			count++
		}
		return nil
	})
	return count
}
