package service

import (
	"fmt"

	"idle/internal/models"

	"gorm.io/gorm"
)

type InventoryItemView struct {
	models.Inventory
	Purposes []string `json:"purposes"`
}

func ListInventoryForUser(db *gorm.DB, userID uint) ([]InventoryItemView, error) {
	var rows []models.Inventory
	if err := db.Where("user_id = ?", userID).Order("kind asc, name asc, id asc").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("读取仓库: %w", err)
	}
	purposes, err := itemPurposeLabelsTx(db, userID)
	if err != nil {
		return nil, err
	}
	views := make([]InventoryItemView, 0, len(rows))
	for _, row := range rows {
		views = append(views, InventoryItemView{Inventory: row, Purposes: purposeLabelsFor(row.ItemID, row.RaidExtract, purposes)})
	}
	return views, nil
}

func purposeLabelsFor(itemID string, raidExtract bool, purposes map[string][]string) []string {
	if !raidExtract {
		return nil
	}
	return append([]string{"局内带出"}, purposes[itemID]...)
}

func itemPurposeLabelsTx(db *gorm.DB, userID uint) (map[string][]string, error) {
	labels := make(map[string][]string)
	add := func(itemID, label string) {
		if itemID == "" || label == "" {
			return
		}
		for _, existing := range labels[itemID] {
			if existing == label {
				return
			}
		}
		labels[itemID] = append(labels[itemID], label)
	}
	quests, err := ListQuestsForUser(db, userID)
	if err != nil {
		return nil, err
	}
	for _, quest := range quests {
		if quest.Status != models.QuestStatusActive || quest.ObjectiveType != models.QuestObjectiveExtractItem || quest.TargetID == "" {
			continue
		}
		add(quest.TargetID, "合同："+quest.Name)
	}
	hideout, err := GetHideoutForUser(db, userID)
	if err != nil {
		return nil, err
	}
	if hideout != nil {
		for _, facility := range hideout.Facilities {
			if facility.NextUpgrade == nil {
				continue
			}
			for _, requirement := range facility.NextUpgrade.Requirements {
				if requirement.RequirementType != "item" || requirement.ReferenceID == "" {
					continue
				}
				add(requirement.ReferenceID, fmt.Sprintf("%s LV.%d", facility.Name, facility.NextUpgrade.Level))
			}
		}
	}
	return labels, nil
}
