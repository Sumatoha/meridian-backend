package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/repository"
	"github.com/xuri/excelize/v2"
)

type ExportService struct {
	queries *repository.Queries
	tierSvc *TierService
}

func NewExportService(queries *repository.Queries, tierSvc *TierService) *ExportService {
	return &ExportService{queries: queries, tierSvc: tierSvc}
}

// ExportExcel generates an Excel file for the given plan.
func (s *ExportService) ExportExcel(ctx context.Context, userID, planID uuid.UUID) ([]byte, string, error) {
	if err := s.tierSvc.CheckExport(ctx, userID); err != nil {
		return nil, "", err
	}

	plan, err := s.queries.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, "", fmt.Errorf("export excel: get plan: %w", err)
	}

	slots, err := s.queries.GetSlotsByPlanID(ctx, planID)
	if err != nil {
		return nil, "", fmt.Errorf("export excel: get slots: %w", err)
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Content Plan"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"Day", "Date", "Time", "Title", "Type", "Format", "Caption", "Hashtags", "Status"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#f0f0f0"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetRowStyle(sheet, 1, 1, headerStyle)

	// Data rows
	for i, slot := range slots {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), slot.DayNumber)
		f.SetCellValue(sheet, cellName(2, row), slot.ScheduledDate.Format("2006-01-02"))
		f.SetCellValue(sheet, cellName(3, row), slot.ScheduledTime.Format("15:04"))
		f.SetCellValue(sheet, cellName(4, row), slot.Title)
		f.SetCellValue(sheet, cellName(5, row), slot.ContentType)
		f.SetCellValue(sheet, cellName(6, row), slot.Format)
		f.SetCellValue(sheet, cellName(7, row), slot.Caption)
		f.SetCellValue(sheet, cellName(8, row), joinHashtags(slot.Hashtags))
		f.SetCellValue(sheet, cellName(9, row), slot.Status)
	}

	// Column widths
	widths := map[string]float64{"A": 6, "B": 12, "C": 8, "D": 30, "E": 14, "F": 10, "G": 50, "H": 30, "I": 12}
	for col, w := range widths {
		f.SetColWidth(sheet, col, col, w)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("export excel: write: %w", err)
	}

	filename := fmt.Sprintf("%s.xlsx", plan.Title)
	return buf.Bytes(), filename, nil
}

// ExportPDF generates an HTML page suitable for printing to PDF.
func (s *ExportService) ExportPDF(ctx context.Context, userID, planID uuid.UUID) ([]byte, string, error) {
	if err := s.tierSvc.CheckExport(ctx, userID); err != nil {
		return nil, "", err
	}

	plan, err := s.queries.GetPlanByID(ctx, planID)
	if err != nil {
		return nil, "", fmt.Errorf("export pdf: get plan: %w", err)
	}

	slots, err := s.queries.GetSlotsByPlanID(ctx, planID)
	if err != nil {
		return nil, "", fmt.Errorf("export pdf: get slots: %w", err)
	}

	type slotRow struct {
		Day       int32
		Date      string
		Time      string
		Title     string
		Type      string
		Format    string
		Caption   string
		Hashtags  string
		Status    string
	}

	rows := make([]slotRow, 0, len(slots))
	for _, slot := range slots {
		rows = append(rows, slotRow{
			Day:      slot.DayNumber,
			Date:     slot.ScheduledDate.Format("2006-01-02"),
			Time:     slot.ScheduledTime.Format("15:04"),
			Title:    slot.Title,
			Type:     slot.ContentType,
			Format:   slot.Format,
			Caption:  slot.Caption,
			Hashtags: joinHashtags(slot.Hashtags),
			Status:   slot.Status,
		})
	}

	data := struct {
		Title     string
		StartDate string
		EndDate   string
		Slots     []slotRow
	}{
		Title:     plan.Title,
		StartDate: plan.StartDate.Format("January 2, 2006"),
		EndDate:   plan.EndDate.Format("January 2, 2006"),
		Slots:     rows,
	}

	var buf bytes.Buffer
	if err := pdfTemplate.Execute(&buf, data); err != nil {
		return nil, "", fmt.Errorf("export pdf: render: %w", err)
	}

	filename := fmt.Sprintf("%s.html", plan.Title)
	return buf.Bytes(), filename, nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func joinHashtags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " "
		}
		result += "#" + tag
	}
	return result
}

var pdfTemplate = template.Must(template.New("pdf").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; font-size: 11px; color: #222; padding: 32px; }
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 4px; }
  .subtitle { color: #888; font-size: 12px; margin-bottom: 24px; }
  table { width: 100%; border-collapse: collapse; }
  th { background: #f5f5f5; text-align: left; padding: 8px 10px; font-size: 10px; text-transform: uppercase; letter-spacing: 0.05em; color: #666; border-bottom: 2px solid #e0e0e0; }
  td { padding: 8px 10px; border-bottom: 1px solid #eee; vertical-align: top; }
  tr:hover { background: #fafafa; }
  .type { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 9px; text-transform: uppercase; letter-spacing: 0.04em; }
  .type-useful { background: #dbeafe; color: #1d4ed8; }
  .type-selling { background: #fce7f3; color: #be185d; }
  .type-personal { background: #fef3c7; color: #92400e; }
  .type-entertaining { background: #d1fae5; color: #065f46; }
  .caption { max-width: 280px; word-wrap: break-word; }
  .hashtags { color: #6366f1; font-size: 10px; }
  @media print { body { padding: 16px; } }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<p class="subtitle">{{.StartDate}} &ndash; {{.EndDate}}</p>
<table>
<thead>
<tr><th>#</th><th>Date</th><th>Time</th><th>Title</th><th>Type</th><th>Format</th><th>Caption</th><th>Hashtags</th><th>Status</th></tr>
</thead>
<tbody>
{{range .Slots}}
<tr>
<td>{{.Day}}</td>
<td>{{.Date}}</td>
<td>{{.Time}}</td>
<td><strong>{{.Title}}</strong></td>
<td><span class="type type-{{.Type}}">{{.Type}}</span></td>
<td>{{.Format}}</td>
<td class="caption">{{.Caption}}</td>
<td class="hashtags">{{.Hashtags}}</td>
<td>{{.Status}}</td>
</tr>
{{end}}
</tbody>
</table>
</body>
</html>`))
