// 地图路线规划测试：验证九宫格简单路径数量、路线方向和节点不重复约束。
package engine

import (
	"strconv"
	"testing"
)

func TestNineGridSimplePathCounts(t *testing.T) {
	snapshot := nineGridRouteSnapshot()
	adjacency := buildAdjacency(snapshot.Edges)
	nodesByID := make(map[string]Node, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		nodesByID[node.ID] = node
	}

	wantLengths := map[int]int{3: 2, 5: 2, 7: 2, 9: 2}
	for _, point := range snapshot.ExtractionPoints {
		candidates := make([]routeCandidate, 0, 16)
		enumerateRoutes(
			snapshot.Map.StartNodeID,
			point.AnchorNodeID,
			[]string{snapshot.Map.StartNodeID},
			map[string]bool{snapshot.Map.StartNodeID: true},
			adjacency,
			nodesByID,
			point,
			RoutePlannerOptions{MaxRouteNodes: 9, MaxCandidates: 64},
			&candidates,
		)
		if len(candidates) != 8 {
			t.Fatalf("撤离点 %s 的简单路径数量 = %d，期望 8", point.ID, len(candidates))
		}

		lengths := make(map[int]int)
		for _, candidate := range candidates {
			if err := ValidateRoutePlan(snapshot, candidate.plan); err != nil {
				t.Fatalf("撤离点 %s 生成了无效路线 %v: %v", point.ID, candidate.plan.NodeIDs, err)
			}
			lengths[len(candidate.plan.NodeIDs)]++
		}
		if len(lengths) != len(wantLengths) {
			t.Fatalf("撤离点 %s 的路线长度分布 = %v，期望 %v", point.ID, lengths, wantLengths)
		}
		for length, want := range wantLengths {
			if lengths[length] != want {
				t.Fatalf("撤离点 %s 的 %d 节点路线数量 = %d，期望 %d", point.ID, length, lengths[length], want)
			}
		}
	}
}

func nineGridRouteSnapshot() ScenarioSnapshot {
	nodes := make([]Node, 0, 9)
	for index := 1; index <= 9; index++ {
		nodes = append(nodes, Node{
			ID: "node_" + strconv.Itoa(index), MapID: "map_grid", PositionX: (index - 1) % 3, PositionY: (index - 1) / 3,
			Name: "节点", ExploreTime: 1, ValueTier: 1,
		})
	}
	gridEdges := [][2]int{{1, 2}, {1, 4}, {2, 3}, {2, 5}, {3, 6}, {4, 5}, {4, 7}, {5, 6}, {5, 8}, {6, 9}, {7, 8}, {8, 9}}
	edges := make([]MapEdge, 0, len(gridEdges))
	for index, pair := range gridEdges {
		edges = append(edges, MapEdge{ID: uint(index + 1), MapID: "map_grid", FromNodeID: "node_" + strconv.Itoa(pair[0]), ToNodeID: "node_" + strconv.Itoa(pair[1]), MoveTime: 1, Bidirectional: true})
	}
	return ScenarioSnapshot{
		Map:   Map{ID: "map_grid", StartNodeID: "node_5", LayoutColumns: 3, LayoutRows: 3},
		Nodes: nodes, Edges: edges,
		ExtractionPoints: []ExtractionPoint{
			{ID: "extract_1", MapID: "map_grid", AnchorNodeID: "node_1", TravelTime: 1, Enabled: true},
			{ID: "extract_9", MapID: "map_grid", AnchorNodeID: "node_9", TravelTime: 1, Enabled: true},
		},
	}
}
