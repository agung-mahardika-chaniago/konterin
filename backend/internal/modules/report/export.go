package report

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// buildRevenueWorkbook renders the revenue report as an xlsx workbook.
func buildRevenueWorkbook(meta ExportMeta, r *RevenueReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Pendapatan"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	headers := []string{"Periode", "Jumlah Invoice", "Servis (unit)", "Jasa (Rp)", "Sparepart (Rp)",
		"Diskon (Rp)", "Total (Rp)", "Dibayar (Rp)", "Belum Dibayar (Rp)"}
	widths := []float64{16, 14, 13, 16, 16, 14, 16, 16, 18}
	if err := writeHeader(f, sheet, 1, headers, widths); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		values := []any{item.Label, item.InvoiceCount, item.ServiceCount, item.ServiceAmount,
			item.PartAmount, item.DiscountAmount, item.TotalRevenue, item.PaidAmount, item.UnpaidAmount}
		if err := writeRow(f, sheet, row, values); err != nil {
			return nil, err
		}
		row++
	}
	if err := writeRow(f, sheet, row, []any{"TOTAL", r.Total.InvoiceCount, r.Total.ServiceCount,
		r.Total.ServiceAmount, r.Total.PartAmount, r.Total.DiscountAmount,
		r.Total.TotalRevenue, r.Total.PaidAmount, r.Total.UnpaidAmount}); err != nil {
		return nil, err
	}
	boldRange(f, sheet, row, len(headers))
	return f, writeFooter(f, sheet, row+2, meta)
}

// buildServicesWorkbook renders the service report as an xlsx workbook.
func buildServicesWorkbook(meta ExportMeta, r *ServicesReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Servis"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	headers := []string{"Metrik", "Nilai"}
	if err := writeHeader(f, sheet, 1, headers, []float64{36, 18}); err != nil {
		return nil, err
	}
	rows := [][]any{
		{"Dari", r.From}, {"Sampai", r.To},
		{"Total servis", r.Total}, {"Selesai", r.Completed},
		{"Berjalan", r.InProgress}, {"Dibatalkan", r.Cancelled},
		{"Rata-rata pengerjaan (jam)", r.AvgHours},
	}
	row := 2
	for _, values := range rows {
		if err := writeRow(f, sheet, row, values); err != nil {
			return nil, err
		}
		row++
	}

	row++
	if err := writeSection(f, sheet, row, "Rincian status"); err != nil {
		return nil, err
	}
	row++
	for _, s := range r.StatusBreakdown {
		if err := writeRow(f, sheet, row, []any{s.Status, s.Count}); err != nil {
			return nil, err
		}
		row++
	}

	row++
	if err := writeSection(f, sheet, row, "Per teknisi"); err != nil {
		return nil, err
	}
	row++
	for _, t := range r.TechnicianRank {
		if err := writeRow(f, sheet, row, []any{t.Name, t.Count}); err != nil {
			return nil, err
		}
		row++
	}
	return f, writeFooter(f, sheet, row+1, meta)
}

// buildBrandsWorkbook renders the brand report as an xlsx workbook.
func buildBrandsWorkbook(meta ExportMeta, r *BrandsReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Merek"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Merek", "Jumlah Servis", "Pendapatan (Rp)"}
	if err := writeHeader(f, sheet, 1, headers, []float64{28, 14, 18}); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		if err := writeRow(f, sheet, row, []any{item.Brand, item.Count, item.Revenue}); err != nil {
			return nil, err
		}
		row++
	}
	return f, writeFooter(f, sheet, row+1, meta)
}

// buildServiceVsPartsWorkbook renders the service vs parts report.
func buildServiceVsPartsWorkbook(meta ExportMeta, r *ServiceVsPartsReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Jasa vs Part"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Jenis Item", "Kuantitas", "Nilai (Rp)"}
	if err := writeHeader(f, sheet, 1, headers, []float64{22, 14, 18}); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		if err := writeRow(f, sheet, row, []any{item.ItemType, item.Qty, item.Amount}); err != nil {
			return nil, err
		}
		row++
	}
	return f, writeFooter(f, sheet, row+1, meta)
}

// buildProfitWorkbook renders the profit report as an xlsx workbook.
func buildProfitWorkbook(meta ExportMeta, r *ProfitReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Laba"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Metrik", "Nilai"}
	if err := writeHeader(f, sheet, 1, headers, []float64{36, 20}); err != nil {
		return nil, err
	}
	rows := [][]any{
		{"Dari", r.From}, {"Sampai", r.To},
		{"Pendapatan", r.Revenue}, {"HPP (biaya part)", r.Cost},
		{"Laba kotor", r.GrossProfit}, {"Total pengeluaran", r.ExpenseTotal},
		{"Laba usaha (bersih)", r.NetProfit},
		{"Jumlah invoice", r.InvoiceCount}, {"Rata-rata invoice", r.AvgInvoice},
	}
	row := 2
	for _, values := range rows {
		if err := writeRow(f, sheet, row, values); err != nil {
			return nil, err
		}
		row++
	}
	return f, writeFooter(f, sheet, row+1, meta)
}

// buildExpensesWorkbook renders the expense report as an xlsx workbook.
func buildExpensesWorkbook(meta ExportMeta, r *ExpensesReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Pengeluaran"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Kategori", "Jumlah Transaksi", "Total (Rp)"}
	if err := writeHeader(f, sheet, 1, headers, []float64{28, 16, 18}); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		if err := writeRow(f, sheet, row, []any{item.Category, item.Count, item.Amount}); err != nil {
			return nil, err
		}
		row++
	}
	if err := writeRow(f, sheet, row, []any{"TOTAL", int64(len(r.Items)), r.Total}); err != nil {
		return nil, err
	}
	boldRange(f, sheet, row, 3)
	return f, writeFooter(f, sheet, row+2, meta)
}

// buildTechniciansWorkbook renders the technician productivity report.
func buildTechniciansWorkbook(meta ExportMeta, r *TechniciansReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Teknisi"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Teknisi", "Ditugaskan", "Selesai", "Aktif", "Rata-rata (jam)", "Kontribusi Pendapatan (Rp)"}
	if err := writeHeader(f, sheet, 1, headers, []float64{26, 12, 12, 12, 15, 24}); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		if err := writeRow(f, sheet, row, []any{item.Name, item.AssignedCount, item.CompletedCount,
			item.ActiveCount, item.AvgRepairHours, item.RevenueContrib}); err != nil {
			return nil, err
		}
		row++
	}
	return f, writeFooter(f, sheet, row+1, meta)
}

// buildStockWorkbook renders the stock report as an xlsx workbook.
func buildStockWorkbook(meta ExportMeta, r *StockReport) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "Stok"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"SKU", "Nama", "Kategori", "Stok", "Min. Stok", "Harga Modal (Rp)", "Nilai Stok (Rp)", "Stok Menipis"}
	if err := writeHeader(f, sheet, 1, headers, []float64{16, 32, 18, 10, 10, 16, 16, 12}); err != nil {
		return nil, err
	}
	row := 2
	for _, item := range r.Items {
		if err := writeRow(f, sheet, row, []any{item.SKU, item.Name, item.Category, item.Stock,
			item.MinStock, item.CostPrice, item.StockValue, yesNo(item.IsLow)}); err != nil {
			return nil, err
		}
		row++
	}
	if err := writeRow(f, sheet, row, []any{"TOTAL", "", "", "", "", "", r.TotalValue, ""}); err != nil {
		return nil, err
	}
	boldRange(f, sheet, row, 8)
	return f, writeFooter(f, sheet, row+2, meta)
}

func yesNo(value bool) string {
	if value {
		return "Ya"
	}
	return "Tidak"
}

// writeHeader writes the bold header row with autofilter and column widths.
func writeHeader(f *excelize.File, sheet string, row int, headers []string, widths []float64) error {
	for col, header := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, row)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return err
		}
	}
	boldRange(f, sheet, row, len(headers))
	for col, width := range widths {
		name, err := excelize.ColumnNumberToName(col + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, name, name, width); err != nil {
			return err
		}
	}
	if err := f.AutoFilter(sheet, fmt.Sprintf("A1:%s1", mustColumnName(len(headers))), []excelize.AutoFilterOptions{}); err != nil {
		return err
	}
	return nil
}

func writeRow(f *excelize.File, sheet string, row int, values []any) error {
	for col, value := range values {
		cell, err := excelize.CoordinatesToCellName(col+1, row)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			return err
		}
	}
	return nil
}

// writeSection writes a bold section title.
func writeSection(f *excelize.File, sheet string, row int, title string) error {
	if err := f.SetCellValue(sheet, fmt.Sprintf("A%d", row), title); err != nil {
		return err
	}
	styleID, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	return f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styleID)
}

// writeFooter writes export metadata at the bottom of the sheet.
func writeFooter(f *excelize.File, sheet string, row int, meta ExportMeta) error {
	footers := [][]string{
		{"Laporan", meta.Report},
		{"Toko", meta.StoreName},
		{"Outlet", meta.OutletName},
		{"Periode", meta.From + " s.d. " + meta.To},
		{"Dibuat oleh", meta.GeneratedBy},
		{"Dibuat pada", meta.GeneratedAt},
	}
	for i, footer := range footers {
		if footer[1] == "" && i == 2 {
			continue
		}
		if err := f.SetCellValue(sheet, fmt.Sprintf("A%d", row+i), footer[0]); err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, fmt.Sprintf("B%d", row+i), footer[1]); err != nil {
			return err
		}
	}
	return nil
}

func boldRange(f *excelize.File, sheet string, row, cols int) {
	styleID, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return
	}
	for col := 1; col <= cols; col++ {
		cell, cellErr := excelize.CoordinatesToCellName(col, row)
		if cellErr != nil {
			continue
		}
		_ = f.SetCellStyle(sheet, cell, cell, styleID)
	}
}

func mustColumnName(col int) string {
	name, err := excelize.ColumnNumberToName(col)
	if err != nil {
		return "A"
	}
	return name
}

// workbookBytes serialises the workbook to bytes for download responses.
func workbookBytes(f *excelize.File) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

