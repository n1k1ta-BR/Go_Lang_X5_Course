package analytics

type Sale struct {
	Product          string
	Category         string
	Quantity         int
	UnitPriceKopecks int64
	Cancelled        bool
}

type Report struct {
	RevenueKopecks    int64
	UnitsSold         int
	RevenueByCategory map[string]int64
	TopProduct        string
	InvalidSales      int
}

func AnalyzeSales(sales []Sale) Report {
	repo := NewReport()
	if len(sales) == 0 {
		return repo
	}
	prodCount := make(map[string]int)
	topCount := 0
	for _, sale := range sales {
		if sale.Cancelled {
			continue
		}
		if !sale.Check() {
			repo.InvalidSales++
			continue
		}

		val := int64(sale.Quantity) * sale.UnitPriceKopecks
		repo.RevenueKopecks += val
		repo.RevenueByCategory[sale.Category] += val
		repo.UnitsSold += sale.Quantity
		prodCount[sale.Product] += sale.Quantity

		if prodCount[sale.Product] > topCount || prodCount[sale.Product] == topCount && sale.Product < repo.TopProduct {
			topCount = prodCount[sale.Product]
			repo.TopProduct = sale.Product
		}
	}
	return repo
}

func NewReport() Report {
	m := make(map[string]int64)
	return Report{RevenueByCategory: m}
}

func (s *Sale) Check() bool {
	if s.Product == "" || s.Category == "" || s.Quantity <= 0 || s.UnitPriceKopecks <= 0 {
		return false
	}
	return true
}
