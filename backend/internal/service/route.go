// 单向地图路线：校验固定长度的正向节点链，并提供运行时的下一节点解析。
package service

import (
	"fmt"

	"idle/internal/models"
)

func validateDirectedRoute(nodes []models.NodeDef, gameMap models.MapDef) error {
	byID := make(map[string]models.NodeDef, len(nodes))
	byOrder := make(map[int]string, len(nodes))
	for _, node := range nodes {
		if node.MapID != gameMap.ID {
			continue
		}
		if node.RouteOrder <= 0 {
			return fmt.Errorf("节点 %s 缺少有效路线顺序", node.ID)
		}
		if _, exists := byID[node.ID]; exists {
			return fmt.Errorf("地图 %s 存在重复节点 %s", gameMap.ID, node.ID)
		}
		if previous, exists := byOrder[node.RouteOrder]; exists {
			return fmt.Errorf("地图 %s 的路线顺序 %d 同时分配给 %s 和 %s", gameMap.ID, node.RouteOrder, previous, node.ID)
		}
		byID[node.ID] = node
		byOrder[node.RouteOrder] = node.ID
	}
	start, ok := byID[gameMap.StartNodeID]
	if !ok {
		return fmt.Errorf("起点节点 %s 不存在", gameMap.StartNodeID)
	}
	if _, ok := byID[gameMap.ExtractionNodeID]; !ok {
		return fmt.Errorf("撤离节点 %s 不存在", gameMap.ExtractionNodeID)
	}
	if len(byID) == 0 {
		return fmt.Errorf("地图 %s 没有节点", gameMap.ID)
	}
	if start.RouteOrder != 1 {
		return fmt.Errorf("地图 %s 的起点路线顺序必须为1", gameMap.ID)
	}

	visited := make(map[string]bool, len(byID))
	current := start
	for step := 0; step < len(byID); step++ {
		if visited[current.ID] {
			return fmt.Errorf("地图 %s 存在路线环路，节点 %s 被重复访问", gameMap.ID, current.ID)
		}
		visited[current.ID] = true
		if current.ID == gameMap.ExtractionNodeID {
			if len(splitIDs(current.Connections)) > 0 {
				return fmt.Errorf("撤离节点 %s 不允许继续连接", current.ID)
			}
			break
		}
		nextIDs := splitIDs(current.Connections)
		if len(nextIDs) != 1 {
			return fmt.Errorf("节点 %s 必须只有一个向前出口", current.ID)
		}
		next, exists := byID[nextIDs[0]]
		if !exists {
			return fmt.Errorf("节点 %s 指向不存在的节点 %s", current.ID, nextIDs[0])
		}
		if next.RouteOrder != current.RouteOrder+1 {
			return fmt.Errorf("节点 %s 必须连接路线顺序%d，实际指向顺序%d的%s", current.ID, current.RouteOrder+1, next.RouteOrder, next.ID)
		}
		current = next
	}
	if !visited[gameMap.ExtractionNodeID] {
		return fmt.Errorf("地图 %s 的起点无法到达撤离点", gameMap.ID)
	}
	if len(visited) != len(byID) {
		return fmt.Errorf("地图 %s 存在不在主路线上的节点", gameMap.ID)
	}
	if byID[gameMap.ExtractionNodeID].RouteOrder != len(byID) {
		return fmt.Errorf("地图 %s 的撤离点必须是路线最后一站", gameMap.ID)
	}
	return nil
}

func nextForwardNode(current models.NodeDef, byID map[string]models.NodeDef) (models.NodeDef, bool, error) {
	nextIDs := splitIDs(current.Connections)
	if len(nextIDs) == 0 {
		return models.NodeDef{}, false, nil
	}
	if len(nextIDs) != 1 {
		return models.NodeDef{}, false, fmt.Errorf("节点 %s 存在多个出口，当前固定路线不支持分支", current.ID)
	}
	next, ok := byID[nextIDs[0]]
	if !ok {
		return models.NodeDef{}, false, fmt.Errorf("节点 %s 指向不存在的节点 %s", current.ID, nextIDs[0])
	}
	if next.RouteOrder != current.RouteOrder+1 {
		return models.NodeDef{}, false, fmt.Errorf("节点 %s 的下一节点%s不是紧邻的向前路线", current.ID, next.ID)
	}
	return next, true, nil
}
