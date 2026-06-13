package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"foliotrack/models"
	"io"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// ExportHoldingsCSV generates a CSV file content representing the user's holdings
func ExportHoldingsCSV(userID uint) ([]byte, error) {
	var holdings []models.Holding
	err := models.DB.Preload("Asset").Preload("Account").Preload("Combos").Where("user_id = ?", userID).Find(&holdings).Error
	if err != nil {
		return nil, err
	}

	// 1. Fetch live prices & exchange rates to get accurate valuations for sorting
	var assets []*models.Asset
	assetMap := make(map[uint]*models.Asset)
	for i := range holdings {
		h := &holdings[i]
		if _, ok := assetMap[h.AssetID]; !ok {
			assetMap[h.AssetID] = &h.Asset
			assets = append(assets, &h.Asset)
		}
	}
	if len(assets) > 0 {
		_ = UpdateAssetPrices(models.DB, assets, false, false)
		PopulateAssetCurrencyAndRates(assets)
	}
	// Re-assign updated assets back to holdings
	for i := range holdings {
		h := &holdings[i]
		if updated, ok := assetMap[h.AssetID]; ok {
			h.Asset = *updated
		}
	}

	// 2. Prepare items for sorting
	type exportHoldingSortItem struct {
		holding      models.Holding
		valCNY       float64
		primaryCombo string
	}

	var sortItems []exportHoldingSortItem
	for _, h := range holdings {
		valCNY := h.Quantity * h.Asset.CurrentPrice * h.Asset.ExchangeRate

		primaryCombo := ""
		if len(h.Combos) > 0 {
			// Sort combos of the holding by name to be deterministic
			sort.Slice(h.Combos, func(i, j int) bool {
				return h.Combos[i].Name < h.Combos[j].Name
			})
			primaryCombo = h.Combos[0].Name
		}

		sortItems = append(sortItems, exportHoldingSortItem{
			holding:      h,
			valCNY:       valCNY,
			primaryCombo: primaryCombo,
		})
	}

	// 3. Group by Market
	marketGroups := make(map[string][]exportHoldingSortItem)
	var markets []string
	marketSeen := make(map[string]bool)

	for _, item := range sortItems {
		m := item.holding.Asset.Market
		marketGroups[m] = append(marketGroups[m], item)
		if !marketSeen[m] {
			marketSeen[m] = true
			markets = append(markets, m)
		}
	}

	// Sort markets: A-share -> Fund -> HK-stock -> US-stock -> others
	marketOrder := map[string]int{
		"A-share":  1,
		"Fund":     2,
		"HK-stock": 3,
		"US-stock": 4,
	}
	sort.Slice(markets, func(i, j int) bool {
		ordI, okI := marketOrder[markets[i]]
		ordJ, okJ := marketOrder[markets[j]]
		if okI && okJ {
			return ordI < ordJ
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return markets[i] < markets[j]
	})

	// 4. Sort holdings in each market group
	var sortedItems []exportHoldingSortItem
	for _, m := range markets {
		items := marketGroups[m]

		// Calculate total valuation for each primary combo in this market
		comboValuations := make(map[string]float64)
		for _, item := range items {
			if item.primaryCombo != "" {
				comboValuations[item.primaryCombo] += item.valCNY
			}
		}

		sort.Slice(items, func(i, j int) bool {
			itemI := items[i]
			itemJ := items[j]

			// Group by primary combo: same combos together
			if itemI.primaryCombo != "" && itemJ.primaryCombo != "" {
				if itemI.primaryCombo == itemJ.primaryCombo {
					// Within the same combo, sort by individual holding valuation descending
					return itemI.valCNY > itemJ.valCNY
				}
				// Different combos: sort by total combo valuation descending
				valI := comboValuations[itemI.primaryCombo]
				valJ := comboValuations[itemJ.primaryCombo]
				if valI != valJ {
					return valI > valJ
				}
				return itemI.primaryCombo < itemJ.primaryCombo
			}

			// If one has no combo, put it at the end of the combos
			if itemI.primaryCombo != "" && itemJ.primaryCombo == "" {
				return true
			}
			if itemI.primaryCombo == "" && itemJ.primaryCombo != "" {
				return false
			}

			// Both have no combo: sort by individual holding valuation descending
			return itemI.valCNY > itemJ.valCNY
		})

		sortedItems = append(sortedItems, items...)
	}

	// 5. Generate CSV output
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write BOM for Excel compatibility in Chinese environments
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	// Header row
	err = writer.Write([]string{"symbol", "market", "account_name", "name", "quantity", "cost_price", "combos"})
	if err != nil {
		return nil, err
	}

	for _, item := range sortedItems {
		h := item.holding
		var comboNames []string
		for _, c := range h.Combos {
			comboNames = append(comboNames, c.Name)
		}

		row := []string{
			h.Asset.Symbol,
			h.Asset.Market,
			h.Account.Name,
			h.Asset.Name,
			strconv.FormatFloat(h.Quantity, 'f', -1, 64),
			strconv.FormatFloat(h.CostPrice, 'f', -1, 64),
			strings.Join(comboNames, ";"),
		}

		err = writer.Write(row)
		if err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), nil
}

// ImportHoldingsCSV processes an uploaded CSV file containing holdings
func ImportHoldingsCSV(userID uint, reader io.Reader) error {
	csvReader := csv.NewReader(reader)

	// Read header
	header, err := csvReader.Read()
	if err != nil {
		return err
	}

	// Map header indices
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	requiredFields := []string{"symbol", "market", "quantity", "cost_price"}
	for _, field := range requiredFields {
		if _, ok := headerMap[field]; !ok {
			return fmt.Errorf("CSV format error: missing required column '%s'", field)
		}
	}

	// Read all rows
	rows, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	var importedAssets []*models.Asset

	// Transaction to import rows
	err = models.DB.Transaction(func(tx *gorm.DB) error {
		for i, row := range rows {
			symbol := strings.TrimSpace(row[headerMap["symbol"]])
			market := strings.TrimSpace(row[headerMap["market"]])
			accountName := ""
			if accountIndex, ok := headerMap["account_name"]; ok && accountIndex < len(row) {
				accountName = strings.TrimSpace(row[accountIndex])
			}
			quantityStr := strings.TrimSpace(row[headerMap["quantity"]])
			costPriceStr := strings.TrimSpace(row[headerMap["cost_price"]])

			if symbol == "" || market == "" {
				continue // skip empty rows
			}
			if !IsValidMarket(market) {
				return fmt.Errorf("row %d: unsupported market '%s'", i+2, market)
			}

			quantity, err := strconv.ParseFloat(quantityStr, 64)
			if err != nil {
				return fmt.Errorf("row %d: invalid quantity '%s'", i+2, quantityStr)
			}

			costPrice, err := strconv.ParseFloat(costPriceStr, 64)
			if err != nil {
				return fmt.Errorf("row %d: invalid cost_price '%s'", i+2, costPriceStr)
			}

			account, err := ResolveAccountByName(tx, userID, market, accountName)
			if err != nil {
				return err
			}

			// 1. Find or create Asset
			var asset models.Asset
			err = tx.Where("symbol = ? AND market = ?", symbol, market).First(&asset).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				asset = models.Asset{
					Symbol: symbol,
					Market: market,
					Name:   symbol, // Temporary fallback name
				}
				if err := tx.Create(&asset).Error; err != nil {
					return err
				}
				// Fetch metadata from Sina immediately using the transaction, skipping immediate Redis write
				_ = UpdateAssetPrices(tx, []*models.Asset{&asset}, true, true)
			} else if err != nil {
				return err
			}

			// 2. Find or create Holding
			var holding models.Holding
			err = tx.Where("user_id = ? AND account_id = ? AND asset_id = ?", userID, account.ID, asset.ID).First(&holding).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				holding = models.Holding{
					UserID:    userID,
					AccountID: account.ID,
					AssetID:   asset.ID,
					Quantity:  quantity,
					CostPrice: costPrice,
				}
				if err := tx.Create(&holding).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// Update existing holding quantity and cost
				holding.Quantity = quantity
				holding.CostPrice = costPrice
				if err := tx.Save(&holding).Error; err != nil {
					return err
				}
			}

			// 3. Clear existing combo associations for this holding before re-associating
			if err := tx.Model(&holding).Association("Combos").Clear(); err != nil {
				return err
			}

			// 4. Handle Combos association
			if comboIndex, ok := headerMap["combos"]; ok {
				combosStr := strings.TrimSpace(row[comboIndex])
				if combosStr != "" {
					comboNames := strings.Split(combosStr, ";")
					var combosToLink []models.Combo

					for _, cName := range comboNames {
						cName = strings.TrimSpace(cName)
						if cName == "" {
							continue
						}

						var combo models.Combo
						err = tx.Where("user_id = ? AND name = ? AND market = ?", userID, cName, asset.Market).First(&combo).Error
						if errors.Is(err, gorm.ErrRecordNotFound) {
							// Assign a random/distinct color for newly created combo
							color := getRandomColor(cName)
							combo = models.Combo{
								UserID: userID,
								Name:   cName,
								Color:  color,
								Market: asset.Market,
							}
							if err := tx.Create(&combo).Error; err != nil {
								return err
							}
						} else if err != nil {
							return err
						}

						combosToLink = append(combosToLink, combo)
					}

					if len(combosToLink) > 0 {
						if err := tx.Model(&holding).Association("Combos").Replace(combosToLink); err != nil {
							return err
						}
					}
				}
			}

			// Collect processed assets for deferred Redis caching
			importedAssets = append(importedAssets, &asset)
		}
		return nil
	})

	if err == nil {
		// Only write to Redis after the entire transaction commits successfully
		WriteAssetsToRedis(importedAssets)
	}

	return err
}

// Helper to generate consistent colors based on combo name
func getRandomColor(name string) string {
	colors := []string{
		"#409EFF", // Blue
		"#67C23A", // Green
		"#E6A23C", // Orange
		"#F56C6C", // Red
		"#909399", // Grey
		"#9b59b6", // Purple
		"#1abc9c", // Turquoise
		"#e67e22", // Pumpkin
		"#34495e", // Wet Asphalt
		"#d35400", // Pumpkin Dark
	}

	// Simple hash function
	var sum int
	for _, char := range name {
		sum += int(char)
	}

	return colors[sum%len(colors)]
}
