package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/smford/tf-blast/pkg/analyzer"
	"github.com/smford/tf-blast/pkg/config"
	"github.com/smford/tf-blast/pkg/graph"
	"github.com/smford/tf-blast/pkg/parser"
	"github.com/smford/tf-blast/pkg/renderer"
)

var (
	Version   = "1.0.0"
	CommitSHA = ""

	filePath          string
	configPath        string
	outputFormat      string
	outFile           string
	failOn            string
	maxBlast          int
	maxScore          int
	silent            bool
	showVersion       bool
	jsonFlag          bool
	failOnDestroy     bool
	failOnReplacement bool
	maxSeverity       string
	interactive       bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tf-blast [flags] [plan-file.json...]",
		Short: "Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu",
		Long: `tf-blast analyzes Terraform and OpenTofu execution plans to provide semantic,
graph-aware blast-radius analysis. Instead of dumping thousands of lines of raw text,
it parses the plan's underlying dependency graph and resource changes, calculates the
blast radius of destructive modifications, and outputs both an interactive terminal
report and a GitHub/GitLab-ready Markdown summary for pull requests.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(getVersionString())
				return nil
			}

			// 1. Resolve Plan Inputs (flags, multiple positional args, or stdin)
			var targetFiles []string
			if filePath != "" {
				targetFiles = append(targetFiles, filePath)
			}
			for _, arg := range args {
				if filePath != "" && arg == filePath {
					continue
				}
				targetFiles = append(targetFiles, arg)
			}

			var plan *parser.Plan
			if len(targetFiles) == 0 {
				stat, err := os.Stdin.Stat()
				if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
					fmt.Fprintln(os.Stderr, "Error: no plan JSON provided via file argument or standard input.")
					fmt.Fprintln(os.Stderr, "Usage: tf-blast [flags] [plan-file.json...] or cat plan.json | tf-blast")
					os.Exit(2)
				}
				p, err := parser.ParsePlan(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing plan JSON from stdin: %v\n", err)
					os.Exit(2)
				}
				plan = p
			} else if len(targetFiles) == 1 {
				targetFile := targetFiles[0]
				f, err := os.Open(targetFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error opening plan file %s: %v\n", targetFile, err)
					os.Exit(2)
				}
				p, err := parser.ParsePlan(f)
				f.Close()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing plan JSON %s: %v\n", targetFile, err)
					os.Exit(2)
				}
				displayName := parser.PlanDisplayName(targetFile)
				for i := range p.ResourceChanges {
					p.ResourceChanges[i].PlanSource = displayName
				}
				plan = p
			} else {
				// Multi-plan aggregation
				namedPlans := make([]parser.NamedPlan, 0, len(targetFiles))
				for _, targetFile := range targetFiles {
					f, err := os.Open(targetFile)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error opening plan file %s: %v\n", targetFile, err)
						os.Exit(2)
					}
					p, err := parser.ParsePlan(f)
					f.Close()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error parsing plan JSON %s: %v\n", targetFile, err)
						os.Exit(2)
					}
					namedPlans = append(namedPlans, parser.NamedPlan{
						Name: parser.PlanDisplayName(targetFile),
						Plan: p,
					})
				}
				plan = parser.AggregatePlans(namedPlans)
			}

			// 2. Load Configuration
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
				os.Exit(2)
			}

			// Apply CLI flag overrides to config
			if failOn != "" {
				cfg.FailOn = failOn
			}
			if failOnDestroy {
				cfg.FailOn = "any-destroy"
			}
			if failOnReplacement {
				cfg.FailOn = "replacement"
			}
			if maxSeverity != "" {
				cfg.FailOn = maxSeverity
			}
			if maxBlast > 0 {
				cfg.MaxBlast = maxBlast
			}
			if maxScore > 0 {
				cfg.MaxScore = maxScore
			}

			// 3. Build Graph & Analyze
			g := graph.BuildGraph(plan)
			report := analyzer.Analyze(plan, g, cfg)

			// 4. Determine Output Format
			format := strings.ToLower(strings.TrimSpace(outputFormat))
			if jsonFlag {
				format = "json"
			}
			if format == "" {
				format = "terminal"
			}

			// 5. Write to OutFile if specified
			if outFile != "" {
				outF, err := os.Create(outFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating output file %s: %v\n", outFile, err)
					os.Exit(2)
				}
				defer outF.Close()

				// If writing to file, infer format if not explicitly set to a specific non-terminal format
				fileFormat := format
				if format == "terminal" {
					if strings.HasSuffix(outFile, ".md") {
						fileFormat = "markdown"
					} else if strings.HasSuffix(outFile, ".json") {
						fileFormat = "json"
					} else if strings.HasSuffix(outFile, ".sarif") {
						fileFormat = "sarif"
					} else if strings.HasSuffix(outFile, ".html") || strings.HasSuffix(outFile, ".htm") {
						fileFormat = "html"
					} else if strings.HasSuffix(outFile, ".mermaid") || strings.HasSuffix(outFile, ".mmd") {
						fileFormat = "mermaid"
					}
				}

				if err := renderReport(outF, fileFormat, report); err != nil {
					fmt.Fprintf(os.Stderr, "Error rendering to file: %v\n", err)
					os.Exit(2)
				}
			}

			// 6. Write to Terminal / Stdout if not silent
			if !silent {
				if interactive {
					if err := renderer.RenderInteractive(report); err != nil {
						fmt.Fprintf(os.Stderr, "Error rendering interactive TUI: %v\n", err)
						os.Exit(2)
					}
				} else {
					if err := renderReport(os.Stdout, format, report); err != nil {
						fmt.Fprintf(os.Stderr, "Error rendering output: %v\n", err)
						os.Exit(2)
					}
				}
			}

			// 7. Exit Code Decision
			if report.Failed {
				os.Exit(1)
			}

			return nil
		},
	}

	rootCmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to terraform/opentofu JSON plan file (default: stdin)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Optional path to .tf-blast.yaml policy config")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "terminal", "Output format: terminal, markdown, json, mermaid, sarif, html")
	rootCmd.Flags().StringVar(&outFile, "out-file", "", "Path to write output to (e.g., pr-comment.md, report.sarif, report.html)")
	rootCmd.Flags().StringVar(&outFile, "output-file", "", "Alias for --out-file")
	rootCmd.Flags().StringVar(&failOn, "fail-on", "", `Exit code 1 threshold: "critical", "high", "replacement", "any-destroy"`)
	rootCmd.Flags().StringVar(&maxSeverity, "max-severity", "", "Exit code 1 threshold based on max severity: critical, high, medium")
	rootCmd.Flags().BoolVar(&failOnDestroy, "fail-on-destroy", false, "Exit code 1 if any resource is destroyed or replaced")
	rootCmd.Flags().BoolVar(&failOnReplacement, "fail-on-replacement", false, "Exit code 1 if any resource is replaced")
	rootCmd.Flags().IntVar(&maxBlast, "max-blast", 0, "Maximum acceptable blast radius before failing (default: 0 = disabled)")
	rootCmd.Flags().IntVar(&maxScore, "max-score", 0, "Maximum acceptable weighted blast score before failing (default: 0 = disabled)")
	rootCmd.Flags().BoolVarP(&silent, "silent", "s", false, "Suppress terminal output (useful in CI when only checking exit codes)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information")
	rootCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output in raw JSON format (equivalent to -o json)")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Launch interactive terminal TUI explorer")

	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newDiffCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

func getVersionString() string {
	v := Version
	if !strings.HasPrefix(v, "v") && v != "dev" {
		v = "v" + v
	}
	sha := CommitSHA
	if sha == "" || sha == "none" || sha == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					sha = setting.Value
					break
				}
			}
		}
	}
	if sha != "" && sha != "none" && sha != "unknown" {
		if len(sha) > 7 {
			sha = sha[:7]
		}
		return fmt.Sprintf("tf-blast version %s (commit: %s)", v, sha)
	}
	return fmt.Sprintf("tf-blast version %s", v)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the tf-blast version and commit SHA",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(getVersionString())
		},
	}
}

func newDiffCmd() *cobra.Command {
	var diffOutput string
	var diffOutFile string

	cmd := &cobra.Command{
		Use:   "diff [flags] <before-plan.json> <after-plan.json>",
		Short: "Compare two execution plans and report blast-radius / risk deltas",
		Long: `Compare a baseline Terraform plan against a revision to evaluate how blast radius,
resource replacements, and risk scores evolved between commits or iterations.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f1, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("error opening before-plan %s: %w", args[0], err)
			}
			defer f1.Close()
			p1, err := parser.ParsePlan(f1)
			if err != nil {
				return fmt.Errorf("error parsing before-plan %s: %w", args[0], err)
			}

			f2, err := os.Open(args[1])
			if err != nil {
				return fmt.Errorf("error opening after-plan %s: %w", args[1], err)
			}
			defer f2.Close()
			p2, err := parser.ParsePlan(f2)
			if err != nil {
				return fmt.Errorf("error parsing after-plan %s: %w", args[1], err)
			}

			cfg, _ := config.LoadConfig(configPath)
			g1 := graph.BuildGraph(p1)
			r1 := analyzer.Analyze(p1, g1, cfg)

			g2 := graph.BuildGraph(p2)
			r2 := analyzer.Analyze(p2, g2, cfg)

			diff := analyzer.DiffReports(r1, r2)

			format := strings.ToLower(strings.TrimSpace(diffOutput))
			if format == "" {
				format = "terminal"
			}

			if diffOutFile != "" {
				outF, err := os.Create(diffOutFile)
				if err != nil {
					return fmt.Errorf("error creating output file %s: %w", diffOutFile, err)
				}
				defer outF.Close()
				fileFormat := format
				if format == "terminal" && strings.HasSuffix(diffOutFile, ".md") {
					fileFormat = "markdown"
				} else if format == "terminal" && strings.HasSuffix(diffOutFile, ".json") {
					fileFormat = "json"
				}
				if err := renderDiffReport(outF, fileFormat, diff); err != nil {
					return err
				}
			}

			if !silent {
				return renderDiffReport(os.Stdout, format, diff)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&diffOutput, "output", "o", "terminal", "Output format: terminal, markdown, json")
	cmd.Flags().StringVar(&diffOutFile, "out-file", "", "Path to write diff output to (e.g. diff.md)")
	return cmd
}

func renderDiffReport(w io.Writer, format string, diff *analyzer.DiffReport) error {
	switch format {
	case "markdown", "md":
		return renderer.RenderDiffMarkdown(w, diff)
	case "json":
		return renderer.RenderDiffJSON(w, diff)
	case "terminal":
		return renderer.RenderDiffTerminal(w, diff)
	default:
		return fmt.Errorf("unknown diff format: %s (must be terminal, markdown, or json)", format)
	}
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `To load completions:

Bash:
  $ source <(tf-blast completion bash)

Zsh:
  $ tf-blast completion zsh > "${fpath[1]}/_tf-blast"

Fish:
  $ tf-blast completion fish | source

PowerShell:
  PS> tf-blast completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}

func renderReport(w io.Writer, format string, report *analyzer.AnalysisReport) error {
	switch format {
	case "markdown", "md":
		return renderer.RenderMarkdown(w, report)
	case "json":
		return renderer.RenderJSON(w, report)
	case "terminal":
		return renderer.RenderTerminal(w, report)
	case "mermaid", "mmd":
		return renderer.RenderMermaid(w, report)
	case "sarif":
		return renderer.RenderSARIF(w, report, Version)
	case "html", "htm":
		return renderer.RenderHTML(w, report)
	default:
		return fmt.Errorf("unknown output format: %s (must be terminal, markdown, json, mermaid, sarif, or html)", format)
	}
}
