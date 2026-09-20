package renderer

import (
	"encoding/json"
	"io"

	"github.com/smford/tf-blast/pkg/analyzer"
)

// RenderJSON outputs structured JSON to io.Writer.
func RenderJSON(w io.Writer, report *analyzer.AnalysisReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
