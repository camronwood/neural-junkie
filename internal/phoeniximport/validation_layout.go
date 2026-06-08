package phoeniximport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/camronwood/neural-junkie/internal/scananalysis"
)

// ValidationSiblingDir returns the comparator sibling folder for a summary import dir.
func ValidationSiblingDir(summaryRel string) string {
	lower := strings.ToLower(summaryRel)
	const suffix = "-summary"
	if strings.HasSuffix(lower, suffix) {
		return summaryRel[:len(summaryRel)-len(suffix)] + "-validation"
	}
	return summaryRel + "-validation"
}

func defaultSummaryOutputDir(analysisID string, docRaw json.RawMessage) string {
	label := extractImportLabel(docRaw)
	if label == "" {
		return "phoenix-" + sanitizeDirName(analysisID) + "-summary"
	}
	base := sanitizeDirName(label)
	if !strings.HasSuffix(strings.ToLower(base), "-summary") {
		base += "-summary"
	}
	return base
}

func extractImportLabel(docRaw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(docRaw, &v); err != nil {
		return ""
	}
	for _, key := range []string{"plateBarcode", "plate_barcode", "analysisName", "analysis_name", "name", "title"} {
		if s := findStringField(v, key); s != "" {
			return s
		}
	}
	return ""
}

func (r *ImportResult) layoutValidationExport(
	ctx context.Context,
	client *timClient,
	root string,
	summaryRel string,
	summaryAbs string,
	analysisID string,
	attNames []string,
) {
	valRel := ValidationSiblingDir(summaryRel)
	valAbs, err := safeJoin(root, valRel)
	if err != nil {
		r.AttachmentNotes = append(r.AttachmentNotes, "validation layout: "+err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Join(valAbs, "reports"), 0o755); err != nil {
		r.AttachmentNotes = append(r.AttachmentNotes, "validation layout: "+err.Error())
		return
	}

	gotZip := false
	for _, name := range attNames {
		if !strings.EqualFold(name, "validation.zip") {
			continue
		}
		tmpZip := filepath.Join(valAbs, ".validation.zip")
		if err := client.downloadAttachment(ctx, "analyses", analysisID, name, tmpZip); err != nil {
			r.AttachmentNotes = append(r.AttachmentNotes, "validation.zip: "+err.Error())
		} else if err := unzipSafe(tmpZip, valAbs); err != nil {
			r.AttachmentNotes = append(r.AttachmentNotes, "validation.zip extract: "+err.Error())
		} else {
			gotZip = true
			r.FilesWritten = append(r.FilesWritten, filepath.Join(valRel, "(validation.zip contents)"))
		}
		_ = os.Remove(tmpZip)
		break
	}

	if !gotZip {
		if err := writeValidationCSVsFromResults(summaryAbs, valAbs); err != nil {
			r.AttachmentNotes = append(r.AttachmentNotes, "validation csv synthesis: "+err.Error())
		} else {
			r.FilesWritten = append(r.FilesWritten,
				filepath.Join(valRel, "reports", scananalysis.ValidationReportFileName),
				filepath.Join(valRel, "reports", scananalysis.AllSpotsFileName),
			)
		}
	}

	r.ValidationDir = valRel
	r.FilesWritten = append(r.FilesWritten, valRel)
}

func writeValidationCSVsFromResults(summaryAbs, validationAbs string) error {
	idx, err := scananalysis.LoadResults(summaryAbs)
	if err != nil {
		return err
	}
	doc := &idx.Document
	reportsDir := filepath.Join(validationAbs, "reports")
	if err := scananalysis.WriteValidationReportCSV(doc, filepath.Join(reportsDir, scananalysis.ValidationReportFileName)); err != nil {
		return fmt.Errorf("validation_report.csv: %w", err)
	}
	if len(doc.SpotIntensities) > 0 {
		if err := scananalysis.WriteAllSpotsCSV(doc, filepath.Join(reportsDir, scananalysis.AllSpotsFileName)); err != nil {
			return fmt.Errorf("allspots.csv: %w", err)
		}
	}
	return nil
}
