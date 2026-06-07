package services

import (
	"context"
	"encoding/json"
	"fmt"
	"foliotrack/models"
	"foliotrack/utils"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type redisPriceCache struct {
	Name         string  `json:"name"`
	CurrentPrice float64 `json:"currentPrice"`
	PrevClose    float64 `json:"prevClose"`
}

// Map from db assets to sina ticker symbol
func GetSinaSymbol(symbol string, market string) string {
	switch market {
	case "A-share":
		if len(symbol) == 6 {
			// Shanghai
			if symbol[0] == '6' || symbol[0] == '9' || (symbol[0] == '7' && symbol[1] == '0') {
				return "sh" + symbol
			}
			// Shenzhen
			if symbol[0] == '0' || symbol[0] == '3' || symbol[0] == '2' || symbol[0] == '1' {
				return "sz" + symbol
			}
			// Beijing
			if symbol[0] == '8' || symbol[0] == '4' {
				return "bj" + symbol
			}
		}
		return symbol
	case "HK-stock":
		padded := symbol
		for len(padded) < 5 {
			padded = "0" + padded
		}
		return "rt_hk" + padded
	case "US-stock":
		return "gb_" + strings.ToLower(symbol)
	case "Fund":
		return "f_" + symbol
	default:
		return symbol
	}
}

// Map from sina symbol to db representation
type PriceInfo struct {
	Name         string
	CurrentPrice float64
	PrevClose    float64
}

// FetchPricesFromSina fetches a batch of prices from Sina Finance
func FetchPricesFromSina(sinaSymbols []string) (map[string]PriceInfo, error) {
	if len(sinaSymbols) == 0 {
		return make(map[string]PriceInfo), nil
	}

	url := fmt.Sprintf("https://hq.sinajs.cn/list=%s", strings.Join(sinaSymbols, ","))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Sina requires Referer header
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Decode GBK to UTF-8
	utf8Bytes, err := utils.GBKToUTF8(bodyBytes)
	if err != nil {
		// Fallback to raw bytes if decoding fails
		utf8Bytes = bodyBytes
	}

	responseString := string(utf8Bytes)
	result := make(map[string]PriceInfo)

	// Regexp to match Sina response: var hq_str_sh600519="贵州茅台,1800.00,...";
	re := regexp.MustCompile(`var hq_str_([a-zA-Z0-9_]+)="([^"]*)";`)
	matches := re.FindAllStringSubmatch(responseString, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		sinaSym := match[1]
		csvData := match[2]
		if csvData == "" {
			continue
		}

		parts := strings.Split(csvData, ",")
		if len(parts) < 2 {
			continue
		}

		var info PriceInfo
		if strings.HasPrefix(sinaSym, "sh") || strings.HasPrefix(sinaSym, "sz") || strings.HasPrefix(sinaSym, "bj") {
			// A-shares
			info.Name = parts[0]
			current, _ := strconv.ParseFloat(parts[3], 64)
			prevClose, _ := strconv.ParseFloat(parts[2], 64)
			// If current trading is suspended or before market open, current price might be 0, fallback to prevClose
			if current == 0 {
				current = prevClose
			}
			info.CurrentPrice = current
			info.PrevClose = prevClose

		} else if strings.HasPrefix(sinaSym, "f_") {
			// Funds
			info.Name = parts[0]
			current, _ := strconv.ParseFloat(parts[1], 64)
			prevClose, _ := strconv.ParseFloat(parts[3], 64)
			if current == 0 {
				current = prevClose
			}
			info.CurrentPrice = current
			info.PrevClose = prevClose

		} else if strings.HasPrefix(sinaSym, "rt_hk") {
			// HK stock
			if len(parts) >= 7 {
				name := parts[1] // Chinese Name
				if name == "" {
					name = parts[0] // Fallback to Eng Name
				}
				info.Name = name
				current, _ := strconv.ParseFloat(parts[6], 64)
				prevClose, _ := strconv.ParseFloat(parts[3], 64)
				if current == 0 {
					current = prevClose
				}
				info.CurrentPrice = current
				info.PrevClose = prevClose
			}

		} else if strings.HasPrefix(sinaSym, "gb_") {
			// US stock
			info.Name = parts[0]
			current, _ := strconv.ParseFloat(parts[1], 64)
			var prevClose float64
			if len(parts) >= 27 {
				prevClose, _ = strconv.ParseFloat(parts[26], 64)
			}
			if prevClose == 0 && len(parts) >= 5 {
				// Fallback: calculate from change percent if possible
				changeAmt, _ := strconv.ParseFloat(parts[4], 64)
				prevClose = current - changeAmt
			}
			if current == 0 {
				current = prevClose
			}
			info.CurrentPrice = current
			info.PrevClose = prevClose
		}

		if info.Name != "" {
			result[strings.ToLower(sinaSym)] = info
		}
	}

	return result, nil
}

// UpdateAssetPrices fetches and updates asset prices in the database.
// Supports caching: assets updated within the last cacheDuration are skipped.
func UpdateAssetPrices(db *gorm.DB, assets []*models.Asset, force bool, skipRedis bool) error {
	if len(assets) == 0 {
		return nil
	}

	cacheDuration := 1 * time.Minute
	now := time.Now()

	// Filter assets that need update
	var assetsToUpdate []*models.Asset
	var sinaSymbols []string
	assetBySinaSymbol := make(map[string][]*models.Asset)

	ctx := context.Background()

	for _, asset := range assets {
		redisCached := false
		// Try Redis first if force is false, Redis is active, and we're not skipping Redis
		if !force && !skipRedis && models.RedisClient != nil {
			redisKey := fmt.Sprintf("foliotrack:price:%s:%s", asset.Market, asset.Symbol)
			val, err := models.RedisClient.Get(ctx, redisKey).Result()
			if err == nil && val != "" {
				var cached redisPriceCache
				if err := json.Unmarshal([]byte(val), &cached); err == nil {
					asset.Name = cached.Name
					asset.CurrentPrice = cached.CurrentPrice
					asset.PrevClose = cached.PrevClose
					asset.LastUpdated = now
					redisCached = true
				}
			}
		}

		if !redisCached {
			if force || now.Sub(asset.LastUpdated) >= cacheDuration || asset.CurrentPrice == 0 {
				sinaSym := strings.ToLower(GetSinaSymbol(asset.Symbol, asset.Market))
				assetsToUpdate = append(assetsToUpdate, asset)
				sinaSymbols = append(sinaSymbols, sinaSym)
				assetBySinaSymbol[sinaSym] = append(assetBySinaSymbol[sinaSym], asset)
			}
		}
	}

	if len(assetsToUpdate) == 0 {
		return nil
	}

	// Fetch prices in batches of 50 to avoid URL length issues
	batchSize := 50
	for i := 0; i < len(sinaSymbols); i += batchSize {
		end := i + batchSize
		if end > len(sinaSymbols) {
			end = len(sinaSymbols)
		}
		chunk := sinaSymbols[i:end]

		prices, err := FetchPricesFromSina(chunk)
		if err != nil {
			return err
		}

		// Update database and structs
		for sinaSym, priceInfo := range prices {
			assetsList, ok := assetBySinaSymbol[sinaSym]
			if !ok {
				continue
			}
			for _, asset := range assetsList {
				asset.Name = priceInfo.Name
				asset.CurrentPrice = priceInfo.CurrentPrice
				asset.PrevClose = priceInfo.PrevClose
				asset.LastUpdated = now

				// Save to DB using the transactional db reference
				db.Save(asset)

				// Cache in Redis (unless skipRedis is set)
				if !skipRedis && models.RedisClient != nil {
					redisKey := fmt.Sprintf("foliotrack:price:%s:%s", asset.Market, asset.Symbol)
					cacheVal := redisPriceCache{
						Name:         priceInfo.Name,
						CurrentPrice: priceInfo.CurrentPrice,
						PrevClose:    priceInfo.PrevClose,
					}
					if cacheBytes, err := json.Marshal(cacheVal); err == nil {
						_ = models.RedisClient.Set(ctx, redisKey, cacheBytes, cacheDuration).Err()
					}
				}
			}
		}
	}

	return nil
}

// WriteAssetsToRedis directly writes a list of assets to the Redis cache
func WriteAssetsToRedis(assets []*models.Asset) {
	if models.RedisClient == nil || len(assets) == 0 {
		return
	}

	ctx := context.Background()
	cacheDuration := 1 * time.Minute

	for _, asset := range assets {
		// Only write if name is valid
		if asset.Name == "" || asset.CurrentPrice <= 0 {
			continue
		}

		redisKey := fmt.Sprintf("foliotrack:price:%s:%s", asset.Market, asset.Symbol)
		cacheVal := redisPriceCache{
			Name:         asset.Name,
			CurrentPrice: asset.CurrentPrice,
			PrevClose:    asset.PrevClose,
		}

		if cacheBytes, err := json.Marshal(cacheVal); err == nil {
			_ = models.RedisClient.Set(ctx, redisKey, cacheBytes, cacheDuration).Err()
		}
	}
}

// FetchIndexData fetches index ticker data for SSE, HSI, DJI, NASDAQ
func FetchIndexData() ([]map[string]interface{}, error) {
	// Sina symbols for indices:
	// s_sh000001: Shanghai Composite
	// rt_hkHSI: Hang Seng Index
	// gb_$dji: Dow Jones Industrial Average
	// gb_ixic: Nasdaq Composite
	symbols := []string{"s_sh000001", "rt_hkHSI", "gb_$dji", "gb_ixic"}
	url := fmt.Sprintf("https://hq.sinajs.cn/list=%s", strings.Join(symbols, ","))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	utf8Bytes, _ := utils.GBKToUTF8(bodyBytes)
	responseString := string(utf8Bytes)

	re := regexp.MustCompile(`var hq_str_([a-zA-Z0-9_$]+)="([^"]*)";`)
	matches := re.FindAllStringSubmatch(responseString, -1)

	var indexList []map[string]interface{}
	nameMap := map[string]string{
		"s_sh000001": "上证指数",
		"rt_hkHSI":   "恒生指数",
		"gb_$dji":    "道琼斯",
		"gb_ixic":    "纳斯达克",
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		sym := match[1]
		csvData := match[2]
		if csvData == "" {
			continue
		}

		parts := strings.Split(csvData, ",")
		var current, change, changePct float64

		if sym == "s_sh000001" { // SSE Composite
			if len(parts) >= 4 {
				current, _ = strconv.ParseFloat(parts[1], 64)
				change, _ = strconv.ParseFloat(parts[2], 64)
				changePct, _ = strconv.ParseFloat(parts[3], 64)
			}
		} else if sym == "rt_hkHSI" { // HSI
			if len(parts) >= 7 {
				current, _ = strconv.ParseFloat(parts[6], 64)
				// parts[3] is yesterday close
				prevClose, _ := strconv.ParseFloat(parts[3], 64)
				change = current - prevClose
				if prevClose != 0 {
					changePct = (change / prevClose) * 100
				}
			}
		} else if sym == "gb_$dji" || sym == "gb_ixic" { // Dow / Nasdaq
			if len(parts) >= 5 {
				current, _ = strconv.ParseFloat(parts[1], 64)
				changePct, _ = strconv.ParseFloat(parts[2], 64)
				change, _ = strconv.ParseFloat(parts[4], 64)
			}
		}

		displayName := nameMap[sym]
		if displayName == "" {
			displayName = sym
		}

		indexList = append(indexList, map[string]interface{}{
			"symbol":    sym,
			"name":      displayName,
			"current":   current,
			"change":    change,
			"changePct": changePct,
		})
	}

	return indexList, nil
}

// ExchangeRates represents exchange rates relative to CNY
type ExchangeRates struct {
	USD float64 `json:"usd"`
	HKD float64 `json:"hkd"`
	CNY float64 `json:"cny"`
}

// DefaultExchangeRates is the fallback in case the Sina API is offline
var DefaultExchangeRates = ExchangeRates{
	USD: 7.20,
	HKD: 0.92,
	CNY: 1.0,
}

// GetExchangeRates fetches exchange rates from Sina Finance or Redis
func GetExchangeRates() (ExchangeRates, error) {
	ctx := context.Background()
	cacheKey := "foliotrack:exchange_rates"

	// Try Redis first
	if models.RedisClient != nil {
		val, err := models.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			var rates ExchangeRates
			if err := json.Unmarshal([]byte(val), &rates); err == nil {
				return rates, nil
			}
		}
	}

	// Fetch from Sina Finance
	url := "https://hq.sinajs.cn/list=fx_susdcny,fx_shkdcny"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return DefaultExchangeRates, err
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return DefaultExchangeRates, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return DefaultExchangeRates, err
	}

	utf8Bytes, _ := utils.GBKToUTF8(bodyBytes)
	responseString := string(utf8Bytes)

	re := regexp.MustCompile(`var hq_str_([a-zA-Z0-9_]+)="([^"]*)";`)
	matches := re.FindAllStringSubmatch(responseString, -1)

	rates := DefaultExchangeRates

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		sym := match[1]
		csvData := match[2]
		if csvData == "" {
			continue
		}
		parts := strings.Split(csvData, ",")
		if len(parts) < 2 {
			continue
		}
		rateVal, err := strconv.ParseFloat(parts[1], 64)
		if err != nil || rateVal <= 0 {
			continue
		}

		if sym == "fx_susdcny" {
			rates.USD = rateVal
		} else if sym == "fx_shkdcny" {
			rates.HKD = rateVal
		}
	}

	// Cache to Redis for 10 minutes
	if models.RedisClient != nil {
		if cacheBytes, err := json.Marshal(rates); err == nil {
			_ = models.RedisClient.Set(ctx, cacheKey, cacheBytes, 10*time.Minute).Err()
		}
	}

	return rates, nil
}

// PopulateAssetCurrencyAndRates fills virtual fields for a slice of assets
func PopulateAssetCurrencyAndRates(assets []*models.Asset) {
	rates, _ := GetExchangeRates()
	for _, asset := range assets {
		switch asset.Market {
		case "A-share", "Fund":
			asset.Currency = "CNY"
			asset.ExchangeRate = 1.0
		case "US-stock":
			asset.Currency = "USD"
			asset.ExchangeRate = rates.USD
		case "HK-stock":
			asset.Currency = "HKD"
			asset.ExchangeRate = rates.HKD
		default:
			asset.Currency = "CNY"
			asset.ExchangeRate = 1.0
		}
	}
}
