package services

import (
	"strings"
	"testing"
)

func TestGetSinaSymbol(t *testing.T) {
	tests := []struct {
		symbol string
		market string
		want   string
	}{
		{"600519", "A-share", "sh600519"},
		{"000001", "A-share", "sz000001"},
		{"830946", "A-share", "bj830946"},
		{"00700", "HK-stock", "rt_hk00700"},
		{"700", "HK-stock", "rt_hk00700"},
		{"AAPL", "US-stock", "gb_aapl"},
		{"110011", "Fund", "f_110011"},
	}

	for _, tt := range tests {
		got := GetSinaSymbol(tt.symbol, tt.market)
		if got != tt.want {
			t.Errorf("GetSinaSymbol(%q, %q) = %q, want %q", tt.symbol, tt.market, got, tt.want)
		}
	}
}

func TestFetchPricesFromSina(t *testing.T) {
	// Query some real symbols
	symbols := []string{"sh600519", "gb_aapl", "rt_hk00700", "f_110011"}
	prices, err := FetchPricesFromSina(symbols)
	if err != nil {
		t.Fatalf("FetchPricesFromSina failed: %v", err)
	}

	t.Logf("Fetched %d prices", len(prices))

	for sym, p := range prices {
		t.Logf("Symbol: %s, Name: %s, Price: %.4f, PrevClose: %.4f", sym, p.Name, p.CurrentPrice, p.PrevClose)
		
		if p.Name == "" {
			t.Errorf("Symbol %s has empty name", sym)
		}
		if p.CurrentPrice <= 0 {
			t.Errorf("Symbol %s has zero/negative current price: %.4f", sym, p.CurrentPrice)
		}
		if p.PrevClose <= 0 {
			t.Errorf("Symbol %s has zero/negative prev close: %.4f", sym, p.PrevClose)
		}
	}

	// Verify we got the expected tickers
	for _, sym := range symbols {
		lowerSym := strings.ToLower(sym)
		if _, ok := prices[lowerSym]; !ok {
			t.Errorf("Missing expected symbol in output: %s", sym)
		}
	}
}

func TestFetchIndexData(t *testing.T) {
	indices, err := FetchIndexData()
	if err != nil {
		t.Fatalf("FetchIndexData failed: %v", err)
	}

	t.Logf("Fetched %d indices", len(indices))
	if len(indices) != 4 {
		t.Errorf("Expected 4 indices, got %d", len(indices))
	}

	for _, idx := range indices {
		t.Logf("Index: %s (%s), Current: %v, Change: %v, ChangePct: %v", 
			idx["name"], idx["symbol"], idx["current"], idx["change"], idx["changePct"])

		if idx["name"] == "" {
			t.Errorf("Index %v has empty name", idx["symbol"])
		}
	}
}
