package renderer

import (
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/smford/tf-blast/pkg/analyzer"
)

// RenderHTML generates a zero-dependency, self-contained, offline-ready HTML audit report.
func RenderHTML(w io.Writer, report *analyzer.AnalysisReport) error {
	if report == nil {
		return nil
	}

	healthColor := "#10b981" // green
	switch report.Summary.MaxSeverity {
	case analyzer.SeverityCritical:
		healthColor = "#ef4444" // red
	case analyzer.SeverityHigh:
		healthColor = "#f97316" // orange
	case analyzer.SeverityMedium:
		healthColor = "#eab308" // yellow
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>tf-blast &bull; Blast Radius Audit Report</title>
  <style>
    :root {
      --bg: #0f172a;
      --card-bg: #1e293b;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --border: #334155;
      --badge-critical: #ef4444;
      --badge-high: #f97316;
      --badge-medium: #eab308;
      --badge-low: #10b981;
      --badge-clean: #64748b;
    }
    @media (prefers-color-scheme: light) {
      :root {
        --bg: #f8fafc;
        --card-bg: #ffffff;
        --text: #0f172a;
        --text-muted: #64748b;
        --border: #e2e8f0;
      }
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 2rem;
    }
    .container { max-width: 1200px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--border);
    }
    .logo { font-size: 1.75rem; font-weight: 800; letter-spacing: -0.025em; display: flex; align-items: center; gap: 0.5rem; }
    .meta { font-size: 0.875rem; color: var(--text-muted); }
    .banner {
      background: var(--card-bg);
      border-left: 6px solid %s;
      padding: 1.25rem 1.5rem;
      border-radius: 8px;
      margin-bottom: 2rem;
      box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1);
    }
    .banner h2 { font-size: 1.25rem; font-weight: 700; }
    .banner p { color: var(--text-muted); margin-top: 0.25rem; }
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .metric-card {
      background: var(--card-bg);
      padding: 1.25rem;
      border-radius: 8px;
      border: 1px solid var(--border);
      text-align: center;
    }
    .metric-val { font-size: 1.75rem; font-weight: 800; }
    .metric-label { font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); margin-top: 0.25rem; }
    .controls {
      display: flex;
      gap: 1rem;
      margin-bottom: 1.5rem;
      flex-wrap: wrap;
    }
    .search-box {
      flex: 1;
      min-width: 250px;
      padding: 0.6rem 1rem;
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      color: var(--text);
      font-size: 0.95rem;
    }
    .search-box:focus { outline: 2px solid #3b82f6; }
    .filter-btn {
      padding: 0.6rem 1rem;
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 6px;
      color: var(--text);
      cursor: pointer;
      font-weight: 600;
      font-size: 0.875rem;
      transition: all 0.2s;
    }
    .filter-btn.active, .filter-btn:hover { background: #3b82f6; color: #ffffff; border-color: #3b82f6; }
    .table-wrap {
      width: 100%%;
      overflow-x: auto;
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      margin-bottom: 2rem;
    }
    table {
      width: 100%%;
      border-collapse: collapse;
      background: transparent;
      margin-bottom: 0;
      border: none;
    }
    th, td {
      padding: 0.875rem 1rem;
      text-align: left;
      border-bottom: 1px solid var(--border);
      overflow-wrap: anywhere;
      word-break: break-word;
    }
    th { background: rgba(0,0,0,0.1); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--text-muted); }
    tr:last-child td { border-bottom: none; }
    tr:hover { background: rgba(255,255,255,0.02); }
    .badge {
      display: inline-block;
      padding: 0.25rem 0.5rem;
      border-radius: 9999px;
      font-size: 0.75rem;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.025em;
      white-space: nowrap;
    }
    .badge-CRITICAL { background: #ef444422; color: #ef4444; border: 1px solid #ef4444; }
    .badge-HIGH { background: #f9731622; color: #f97316; border: 1px solid #f97316; }
    .badge-MEDIUM { background: #eab30822; color: #eab308; border: 1px solid #eab308; }
    .badge-LOW { background: #10b98122; color: #10b981; border: 1px solid #10b981; }
    .badge-CLEAN { background: #64748b22; color: #64748b; border: 1px solid #64748b; }
    .action-tag { font-family: monospace; font-weight: bold; }
    .act-REPLACE { color: #d946ef; }
    .act-DESTROY { color: #ef4444; }
    .act-UPDATE_IN_PLACE { color: #eab308; }
    .act-CREATE { color: #10b981; }
    .addr { font-family: monospace; font-size: 0.9rem; font-weight: 600; }
    .tree-box {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.5rem;
      margin-bottom: 2rem;
    }
    .tree-box h3 { margin-bottom: 1rem; font-size: 1.1rem; }
    pre { font-family: monospace; font-size: 0.875rem; line-height: 1.6; color: var(--text-muted); overflow-x: auto; }
    footer { text-align: center; color: var(--text-muted); font-size: 0.85rem; margin-top: 3rem; }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="logo">💥 tf-blast</div>
      <div class="meta">Generated: %s &bull; Zero-Trust Blast Radius Analyzer</div>
    </header>

    <div class="banner">
      <h2>Plan Health: %s</h2>
      <p>%s</p>
    </div>

    <div class="metrics-grid">
      <div class="metric-card">
        <div class="metric-val" style="color: #10b981;">%d</div>
        <div class="metric-label">To Add</div>
      </div>
      <div class="metric-card">
        <div class="metric-val" style="color: #eab308;">%d</div>
        <div class="metric-label">To Update</div>
      </div>
      <div class="metric-card">
        <div class="metric-val" style="color: #ef4444;">%d</div>
        <div class="metric-label">To Destroy</div>
      </div>
      <div class="metric-card">
        <div class="metric-val" style="color: #d946ef;">%d</div>
        <div class="metric-label">To Replace</div>
      </div>
      <div class="metric-card">
        <div class="metric-val">%d</div>
        <div class="metric-label">Blast Radius</div>
      </div>
      <div class="metric-card">
        <div class="metric-val" style="color: #f97316;">%d</div>
        <div class="metric-label">Blast Score</div>
      </div>
    </div>

    <div class="controls">
      <input type="text" id="searchInput" class="search-box" placeholder="Search address, type, or root cause...">
      <button class="filter-btn active" onclick="filterSev('ALL')">All</button>
      <button class="filter-btn" onclick="filterSev('CRITICAL')">Critical</button>
      <button class="filter-btn" onclick="filterSev('HIGH')">High</button>
      <button class="filter-btn" onclick="filterSev('MEDIUM')">Medium</button>
      <button class="filter-btn" onclick="filterSev('LOW')">Low</button>
    </div>

    <div class="table-wrap">
      <table id="resourceTable">
        <thead>
          <tr>
            <th>Severity</th>
            <th>Resource Address</th>
            <th>Action</th>
            <th>Root Cause</th>
            <th>Downstream Impact</th>
            <th>Score</th>
          </tr>
        </thead>
        <tbody>
`

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
	policyMsg := "All policy checks passed."
	if report.Failed {
		policyMsg = fmt.Sprintf("⚠️ Policy Violation: %s", report.FailReason)
	}

	fmt.Fprintf(w, htmlTemplate,
		healthColor,
		timestamp,
		html.EscapeString(report.Summary.PlanHealth),
		html.EscapeString(policyMsg),
		report.Summary.ToAdd,
		report.Summary.ToUpdate,
		report.Summary.ToDestroy,
		report.Summary.ToReplace,
		report.Summary.TotalBlastRadius,
		report.Summary.BlastScore,
	)

	// Rows
	for _, res := range report.Resources {
		addr := res.Address
		if res.HasDrift {
			addr = "⚠️ " + addr
		}

		downstreamDesc := "-"
		if res.DownstreamCount > 0 {
			downstreamDesc = fmt.Sprintf("%d dependent(s)", res.DownstreamCount)
		}

		fmt.Fprintf(w, `        <tr data-sev="%s">
          <td><span class="badge badge-%s">%s</span></td>
          <td class="addr">%s</td>
          <td class="action-tag act-%s">%s</td>
          <td>%s</td>
          <td>%s</td>
          <td><strong>%d</strong></td>
        </tr>
`,
			res.Severity,
			res.Severity,
			res.Severity,
			html.EscapeString(addr),
			res.Action,
			res.Action,
			html.EscapeString(res.RootCause),
			html.EscapeString(downstreamDesc),
			res.RiskScore,
		)
	}

	fmt.Fprintln(w, `      </tbody>
    </table>
  </div>`)

	// Impact Trees
	hasTrees := false
	for _, res := range report.Resources {
		if res.DownstreamTree != nil && len(res.DownstreamTree.Children) > 0 {
			hasTrees = true
			break
		}
	}

	if hasTrees {
		fmt.Fprintln(w, `    <div class="tree-box">
      <h3>🌳 Cascading Blast Radius Trees</h3>
      <pre>`)
		for _, res := range report.Resources {
			if res.DownstreamTree != nil && len(res.DownstreamTree.Children) > 0 {
				fmt.Fprintf(w, "%s (%s)\n", html.EscapeString(res.Address), res.Action)
				rendered := res.DownstreamTree.RenderTree()
				lines := strings.Split(rendered, "\n")
				if len(lines) > 1 {
					for _, l := range lines[1:] {
						if strings.TrimSpace(l) != "" {
							fmt.Fprintf(w, "  %s\n", html.EscapeString(l))
						}
					}
				}
				fmt.Fprintln(w)
			}
		}
		fmt.Fprintln(w, `      </pre>
    </div>`)
	}

	fmt.Fprintln(w, `    <footer>
      Generated by <a href="https://github.com/smford/tf-blast" style="color: inherit; font-weight: bold;">tf-blast</a> &bull; Zero-Trust Blast Radius Analysis
    </footer>
  </div>

  <script>
    function filterSev(sev) {
      document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.classList.toggle('active', btn.textContent.trim().toUpperCase() === sev);
      });
      const rows = document.querySelectorAll('#resourceTable tbody tr');
      rows.forEach(r => {
        const rowSev = r.getAttribute('data-sev');
        if (sev === 'ALL' || rowSev === sev) {
          r.style.display = '';
        } else {
          r.style.display = 'none';
        }
      });
    }

    document.getElementById('searchInput').addEventListener('input', function(e) {
      const term = e.target.value.toLowerCase();
      const rows = document.querySelectorAll('#resourceTable tbody tr');
      rows.forEach(r => {
        const text = r.textContent.toLowerCase();
        r.style.display = text.includes(term) ? '' : 'none';
      });
    });
  </script>
</body>
</html>`)

	return nil
}
