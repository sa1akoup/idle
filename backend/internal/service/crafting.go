// 工作台制造服务：配方目录展示（等级门槛与材料 have/need）与制造作业入队。
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"idle/internal/models"
	"idle/internal/repository/catalog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CraftingMaterialView 配方材料：需求与当前持有量。
type CraftingMaterialView struct {
	ItemID    string `json:"itemId"`
	Name      string `json:"name"`
	Need      int    `json:"need"`
	Have      int    `json:"have"`
	Satisfied bool   `json:"satisfied"`
}

// CraftingOutputView 配方产物信息。
type CraftingOutputView struct {
	ItemID           string `json:"itemId"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Quantity         int    `json:"quantity"`
	InstanceRequired bool   `json:"instanceRequired"`
}

// CraftingRecipeView 配方视图：叠加工作台等级、忙碌状态与可制造性。
type CraftingRecipeView struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	RequiredLevel  int                    `json:"requiredLevel"`
	CraftSeconds   int64                  `json:"craftSeconds"`
	CraftMinutes   int                    `json:"craftMinutes"`
	Output         CraftingOutputView     `json:"output"`
	Inputs         []CraftingMaterialView `json:"inputs"`
	WorkbenchLevel int                    `json:"workbenchLevel"`
	WorkbenchBusy  bool                   `json:"workbenchBusy"`
	CanStart       bool                   `json:"canStart"`
	Reason         string                 `json:"reason,omitempty"`
}

// ListCraftingRecipesForUser 返回全部配方及当前工作台门槛与材料余量。
func ListCraftingRecipesForUser(db *gorm.DB, userID uint) ([]CraftingRecipeView, error) {
	if err := settleDueHideoutJobsForUser(db, userID); err != nil {
		return nil, err
	}
	workbench, busy, err := workbenchStatus(db, userID)
	if err != nil {
		return nil, err
	}
	var recipes []models.RecipeDef
	if err := db.Order("sort_order asc, id asc").Find(&recipes).Error; err != nil {
		return nil, fmt.Errorf("读取制造配方: %w", err)
	}
	parsedInputs := make(map[string][]models.RecipeInput, len(recipes))
	itemIDs := make([]string, 0)
	for _, recipe := range recipes {
		inputs, err := parseRecipeInputs(recipe.InputsJSON)
		if err != nil {
			return nil, fmt.Errorf("解析配方 %s: %w", recipe.ID, err)
		}
		parsedInputs[recipe.ID] = inputs
		for _, input := range inputs {
			itemIDs = append(itemIDs, input.ItemID)
		}
		itemIDs = append(itemIDs, recipe.OutputItemID)
	}
	catalogRepo := catalog.New(db)
	catalogItems, catalogErr := catalogRepo.FindByIDs(itemIDs)
	if catalogErr != nil && !errors.Is(catalogErr, catalog.ErrItemNotFound) {
		return nil, fmt.Errorf("读取制造商品目录: %w", catalogErr)
	}
	views := make([]CraftingRecipeView, 0, len(recipes))
	for _, recipe := range recipes {
		inputs := parsedInputs[recipe.ID]
		materials := make([]CraftingMaterialView, 0, len(inputs))
		allSatisfied := true
		for _, input := range inputs {
			have, err := ownedItemQuantityTx(db, userID, input.ItemID)
			if err != nil {
				return nil, err
			}
			name := input.ItemID
			if item, ok := catalogItems[input.ItemID]; ok {
				name = item.Name
			}
			satisfied := have >= input.Quantity
			if !satisfied {
				allSatisfied = false
			}
			materials = append(materials, CraftingMaterialView{
				ItemID: input.ItemID, Name: name, Need: input.Quantity, Have: have, Satisfied: satisfied,
			})
		}
		output, ok := catalogItems[recipe.OutputItemID]
		if !ok {
			return nil, fmt.Errorf("配方产物 %s 不在目录: %w", recipe.OutputItemID, catalog.ErrItemNotFound)
		}
		use, found, err := catalogRepo.FindUseByID(recipe.OutputItemID)
		if err != nil {
			return nil, fmt.Errorf("读取物品使用效果 %s: %w", recipe.OutputItemID, err)
		}
		instanceRequired := found && use.InstanceRequired
		levelReady := workbench.Level >= recipe.RequiredLevel
		// 可制造条件 = 工作台等级达标 + 状态就绪 + 无进行中作业 + 材料齐全。
		canStart := levelReady && workbench.State == "ready" && !busy && allSatisfied
		reason := ""
		switch {
		case !levelReady:
			reason = fmt.Sprintf("需要工作台 LV.%d", recipe.RequiredLevel)
		case workbench.State != "ready":
			reason = "工作台正在升级中"
		case busy:
			reason = "工作台正在执行作业"
		case !allSatisfied:
			reason = "材料不足"
		}
		views = append(views, CraftingRecipeView{
			ID: recipe.ID, Name: recipe.Name, RequiredLevel: recipe.RequiredLevel,
			CraftSeconds: recipe.CraftSeconds, CraftMinutes: int((recipe.CraftSeconds + 59) / 60),
			Output: CraftingOutputView{
				ItemID: output.ID, Name: output.Name, Kind: output.Kind,
				Quantity: recipe.OutputQuantity, InstanceRequired: instanceRequired,
			},
			Inputs: materials, WorkbenchLevel: workbench.Level, WorkbenchBusy: busy, CanStart: canStart, Reason: reason,
		})
	}
	return views, nil
}

// StartCraftForUser 校验门槛与材料后即时扣除材料并入队制造作业。
// 流程：资源锁 → 惰性结算 → 工作台 ready+等级 → 单作业互斥 → 扣材料 → 入队容量预检 → 建作业。
func StartCraftForUser(db *gorm.DB, userID uint, recipeID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserResourcesTx(tx, userID); err != nil {
			return err
		}
		if err := settleDueHideoutJobsTx(tx, userID, time.Now()); err != nil {
			return err
		}
		var recipe models.RecipeDef
		if err := tx.Where("id = ?", recipeID).First(&recipe).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("制造配方不存在")
			}
			return fmt.Errorf("读取制造配方: %w", err)
		}
		var workbench models.HideoutFacility
		// 行级 UPDATE 锁锁定工作台，配合下方作业计数实现单作业互斥。
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND facility_id = ?", userID, workbenchFacilityID).First(&workbench).Error; err != nil {
			return fmt.Errorf("工作台不可用")
		}
		if workbench.State != "ready" {
			return fmt.Errorf("工作台正在升级中")
		}
		if workbench.Level < recipe.RequiredLevel {
			return fmt.Errorf("需要工作台 LV.%d", recipe.RequiredLevel)
		}
		var activeJobs int64
		if err := tx.Model(&models.FacilityJob{}).
			Where("user_id = ? AND facility_id = ? AND status = ?", userID, workbenchFacilityID, facilityJobRunning).
			Count(&activeJobs).Error; err != nil {
			return fmt.Errorf("检查工作台作业: %w", err)
		}
		if activeJobs > 0 {
			return fmt.Errorf("工作台已有作业，请等待当前作业完成")
		}
		inputs, err := parseRecipeInputs(recipe.InputsJSON)
		if err != nil {
			return fmt.Errorf("解析配方材料: %w", err)
		}
		for _, input := range inputs {
			if err := consumeRequirementItemTx(tx, userID, input.ItemID, input.Quantity); err != nil {
				return fmt.Errorf("扣除制造材料 %s: %w", input.ItemID, err)
			}
		}
		output, err := catalog.New(tx).FindByID(recipe.OutputItemID)
		if err != nil {
			return err
		}
		outputSlots, err := craftingOutputSlots(tx, userID, output, recipe.OutputQuantity)
		if err != nil {
			return err
		}
		used, err := inventoryUsage(tx, userID)
		if err != nil {
			return err
		}
		capacity, err := storageCapacityForUser(tx, userID)
		if err != nil {
			return err
		}
		// 入队前容量预检：产物所需格数不得超过当前剩余容量。
		if used+outputSlots > capacity {
			return fmt.Errorf("仓库空间不足：产物需 %d 个空位，当前仅剩 %d 个", outputSlots, capacity-used)
		}
		now := time.Now()
		job := models.FacilityJob{
			UserID: userID, FacilityID: workbenchFacilityID, JobType: facilityJobTypeCraft,
			TargetRef: recipe.ID, StartedAt: now, CompleteAt: now.Add(time.Duration(recipe.CraftSeconds) * time.Second),
			Status: facilityJobRunning,
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("创建制造作业: %w", err)
		}
		return nil
	})
}

// workbenchStatus 返回工作台状态与是否有正在执行的作业。
func workbenchStatus(db *gorm.DB, userID uint) (models.HideoutFacility, bool, error) {
	var workbench models.HideoutFacility
	err := db.Where("user_id = ? AND facility_id = ?", userID, workbenchFacilityID).First(&workbench).Error
	if err == gorm.ErrRecordNotFound {
		workbench.Level = 0
		workbench.State = "ready"
	} else if err != nil {
		return workbench, false, fmt.Errorf("读取工作台状态: %w", err)
	}
	var busy int64
	if err := db.Model(&models.FacilityJob{}).
		Where("user_id = ? AND facility_id = ? AND status = ?", userID, workbenchFacilityID, facilityJobRunning).
		Count(&busy).Error; err != nil {
		return workbench, false, fmt.Errorf("检查工作台作业: %w", err)
	}
	return workbench, busy > 0, nil
}

// parseRecipeInputs 解析配方材料 JSON，空串返回空列表。
func parseRecipeInputs(raw string) ([]models.RecipeInput, error) {
	if raw == "" {
		return nil, nil
	}
	var inputs []models.RecipeInput
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return nil, err
	}
	return inputs, nil
}

// craftingOutputSlots 计算产物入库将占用的容量格数（弹药按每格叠加口径折算）。
func craftingOutputSlots(tx *gorm.DB, userID uint, item catalogItem, quantity int) (int, error) {
	if item.Kind != "ammo" {
		return quantity, nil
	}
	perSlot := item.RoundsPerSlot
	if perSlot <= 0 {
		return 0, fmt.Errorf("弹药 %s 的每格容量无效", item.ID)
	}
	var existing int
	if err := tx.Model(&models.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, item.ID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&existing).Error; err != nil {
		return 0, fmt.Errorf("读取弹药库存 %s: %w", item.ID, err)
	}
	return ceilDiv(existing+quantity, perSlot) - ceilDiv(existing, perSlot), nil
}
