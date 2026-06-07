package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"foliotrack/models"
	"io"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// ExportHoldingsCSV generates a CSV file content representing the user's holdings
func ExportHoldingsCSV(userID uint) ([]byte, error) {
	var holdings []models.Holding
	err := models.DB.Preload("Asset").Preload("Combos").Where("user_id = ?", userID).Find(&holdings).Error
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Write BOM for Excel compatibility in Chinese environments
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	// Header row
	err = writer.Write([]string{"symbol", "market", "quantity", "cost_price", "combos"})
	if err != nil {
		return nil, err
	}

	for _, h := range holdings {
		var comboNames []string
		for _, c := range h.Combos {
			comboNames = append(comboNames, c.Name)
		}

		row := []string{
			h.Asset.Symbol,
			h.Asset.Market,
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

	// Transaction to import rows
	return models.DB.Transaction(func(tx *gorm.DB) error {
		for i, row := range rows {
			symbol := strings.TrimSpace(row[headerMap["symbol"]])
			market := strings.TrimSpace(row[headerMap["market"]])
			quantityStr := strings.TrimSpace(row[headerMap["quantity"]])
			costPriceStr := strings.TrimSpace(row[headerMap["cost_price"]])

			if symbol == "" || market == "" {
				continue // skip empty rows
			}

			quantity, err := strconv.ParseFloat(quantityStr, 64)
			if err != nil {
				return fmt.Errorf("row %d: invalid quantity '%s'", i+2, quantityStr)
			}

			costPrice, err := strconv.ParseFloat(costPriceStr, 64)
			if err != nil {
				return fmt.Errorf("row %d: invalid cost_price '%s'", i+2, costPriceStr)
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
				// Fetch metadata from Sina immediately
				_ = UpdateAssetPrices([]*models.Asset{&asset}, true)
			} else if err != nil {
				return err
			}

			// 2. Find or create Holding
			var holding models.Holding
			err = tx.Where("user_id = ? AND asset_id = ?", userID, asset.ID).First(&holding).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				holding = models.Holding{
					UserID:    userID,
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
						err = tx.Where("user_id = ? AND name = ?", userID, cName).First(&combo).Error
						if errors.Is(err, gorm.ErrRecordNotFound) {
							// Assign a random/distinct color for newly created combo
							color := getRandomColor(cName)
							combo = models.Combo{
								UserID: userID,
								Name:   cName,
								Color:  color,
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
		}
		return nil
	})
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
