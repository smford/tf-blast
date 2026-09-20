package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/smford/tf-blast/pkg/analyzer"
)

// RenderTerminal outputs a rich ANSI colorized report to the provided io.Writer.
func RenderTerminal(w io.Writer, report *analyzer.AnalysisReport) error {
	if report == nil {
		return nil
	}

	// Disable color if stdout is piped or color.NoColor is set
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	magenta := color.New(color.FgMagenta, color.Bold).SprintFunc()
	whiteBold := color.New(color.FgWhite, color.Bold).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	// 1. Banner
	fmt.Fprintln(w)
	var healthBadge string
	switch report.Summary.MaxSeverity {
	case analyzer.SeverityCritical:
		healthBadge = red("🚨 Plan Health: " + report.Summary.PlanHealth)
	case analyzer.SeverityHigh:
		healthBadge = yellow("⚠️  Plan Health: " + report.Summary.PlanHealth)
	case analyzer.SeverityMedium:
		healthBadge = magenta("⚡ Plan Health: " + report.Summary.PlanHealth)
	default:
		healthBadge = green("✅ Plan Health: " + report.Summary.PlanHealth)
	}
	fmt.Fprintf(w, "%s\n\n", healthBadge)

	// 2. Metrics Row
	metrics := fmt.Sprintf("%s %d to add %s %s %d to update %s %s %d to destroy %s %s %d to replace",
		green("+"), report.Summary.ToAdd, gray("|"),
		yellow("~"), report.Summary.ToUpdate, gray("|"),
		red("-"), report.Summary.ToDestroy, gray("|"),
		magenta("±"), report.Summary.ToReplace,
	)
	fmt.Fprintln(w, metrics)
	fmt.Fprintf(w, "%s %d resource(s) impacted", gray("Total Blast Radius:"), report.Summary.TotalBlastRadius)
	if report.Summary.DriftCount > 0 {
		fmt.Fprintf(w, "  %s %s", gray("|"), yellow(fmt.Sprintf("⚠️  %d resource(s) with out-of-band drift", report.Summary.DriftCount)))
	}
	fmt.Fprintf(w, "\n\n")

	if len(report.Resources) == 0 {
		fmt.Fprintln(w, gray("No modified resources found in plan."))
		return nil
	}

	// 3. Impact Table Header
	fmt.Fprintf(w, "%-10s  %-38s  %-12s  %-26s  %s\n",
		whiteBold("SEVERITY"),
		whiteBold("RESOURCE ADDRESS"),
		whiteBold("ACTION"),
		whiteBold("ROOT CAUSE"),
		whiteBold("DOWNSTREAM"),
	)
	fmt.Fprintln(w, strings.Repeat("─", 100))

	// 4. Rows
	for _, res := range report.Resources {
		sevCol := formatSeverityTerminal(res.Severity, red, yellow, magenta, green, gray)
		actCol := formatActionTerminal(res.Action, red, yellow, magenta, green, gray)

		addr := res.Address
		if res.HasDrift {
			addr = "⚠️ " + addr
		}
		if len(addr) > 38 {
			addr = addr[:35] + "..."
		}

		rc := res.RootCause
		if len(rc) > 26 {
			rc = rc[:23] + "..."
		}

		downstreamStr := "-"
		if res.DownstreamCount > 0 {
			downstreamStr = fmt.Sprintf("%d dependent(s)", res.DownstreamCount)
			if res.DownstreamCount >= 5 {
				downstreamStr = red(downstreamStr)
			}
		}

		fmt.Fprintf(w, "%-10s  %-38s  %-12s  %-26s  %s\n",
			sevCol,
			addr,
			actCol,
			rc,
			downstreamStr,
		)
	}
	fmt.Fprintln(w)

	// 5. Downstream impact trees for notable resources
	hasTrees := false
	for _, res := range report.Resources {
		if res.DownstreamTree != nil && len(res.DownstreamTree.Children) > 0 {
			if !hasTrees {
				fmt.Fprintln(w, whiteBold("🌳 Cascading Blast Radius Trees:"))
				fmt.Fprintln(w)
				hasTrees = true
			}
			fmt.Fprintf(w, "%s (%s)\n", whiteBold(res.Address), string(res.Action))
			renderedTree := res.DownstreamTree.RenderTree()
			// Skip the first line since we already printed the root
			lines := strings.Split(renderedTree, "\n")
			if len(lines) > 1 {
				for _, line := range lines[1:] {
					if strings.TrimSpace(line) != "" {
						fmt.Fprintf(w, "  %s\n", line)
					}
				}
			}
			fmt.Fprintln(w)
		}
	}

	// 6. Fail notification if policy failed
	if report.Failed {
		fmt.Fprintf(w, "%s %s\n\n", red("✖ Policy Check FAILED:"), report.FailReason)
	}

	return nil
}

func formatSeverityTerminal(
	s analyzer.Severity,
	red, yellow, magenta, green, gray func(...any) string,
) string {
	switch s {
	case analyzer.SeverityCritical:
		return red("CRITICAL")
	case analyzer.SeverityHigh:
		return yellow("HIGH    ")
	case analyzer.SeverityMedium:
		return magenta("MEDIUM  ")
	case analyzer.SeverityLow:
		return gray("LOW     ")
	default:
		return gray("CLEAN   ")
	}
}

func formatActionTerminal(
	a analyzer.ActionType,
	red, yellow, magenta, green, gray func(...any) string,
) string {
	switch a {
	case analyzer.ActionReplace:
		return magenta("REPLACE     ")
	case analyzer.ActionDestroy:
		return red("DESTROY     ")
	case analyzer.ActionUpdateInPlace:
		return yellow("UPDATE      ")
	case analyzer.ActionCreate:
		return green("CREATE      ")
	default:
		return gray(string(a))
	}
}
