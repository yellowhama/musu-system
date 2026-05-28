package preflight

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corepreflight "github.com/yellowhama/musu-core/preflight"
	"github.com/yellowhama/musu-marketer/internal/bridge"
)

type DoctorOptions struct {
	Project    string
	WikiDir    string
	AIProvider string
	AIURL      string
	Topic      string
	AutoFix    bool
	FixProject func() error
}

type DoctorReport struct {
	Project           string `json:"project"`
	WikiDir           string `json:"wiki_dir"`
	AIProvider        string `json:"ai_provider"`
	AIURL             string `json:"ai_url"`
	ProjectDir        string `json:"project_dir"`
	WikiExists        bool   `json:"wiki_exists"`
	WikiMarkdownCount int    `json:"wiki_markdown_count"`
	ProjectExists     bool   `json:"project_exists"`
	DBExists          bool   `json:"db_exists"`
	AIReachable       bool   `json:"ai_reachable"`

	Topic           string `json:"topic,omitempty"`
	TopicReady      bool   `json:"topic_ready,omitempty"`
	TopicSourceCount int   `json:"topic_source_count,omitempty"`

	AIError         string `json:"ai_error,omitempty"`
	ProjectFixError string `json:"project_fix_error,omitempty"`
	TopicError      string `json:"topic_error,omitempty"`
}

type DoctorResult struct {
	Report        DoctorReport `json:"report"`
	Blocking      bool         `json:"blocking"`
	ActionableFix string       `json:"actionable_fix"`
}

func EvaluateDoctor(opts DoctorOptions) DoctorResult {
	projectDir := filepath.Join("projects", opts.Project)
	dbPath := filepath.Join(projectDir, "data", "marketer.db")
	report := DoctorReport{
		Project:           opts.Project,
		WikiDir:           opts.WikiDir,
		AIProvider:        opts.AIProvider,
		AIURL:             opts.AIURL,
		ProjectDir:        projectDir,
		WikiExists:        false,
		WikiMarkdownCount: 0,
		ProjectExists:     false,
		DBExists:          false,
		AIReachable:       false,
	}
	if strings.TrimSpace(opts.Topic) != "" {
		report.Topic = opts.Topic
		report.TopicReady = false
		report.TopicSourceCount = 0
	}

	result := DoctorResult{
		Report:        report,
		ActionableFix: "Pass --wiki explicitly or set MUSU_MARKETER_WIKI / MUSU_WIKI, run init/doctor --fix for the project scaffold, and start the configured AI endpoint.",
	}

	if info, err := os.Stat(opts.WikiDir); err == nil && info.IsDir() {
		result.Report.WikiExists = true
		result.Report.WikiMarkdownCount = countMarkdownFiles(opts.WikiDir)
	} else {
		result.Blocking = true
	}

	if _, err := os.Stat(projectDir); err != nil {
		if opts.AutoFix && opts.FixProject != nil {
			if fixErr := opts.FixProject(); fixErr == nil {
				result.Report.ProjectExists = true
				result.Report.DBExists = true
			} else {
				result.Report.ProjectFixError = fixErr.Error()
				result.Blocking = true
			}
		} else {
			result.Blocking = true
		}
	} else {
		result.Report.ProjectExists = true
	}

	if _, err := os.Stat(dbPath); err == nil {
		result.Report.DBExists = true
	} else {
		result.Blocking = true
	}

	if err := corepreflight.Probe(opts.AIURL); err != nil {
		result.Report.AIError = err.Error()
		result.Blocking = true
	} else {
		result.Report.AIReachable = true
	}

	if result.Report.Topic != "" && result.Report.WikiExists {
		b := bridge.NewWikiBridge(opts.WikiDir)
		sources, topicErr := b.FindByTopic(opts.Topic)
		if topicErr != nil {
			result.Report.TopicError = topicErr.Error()
			result.Blocking = true
		} else {
			result.Report.TopicSourceCount = len(sources)
			if len(sources) == 0 {
				result.Blocking = true
			} else {
				result.Report.TopicReady = true
			}
		}
	}

	result.ActionableFix = buildActionableFix(result.Report)

	return result
}

func buildActionableFix(report DoctorReport) string {
	var fixes []string
	if !report.WikiExists {
		fixes = append(fixes, "pass --wiki explicitly or set MUSU_MARKETER_WIKI / MUSU_WIKI")
	}
	if !report.ProjectExists || !report.DBExists {
		fixes = append(fixes, "run init or doctor --fix to recreate the project scaffold and local database")
	}
	if report.Topic != "" && !report.TopicReady {
		if report.TopicError != "" {
			fixes = append(fixes, "repair or reindex the wiki so topic lookup can read source documents")
		} else {
			fixes = append(fixes, fmt.Sprintf("add or sync wiki content for topic %q before drafting", report.Topic))
		}
	}
	if !report.AIReachable {
		fixes = append(fixes, "start the configured AI endpoint or pass a reachable --ai-url")
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
