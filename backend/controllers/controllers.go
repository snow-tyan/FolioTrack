package controllers

import (
	"crypto/subtle"
	"errors"
	"foliotrack/config"
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
	Username   string `json:"username" binding:"required,min=3,max=50"`
	Password   string `json:"password" binding:"required,min=6"`
	InviteCode string `json:"inviteCode"`
}

func Register(c *gin.Context) {
	if !config.AppConfig.AllowRegistration {
		utils.Error(c, http.StatusForbidden, 40032, "当前站点已关闭公开注册，请联系管理员")
		return
	}

	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40001, "输入不合法: "+err.Error())
		return
	}

	if config.AppConfig.RegistrationInviteCode != "" {
		if subtle.ConstantTimeCompare([]byte(input.InviteCode), []byte(config.AppConfig.RegistrationInviteCode)) != 1 {
			utils.Error(c, http.StatusForbidden, 40033, "邀请码不正确")
			return
		}
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
		Username:      input.Username,
		PasswordHash:  string(hashedPassword),
		Role:          "user",
		LoginAttempts: 0,
		IsLocked:      false,
	}

	if err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return services.EnsureDefaultAccounts(tx, user.ID)
	}); err != nil {
		utils.Error(c, http.StatusInternalServerError, 50002, "创建用户失败")
		return
	}

	utils.Success(c, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
}

func GetSystemConfig(c *gin.Context) {
	utils.Success(c, gin.H{
		"allowRegistration": config.AppConfig.AllowRegistration,
		"requireInviteCode": config.AppConfig.RegistrationInviteCode != "",
	})
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

	// Check if account is locked
	if user.IsLocked {
		utils.Error(c, http.StatusForbidden, 40016, "该账号因密码连续输入错误5次已被锁定，请联系管理员重置或解锁")
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		// Increment login attempts
		user.LoginAttempts++
		if user.LoginAttempts >= 5 {
			user.IsLocked = true
		}
		models.DB.Save(&user)

		if user.IsLocked {
			utils.Error(c, http.StatusForbidden, 40016, "密码输入错误次数过多，账号已被锁定，请联系管理员解锁")
		} else {
			utils.Error(c, http.StatusUnauthorized, 40005, "用户名或密码错误")
		}
		return
	}

	// Reset login attempts on successful login
	if user.LoginAttempts > 0 {
		user.LoginAttempts = 0
		models.DB.Save(&user)
	}

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Username, user.Role)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50003, "生成 Token 失败")
		return
	}

	utils.Success(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
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
		"role":     user.Role,
	})
}

// === HOLDING CONTROLLER ===

type HoldingInput struct {
	Symbol    string  `json:"symbol" binding:"required"`
	Market    string  `json:"market" binding:"required"` // A-share, HK-stock, US-stock, Fund
	AccountID uint    `json:"accountId"`
	Quantity  float64 `json:"quantity" binding:"required"`
	CostPrice float64 `json:"costPrice" binding:"required"`
	ComboIDs  []uint  `json:"comboIds"`
	IsPublic  *bool   `json:"isPublic"`
}

func ListHoldings(c *gin.Context) {
	userID, _ := c.Get("userID")
	userIDVal := userID.(uint)

	var holdings []*models.Holding
	err := models.DB.Preload("Asset").Preload("Account").Preload("Combos").Preload("User").Where("user_id = ?", userIDVal).Find(&holdings).Error

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
	if !services.IsValidMarket(market) {
		utils.Error(c, http.StatusBadRequest, 40009, "不受支持的资产市场类型")
		return
	}
	if symbol == "" {
		utils.Error(c, http.StatusBadRequest, 40008, "资产代码不能为空")
		return
	}
	if err := validateHoldingNumbers(market, input.Quantity, input.CostPrice, false); err != nil {
		utils.Error(c, http.StatusBadRequest, 40008, err.Error())
		return
	}

	// Use Transaction
	err := models.DB.Transaction(func(tx *gorm.DB) error {
		account, err := services.ResolveAccountForHolding(tx, userID.(uint), market, input.AccountID)
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

		// 2. Check if this account already holds this asset
		var holding models.Holding
		err = tx.Where("user_id = ? AND account_id = ? AND asset_id = ?", userID, account.ID, asset.ID).First(&holding).Error
		if err == nil {
			return errors.New("该账户已持有该标的，请直接修改持仓")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 3. Create holding
		isPublic := false
		if input.IsPublic != nil {
			isPublic = *input.IsPublic
		}
		holding = models.Holding{
			UserID:    userID.(uint),
			AccountID: account.ID,
			AssetID:   asset.ID,
			Quantity:  input.Quantity,
			CostPrice: input.CostPrice,
			IsPublic:  isPublic,
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
	err = models.DB.Preload("Asset").Where("id = ? AND user_id = ?", holdingID, userID).First(&holding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Error(c, http.StatusNotFound, 40013, "未找到持仓记录")
		return
	} else if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50005, "数据库错误")
		return
	}
	if err := validateHoldingNumbers(holding.Asset.Market, input.Quantity, input.CostPrice, false); err != nil {
		utils.Error(c, http.StatusBadRequest, 40012, err.Error())
		return
	}

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		account, err := services.ResolveAccountForHolding(tx, userID.(uint), holding.Asset.Market, input.AccountID)
		if err != nil {
			return err
		}

		var duplicate models.Holding
		err = tx.Where("id <> ? AND user_id = ? AND account_id = ? AND asset_id = ?", holding.ID, userID, account.ID, holding.AssetID).First(&duplicate).Error
		if err == nil {
			return errors.New("该账户已持有该标的")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		holding.AccountID = account.ID
		holding.Quantity = input.Quantity
		holding.CostPrice = input.CostPrice
		if input.IsPublic != nil {
			holding.IsPublic = *input.IsPublic
		}

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
		utils.Error(c, http.StatusInternalServerError, 50006, "修改持仓失败: "+err.Error())
		return
	}

	utils.Success(c, "更新持仓成功")
}

func ClearHolding(c *gin.Context) {
	userID, _ := c.Get("userID")
	holdingIDStr := c.Param("id")
	holdingID, err := strconv.ParseUint(holdingIDStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40040, "无效的 ID")
		return
	}

	result := models.DB.Model(&models.Holding{}).
		Where("id = ? AND user_id = ?", holdingID, userID).
		Updates(map[string]interface{}{
			"quantity":   0,
			"cost_price": 0,
		})
	if result.Error != nil {
		utils.Error(c, http.StatusInternalServerError, 50025, "清仓失败")
		return
	}
	if result.RowsAffected == 0 {
		utils.Error(c, http.StatusNotFound, 40041, "未找到持仓记录")
		return
	}

	utils.Success(c, "清仓成功")
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

// === ACCOUNT CONTROLLER ===

type AccountInput struct {
	Name   string `json:"name" binding:"required,min=1,max=50"`
	Market string `json:"market" binding:"required"`
}

func ListAccounts(c *gin.Context) {
	userID, _ := c.Get("userID")

	if err := services.EnsureDefaultAccounts(models.DB, userID.(uint)); err != nil {
		utils.Error(c, http.StatusInternalServerError, 50022, "初始化账户失败")
		return
	}

	var accounts []models.Account
	err := models.DB.Where("user_id = ?", userID).Order("market asc, is_default desc, id asc").Find(&accounts).Error
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50023, "获取账户列表失败")
		return
	}

	utils.Success(c, accounts)
}

func CreateAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	var input AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40034, "参数不合法: "+err.Error())
		return
	}

	market := strings.TrimSpace(input.Market)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		utils.Error(c, http.StatusBadRequest, 40034, "账户名称不能为空")
		return
	}
	if !services.IsValidMarket(market) {
		utils.Error(c, http.StatusBadRequest, 40009, "不受支持的资产市场类型")
		return
	}

	account := models.Account{
		UserID: userID.(uint),
		Market: market,
		Name:   name,
	}
	if err := models.DB.Create(&account).Error; err != nil {
		utils.Error(c, http.StatusBadRequest, 40035, "创建账户失败，可能存在同名账户")
		return
	}

	utils.Success(c, account)
}

func UpdateAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40036, "无效的账户 ID")
		return
	}

	var input AccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40034, "参数不合法: "+err.Error())
		return
	}

	var account models.Account
	if err := models.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40037, "账户不存在")
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		utils.Error(c, http.StatusBadRequest, 40034, "账户名称不能为空")
		return
	}
	account.Name = name
	if err := models.DB.Save(&account).Error; err != nil {
		utils.Error(c, http.StatusBadRequest, 40035, "更新账户失败，可能存在同名账户")
		return
	}

	utils.Success(c, account)
}

func DeleteAccount(c *gin.Context) {
	userID, _ := c.Get("userID")
	accountID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40036, "无效的账户 ID")
		return
	}

	var account models.Account
	if err := models.DB.Where("id = ? AND user_id = ?", accountID, userID).First(&account).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40037, "账户不存在")
		return
	}
	if account.IsDefault {
		utils.Error(c, http.StatusBadRequest, 40038, "默认账户不能删除")
		return
	}

	var count int64
	if err := models.DB.Model(&models.Holding{}).Where("account_id = ?", account.ID).Count(&count).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50024, "检查账户持仓失败")
		return
	}
	if count > 0 {
		utils.Error(c, http.StatusBadRequest, 40039, "该账户仍有持仓，请先迁移或删除持仓")
		return
	}

	if err := models.DB.Delete(&account).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50025, "删除账户失败")
		return
	}

	utils.Success(c, "删除账户成功")
}

// === COMBO CONTROLLER ===

type ComboInput struct {
	Name     string `json:"name" binding:"required,min=1,max=50"`
	Color    string `json:"color"`
	Market   string `json:"market"`
	IsPublic *bool  `json:"isPublic"`
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

	market := input.Market
	if market == "" {
		market = "A-share" // Default fallback
	}

	isPublic := false
	if input.IsPublic != nil {
		isPublic = *input.IsPublic
	}

	combo := models.Combo{
		UserID:   userID.(uint),
		Name:     input.Name,
		Color:    color,
		Market:   market,
		IsPublic: isPublic,
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

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		combo.Name = input.Name
		if input.Color != "" {
			combo.Color = input.Color
		}
		if input.Market != "" {
			combo.Market = input.Market
		}
		if input.IsPublic != nil {
			combo.IsPublic = *input.IsPublic
			if *input.IsPublic {
				// Cascaded publish: set all holdings in this combo to public
				if err := tx.Model(&models.Holding{}).
					Where("id IN (SELECT holding_id FROM holding_combos WHERE combo_id = ?)", combo.ID).
					Update("is_public", true).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Save(&combo).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
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

// === PERSONNEL MANAGEMENT ===

func ListUsers(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权访问人员管理")
		return
	}

	var users []models.User
	if err := models.DB.Order("id asc").Find(&users).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50015, "获取用户列表失败")
		return
	}

	utils.Success(c, users)
}

type UserUpdateInput struct {
	Role     string `json:"role" binding:"required"`
	IsLocked *bool  `json:"isLocked" binding:"required"`
}

func UpdateUser(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权修改用户权限")
		return
	}

	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40025, "无效的用户 ID")
		return
	}

	var input UserUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40026, "参数不合法")
		return
	}

	validRoles := map[string]bool{"admin": true, "manager": true, "advanced": true, "user": true}
	if !validRoles[input.Role] {
		utils.Error(c, http.StatusBadRequest, 40027, "不支持的目标角色")
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40028, "用户未找到")
		return
	}

	// Enforce hierarchical restriction
	if role == "manager" {
		if targetUser.Role == "admin" || targetUser.Role == "manager" {
			utils.Error(c, http.StatusForbidden, 40029, "管理员无权修改更高或同等权限的用户")
			return
		}
		if input.Role == "admin" {
			utils.Error(c, http.StatusForbidden, 40030, "管理员不能将用户提权为超级管理员")
			return
		}
	}

	targetUser.Role = input.Role
	if input.IsLocked != nil {
		targetUser.IsLocked = *input.IsLocked
		if !targetUser.IsLocked {
			targetUser.LoginAttempts = 0
		}
	}

	if err := models.DB.Save(&targetUser).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50016, "更新用户失败")
		return
	}

	utils.Success(c, "更新用户成功")
}

func DeleteUser(c *gin.Context) {
	userID, _ := c.Get("userID")
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权删除用户")
		return
	}

	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40025, "无效的用户 ID")
		return
	}

	if userID.(uint) == uint(targetID) {
		utils.Error(c, http.StatusBadRequest, 40031, "不能删除当前登录的账号")
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40028, "用户未找到")
		return
	}

	if role == "manager" {
		if targetUser.Role == "admin" || targetUser.Role == "manager" {
			utils.Error(c, http.StatusForbidden, 40029, "管理员无权删除更高或同等权限的用户")
			return
		}
	}

	err = models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", targetID).Delete(&models.Holding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", targetID).Delete(&models.Combo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", targetID).Delete(&models.Account{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.User{}, targetID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50017, "删除用户失败")
		return
	}

	utils.Success(c, "删除用户成功")
}

type ResetPasswordInput struct {
	Password string `json:"password" binding:"required,min=6"`
}

func ResetUserPassword(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权重置密码")
		return
	}

	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40025, "无效的用户 ID")
		return
	}

	var input ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.Error(c, http.StatusBadRequest, 40026, "密码格式不合法，最少 6 位字符")
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40028, "用户未找到")
		return
	}

	if role == "manager" {
		if targetUser.Role == "admin" || targetUser.Role == "manager" {
			utils.Error(c, http.StatusForbidden, 40029, "管理员无权修改更高或同等权限的用户密码")
			return
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50001, "密码加密失败")
		return
	}

	targetUser.PasswordHash = string(hashedPassword)
	targetUser.LoginAttempts = 0
	targetUser.IsLocked = false

	if err := models.DB.Save(&targetUser).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50018, "重置密码失败")
		return
	}

	utils.Success(c, "重置密码成功")
}

func UnlockUser(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权解锁用户")
		return
	}

	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40025, "无效的用户 ID")
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40028, "用户未找到")
		return
	}

	if role == "manager" {
		if targetUser.Role == "admin" || targetUser.Role == "manager" {
			utils.Error(c, http.StatusForbidden, 40029, "管理员无权解锁更高或同等权限的用户")
			return
		}
	}

	targetUser.LoginAttempts = 0
	targetUser.IsLocked = false

	if err := models.DB.Save(&targetUser).Error; err != nil {
		utils.Error(c, http.StatusInternalServerError, 50019, "解锁用户失败")
		return
	}

	utils.Success(c, "解锁成功且错误次数已清零")
}

// === PUBLIC MARKET ===

func ListPublicHoldings(c *gin.Context) {
	var holdings []*models.Holding
	// Query only holdings where is_public = true (including current user's own public holdings)
	err := models.DB.Preload("Asset").Preload("Account").Preload("Combos").Preload("User").
		Where("is_public = ?", true).
		Find(&holdings).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50020, "获取公开持仓失败")
		return
	}

	// Dynamic price update (same logic as list holdings)
	var assetsToUpdate []*models.Asset
	assetMap := make(map[uint]*models.Asset)
	for _, h := range holdings {
		if _, ok := assetMap[h.AssetID]; !ok {
			assetMap[h.AssetID] = &h.Asset
			assetsToUpdate = append(assetsToUpdate, &h.Asset)
		}
	}
	if len(assetsToUpdate) > 0 {
		_ = services.UpdateAssetPrices(models.DB, assetsToUpdate, false, false)
		services.PopulateAssetCurrencyAndRates(assetsToUpdate)
	}
	for _, h := range holdings {
		if updatedAsset, ok := assetMap[h.AssetID]; ok {
			h.Asset = *updatedAsset
		}
	}

	utils.Success(c, holdings)
}

func ListPublicCombos(c *gin.Context) {
	var combos []models.Combo
	// Query public combos, and preload only their public holdings (including current user's own public combos)
	err := models.DB.Preload("Holdings", "is_public = ?", true).
		Preload("Holdings.Asset").Preload("Holdings.Account").Preload("Holdings.User").
		Where("is_public = ?", true).
		Find(&combos).Error

	if err != nil {
		utils.Error(c, http.StatusInternalServerError, 50021, "获取公开组合失败")
		return
	}

	// Collect unique assets inside combos for dynamic price update
	var assetsToUpdate []*models.Asset
	assetMap := make(map[uint]*models.Asset)
	for i := range combos {
		for j := range combos[i].Holdings {
			h := &combos[i].Holdings[j]
			if _, ok := assetMap[h.AssetID]; !ok {
				assetMap[h.AssetID] = &h.Asset
				assetsToUpdate = append(assetsToUpdate, &h.Asset)
			}
		}
	}
	if len(assetsToUpdate) > 0 {
		_ = services.UpdateAssetPrices(models.DB, assetsToUpdate, false, false)
		services.PopulateAssetCurrencyAndRates(assetsToUpdate)
	}
	// Re-assign prices and load rates
	for i := range combos {
		for j := range combos[i].Holdings {
			h := &combos[i].Holdings[j]
			if updatedAsset, ok := assetMap[h.AssetID]; ok {
				h.Asset = *updatedAsset
			}
		}
	}

	utils.Success(c, combos)
}

func GetUserHoldings(c *gin.Context) {
	role, _ := c.Get("role")
	if role != "admin" && role != "manager" {
		utils.Error(c, http.StatusForbidden, 40024, "无权访问此数据")
		return
	}

	idStr := c.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, 40025, "无效的用户 ID")
		return
	}

	var targetUser models.User
	if err := models.DB.First(&targetUser, targetID).Error; err != nil {
		utils.Error(c, http.StatusNotFound, 40028, "用户未找到")
		return
	}

	if role == "manager" {
		if targetUser.Role == "admin" || targetUser.Role == "manager" {
			utils.Error(c, http.StatusForbidden, 40029, "管理员无权查看同等或更高权限用户的持仓")
			return
		}
	}

	var holdings []*models.Holding
	err = models.DB.Preload("Asset").Preload("Account").Preload("Combos").Preload("User").Where("user_id = ?", targetID).Find(&holdings).Error
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

	// Fetch & update prices dynamically
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
