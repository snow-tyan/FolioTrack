package controllers

import (
	"errors"
	"foliotrack/models"
	"foliotrack/services"
	"foliotrack/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// === AUTH CONTROLLER ===

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40001, "输入不合法: "+err.Error())
		return
	}

	// Check if user exists
	var existingUser models.User
	err := models.DB.Where("username = ?", input.Username).First(&existingUser).Error
	if err == nil {
		utils.Error(c, http.StatusConflict, 40002, "用户名已存在")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50001, "密码加密失败")
		return
	}

	user := models.User{
		Username:     input.Username,
		PasswordHash: string(hashedPassword),
	}

	if err := models.DB.Create(&user).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50002, "创建用户失败")
		return
	}

	utils.Success(c, gin.H{"id": user.ID, "username": user.Username})
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40003, "用户名或密码格式不正确")
		return
	}

	var user models.User
	err := models.DB.Where("username = ?", input.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Error(c, http.StatusUnauthorized, 40004, "用户名或密码错误")
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		utils.Error(c, http.StatusUnauthorized, 40005, "用户名或密码错误")
		return
	}

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Username)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50003, "生成 Token 失败")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func Me(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		utils.Error(c, http.StatusUnauthorized, 40006, "未登录")
		return
	}

	var user models.User
	if err := models.DB.First(&user, userID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40007, "用户未找到")
		return
	}

	utils.Success(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
	})
}

// === HOLDING CONTROLLER ===

type HoldingInput struct {
	Symbol    string   `json:"symbol" binding:"required"`
	Market    string   `json:"market" binding:"required"` // A-share, HK-stock, US-stock, Fund
	Quantity  float64  `json:"quantity" binding:"required,gt=0"`
	CostPrice float64  `json:"costPrice" binding:"required,gt=0"`
	ComboIDs  []uint   `json:"comboIds"`
}

func ListHoldings(c *gin.Context) {
	userID, _ := c.Get("userID")

	var holdings []*models.Holding
	err := models.DB.Preload("Asset").Preload("Combos").Where("user_id = ?", userID).Find(&holdings).Error
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50004, "获取持仓列表失败")
		return
	}

	// Extract unique assets to update prices
	var assetsToUpdate []*models.Asset
	assetMap := make(map[uint]*models.Asset)
	for _, h := range holdings {
		if _, ok := assetMap[h.AssetID]; !ok {
			assetMap[h.AssetID] = &h.Asset
			assetsToUpdate = append(assetsToUpdate, &h.Asset)
		}
	}

	// Fetch & update prices dynamically (with 1-min caching inside the service)
	if len(assetsToUpdate) > 0 {
		_ = services.UpdateAssetPrices(models.DB, assetsToUpdate, false, false)
		services.PopulateAssetCurrencyAndRates(assetsToUpdate)
	}

	// Re-assign updated assets back to response
	for _, h := range holdings {
		if updatedAsset, ok := assetMap[h.AssetID]; ok {
			h.Asset = *updatedAsset
		}
	}

	utils.Success(c, holdings)
}

func CreateHolding(c *gin.Context) {
	userID, _ := c.Get("userID")
	var input HoldingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40008, "参数不合法: "+err.Error())
		return
	}

	// Normalize input
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	market := strings.TrimSpace(input.Market)

	// Validate Market
	validMarkets := map[string]bool{"A-share": true, "HK-stock": true, "US-stock": true, "Fund": true}
	if !validMarkets[market] {
		utils.Error(c, http.StatusBadRequest, 40009, "不受支持的资产市场类型")
		return
	}

	// Use Transaction
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Find or create Asset
		var asset models.Asset
		err := tx.Where("symbol = ? AND market = ?", symbol, market).First(&asset).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			asset = models.Asset{
				Symbol: symbol,
				Market: market,
				Name:   symbol, // temporary name
			}
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
			// Fetch price immediately using the transaction
			_ = services.UpdateAssetPrices(tx, []*models.Asset{&asset}, true, false)
		} else if err != nil {
			return err
		}

		// 2. Check if user already holds this asset
		var holding models.Holding
		err = tx.Where("user_id = ? AND asset_id = ?", userID, asset.ID).First(&holding).Error
		if err == nil {
			return errors.New("已持有该股票，请直接修改持仓")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 3. Create holding
		holding = models.Holding{
			UserID:    userID.(uint),
			AssetID:   asset.ID,
			Quantity:  input.Quantity,
			CostPrice: input.CostPrice,
		}
		if err := tx.Create(&holding).Error; err != nil {
			return err
		}

		// 4. Associate Combos
		if len(input.ComboIDs) > 0 {
			var combos []models.Combo
			if err := tx.Where("user_id = ? AND id IN ?", userID, input.ComboIDs).Find(&combos).Error; err != nil {
				return err
			}
			if len(combos) > 0 {
				if err := tx.Model(&holding).Association("Combos").Replace(combos); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40010, err.Error())
		return
	}

	utils.Success(c, "创建持仓成功")
}

func UpdateHolding(c *gin.Context) {
	userID, _ := c.Get("userID")
	holdingIDStr := c.Param("id")
	holdingID, err := strconv.ParseUint(holdingIDStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40011, "无效的 ID")
		return
	}

	var input HoldingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40012, "参数不合法")
		return
	}

	var holding models.Holding
	err = models.DB.Where("id = ? AND user_id = ?", holdingID, userID).First(&holding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Error(c, http.StatusNotFound, 40013, "未找到持仓记录")
		return
	} else if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50005, "数据库错误")
		return
	}

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		holding.Quantity = input.Quantity
		holding.CostPrice = input.CostPrice

		if err := tx.Save(&holding).Error; err != nil {
			return err
		}

		// Replace combos
		var combos []models.Combo
		if len(input.ComboIDs) > 0 {
			if err := tx.Where("user_id = ? AND id IN ?", userID, input.ComboIDs).Find(&combos).Error; err != nil {
				return err
			}
		}
		
		if err := tx.Model(&holding).Association("Combos").Replace(combos); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50006, "修改持仓失败")
		return
	}

	utils.Success(c, "更新持仓成功")
}

func DeleteHolding(c *gin.Context) {
	userID, _ := c.Get("userID")
	holdingIDStr := c.Param("id")
	holdingID, err := strconv.ParseUint(holdingIDStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40014, "无效的 ID")
		return
	}

	// Delete holding (association will be handled by DB cascade if schema handles it, GORM handles it since we declared constraint:OnDelete:CASCADE)
	result := models.DB.Where("id = ? AND user_id = ?", holdingID, userID).Delete(&models.Holding{})
	if result.Error != nil {
		utils.Error(c, http.StatusInternalServerError, 50007, "删除持仓失败")
		return
	}

	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, 40015, "未找到持仓记录")
		return
	}

	utils.Success(c, "删除持仓成功")
}

// === COMBO CONTROLLER ===

type ComboInput struct {
	Name  string `json:"name" binding:"required,min=1,max=50"`
	Color string `json:"color"`
}

func ListCombos(c *gin.Context) {
	userID, _ := c.Get("userID")

	var combos []models.Combo
	err := models.DB.Where("user_id = ?", userID).Find(&combos).Error
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50008, "获取组合列表失败")
		return
	}

	utils.Success(c, combos)
}

func CreateCombo(c *gin.Context) {
	userID, _ := c.Get("userID")
	var input ComboInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40016, "参数不合法")
		return
	}

	color := input.Color
	if color == "" {
		color = "#409EFF" // Default Blue
	}

	combo := models.Combo{
		UserID: userID.(uint),
		Name:   input.Name,
		Color:  color,
	}

	if err := models.DB.Create(&combo).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50009, "创建组合失败")
		return
	}

	utils.Success(c, combo)
}

func UpdateCombo(c *gin.Context) {
	userID, _ := c.Get("userID")
	comboIDStr := c.Param("id")
	comboID, err := strconv.ParseUint(comboIDStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40017, "无效的 ID")
		return
	}

	var input ComboInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40018, "参数不合法")
		return
	}

	var combo models.Combo
	err = models.DB.Where("id = ? AND user_id = ?", comboID, userID).First(&combo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Error(c, http.StatusNotFound, 40019, "组合不存在")
		return
	}

	combo.Name = input.Name
	if input.Color != "" {
		combo.Color = input.Color
	}

	if err := models.DB.Save(&combo).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50010, "更新组合失败")
		return
	}

	utils.Success(c, combo)
}

func DeleteCombo(c *gin.Context) {
	userID, _ := c.Get("userID")
	comboIDStr := c.Param("id")
	comboID, err := strconv.ParseUint(comboIDStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40020, "无效的 ID")
		return
	}

	// Delete combo
	result := models.DB.Where("id = ? AND user_id = ?", comboID, userID).Delete(&models.Combo{})
	if result.Error != nil {
		utils.Error(c, http.StatusInternalServerError, 50011, "删除组合失败")
		return
	}

	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, 40021, "组合不存在")
		return
	}

	utils.Success(c, "删除组合成功")
}

// === IMPORT / EXPORT ===

func ExportHoldings(c *gin.Context) {
	userID, _ := c.Get("userID")

	csvBytes, err := services.ExportHoldingsCSV(userID.(uint))
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50012, "导出 CSV 失败: "+err.Error())
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=foliotrack_holdings.csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Data(http.StatusOK, "text/csv", csvBytes)
}

func ImportHoldings(c *gin.Context) {
	userID, _ := c.Get("userID")

	file, err := c.FormFile("file")
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40022, "未接收到上传文件")
		return
	}

	srcFile, err := file.Open()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50013, "打开上传文件失败")
		return
	}
	defer srcFile.Close()

	err = services.ImportHoldingsCSV(userID.(uint), srcFile)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40023, "导入持仓失败: "+err.Error())
		return
	}

	utils.Success(c, "导入持仓成功")
}

// === MARKET INDEXES ===

func GetMarketIndexes(c *gin.Context) {
	indices, err := services.FetchIndexData()
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50014, "获取市场指数失败")
		return
	}
	utils.Success(c, indices)
}
