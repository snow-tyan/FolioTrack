package controllers

import "testing"

func TestValidateHoldingNumbers(t *testing.T) {
	tests := []struct {
		name       string
		market     string
		quantity   float64
		costPrice  float64
		allowClear bool
		wantErr    bool
	}{
		{
			name:      "long position in A share is valid",
			market:    "A-share",
			quantity:  100,
			costPrice: 10,
		},
		{
			name:      "negative quantity rejected outside US stock",
			market:    "HK-stock",
			quantity:  -1,
			costPrice: 20,
			wantErr:   true,
		},
		{
			name:      "US short position is valid",
			market:    "US-stock",
			quantity:  -5,
			costPrice: 180.5,
		},
		{
			name:      "US stock zero quantity rejected during edit",
			market:    "US-stock",
			quantity:  0,
			costPrice: 180.5,
			wantErr:   true,
		},
		{
			name:       "clear position allows zero quantity and zero cost",
			market:     "US-stock",
			quantity:   0,
			costPrice:  0,
			allowClear: true,
		},
		{
			name:       "clear position rejects partial zero values",
			market:     "A-share",
			quantity:   0,
			costPrice:  10,
			allowClear: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHoldingNumbers(tt.market, tt.quantity, tt.costPrice, tt.allowClear)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHoldingNumbers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
