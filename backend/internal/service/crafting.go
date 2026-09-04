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
	FacilityID     string                 `json:"facilityId"`
	FacilityName   string                 `json:"facilityName"`
	FacilityLevel  int                    `json:"facilityLevel"`
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
	statuses, err := craftFacilityStatuses(db, userID)
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
			have, err := ownedRaidExtractQuantityTx(db, userID, input.ItemID)
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
		status := statuses[recipe.FacilityID]
		if status.Name == "" {
			status.Name = recipe.FacilityID
			status.State = "ready"
		}
		levelReady := status.Level >= recipe.RequiredLevel
		canStart := levelReady && status.State == "ready" && !status.Busy && allSatisfied
		reason := ""
		switch {
		case !levelReady:
			reason = fmt.Sprintf("需要%s LV.%d", status.Name, recipe.RequiredLevel)
		case status.State != "ready":
			reason = status.Name + "正在升级中"
		case status.Busy:
			reason = status.Name + "正在执行作业"
		case !allSatisfied:
			reason = "局内带出材料不足"
		}
		views = append(views, CraftingRecipeView{
			ID: recipe.ID, Name: recipe.Name, RequiredLevel: recipe.RequiredLevel,
			CraftSeconds: recipe.CraftSeconds, CraftMinutes: int((recipe.CraftSeconds + 59) / 60),
			Output: CraftingOutputView{
				ItemID: output.ID, Name: output.Name, Kind: output.Kind,
				Quantity: recipe.OutputQuantity, InstanceRequired: instanceRequired,
			},
			Inputs: materials, FacilityID: recipe.FacilityID, FacilityName: status.Name,
			FacilityLevel: status.Level, WorkbenchLevel: status.Level, WorkbenchBusy: status.Busy,
			CanStart: canStart, Reason: reason,
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
		facilityID := recipe.FacilityID
		if facilityID == "" {
			facilityID = workbenchFacilityID
		}
		var facility models.HideoutFacility
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND facility_id = ?", userID, facilityID).First(&facility).Error; err != nil {
			return fmt.Errorf("制造设施不可用")
		}
		facilityName := craftFacilityName(tx, facilityID)
		if facility.State != "ready" {
			return fmt.Errorf("%s正在升级中", facilityName)
		}
		if facility.Level < recipe.RequiredLevel {
			return fmt.Errorf("需要%s LV.%d", facilityName, recipe.RequiredLevel)
		}
		var activeJobs int64
		if err := tx.Model(&models.FacilityJob{}).
			Where("user_id = ? AND facility_id = ? AND status = ?", userID, facilityID, facilityJobRunning).
			Count(&activeJobs).Error; err != nil {
			return fmt.Errorf("检查制造作业: %w", err)
		}
		if activeJobs > 0 {
			return fmt.Errorf("%s已有作业，请等待当前作业完成", facilityName)
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
			UserID: userID, FacilityID: facilityID, JobType: facilityJobTypeCraft,
			TargetRef: recipe.ID, StartedAt: now, CompleteAt: now.Add(time.Duration(recipe.CraftSeconds) * time.Second),
			Status: facilityJobRunning,
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("创建制造作业: %w", err)
		}
		return nil
	})
}

type craftFacilityStatus struct {
	ID    string
	Name  string
	Level int
	State string
	Busy  bool
}

func craftFacilityStatuses(db *gorm.DB, userID uint) (map[string]craftFacilityStatus, error) {
	var defs []models.FacilityDef
	if err := db.Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("读取设施名称: %w", err)
	}
	names := make(map[string]string, len(defs))
	for _, def := range defs {
		names[def.ID] = def.Name
	}
	var states []models.HideoutFacility
	if err := db.Where("user_id = ?", userID).Find(&states).Error; err != nil {
		return nil, fmt.Errorf("读取制造设施: %w", err)
	}
	result := make(map[string]craftFacilityStatus, len(states))
	for _, state := range states {
		var busy int64
		if err := db.Model(&models.FacilityJob{}).
			Where("user_id = ? AND facility_id = ? AND status = ?", userID, state.FacilityID, facilityJobRunning).
			Count(&busy).Error; err != nil {
			return nil, fmt.Errorf("检查%s作业: %w", names[state.FacilityID], err)
		}
		name := names[state.FacilityID]
		if name == "" {
			name = state.FacilityID
		}
		result[state.FacilityID] = craftFacilityStatus{
			ID: state.FacilityID, Name: name, Level: state.Level, State: state.State, Busy: busy > 0,
		}
	}
	return result, nil
}

func craftFacilityName(tx *gorm.DB, facilityID string) string {
	var def models.FacilityDef
	if err := tx.Where("id = ?", facilityID).First(&def).Error; err != nil || def.Name == "" {
		if facilityID == workbenchFacilityID {
			return "工作台"
		}
		return facilityID
	}
	return def.Name
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
