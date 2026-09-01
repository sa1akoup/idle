// 地图图算法：校验拓扑、枚举受限简单路径并按行动风格选择路线。
package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// RoutePlan 是单局开始时固定下来的探索节点序列和撤离终点。
type RoutePlan struct {
	NodeIDs      []string `json:"nodeIds"`
	ExtractionID string   `json:"extractionId"`
	AnchorNodeID string   `json:"anchorNodeId"`
}

// RoutePlannerOptions 为复杂地图预留路径枚举上限。
type RoutePlannerOptions struct {
	MaxRouteNodes  int
	MaxCandidates  int
	MaxDetourRatio float64
}

type routeCandidate struct {
	plan        RoutePlan
	value       int
	risk        int
	moveTime    int
	exploreTime int
	length      int
	score       int64
}

type graphNeighbor struct {
	nodeID   string
	moveTime int
}

// maxInt 返回 value 与 floor 中的较大值（最小值下限约束）。
func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

// clamp 把浮点值钳制到 [min, max] 区间。
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// containsString 判断切片中是否存在指定字符串。
func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// sortedNodes 按 Y、X、ID 稳定排序节点，保证枚举顺序确定、可重放。
func sortedNodes(nodes []Node) []Node {
	result := append([]Node(nil), nodes...)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].PositionY == result[j].PositionY {
			if result[i].PositionX == result[j].PositionX {
				return result[i].ID < result[j].ID
			}
			return result[i].PositionX < result[j].PositionX
		}
		return result[i].PositionY < result[j].PositionY
	})
	return result
}

// ValidateMapGraph 校验节点、边和撤离点的引用完整性及可达性。
func ValidateMapGraph(gameMap Map, nodes []Node, edges []MapEdge, points []ExtractionPoint) error {
	if gameMap.ID == "" {
		return fmt.Errorf("地图缺少 ID")
	}
	if gameMap.LayoutColumns <= 0 || gameMap.LayoutRows <= 0 {
		return fmt.Errorf("地图 %s 的布局尺寸无效", gameMap.ID)
	}
	byID := make(map[string]Node, len(nodes))
	for _, node := range nodes {
		if node.ID == "" || node.MapID != gameMap.ID {
			return fmt.Errorf("节点 %s 的地图引用无效", node.ID)
		}
		if _, exists := byID[node.ID]; exists {
			return fmt.Errorf("地图 %s 存在重复节点 %s", gameMap.ID, node.ID)
		}
		if node.ExploreTime < 0 {
			return fmt.Errorf("节点 %s 的探索时间无效", node.ID)
		}
		if node.EncounterChance < 0 || node.EncounterChance > 100 {
			return fmt.Errorf("节点 %s 的遇敌概率无效", node.ID)
		}
		byID[node.ID] = node
	}
	if len(byID) == 0 {
		return fmt.Errorf("地图 %s 没有节点", gameMap.ID)
	}
	if _, ok := byID[gameMap.StartNodeID]; !ok {
		return fmt.Errorf("起点节点 %s 不存在", gameMap.StartNodeID)
	}

	adjacency := make(map[string][]graphNeighbor, len(byID))
	seenDirected := make(map[string]bool, len(edges)*2)
	for _, edge := range edges {
		if edge.MapID != gameMap.ID {
			return fmt.Errorf("边 %d 的地图引用无效", edge.ID)
		}
		if edge.FromNodeID == "" || edge.ToNodeID == "" || edge.FromNodeID == edge.ToNodeID {
			return fmt.Errorf("边 %d 存在空节点或自环", edge.ID)
		}
		if edge.MoveTime <= 0 {
			return fmt.Errorf("边 %d 的移动时间必须大于 0", edge.ID)
		}
		if _, ok := byID[edge.FromNodeID]; !ok {
			return fmt.Errorf("边 %d 引用不存在节点 %s", edge.ID, edge.FromNodeID)
		}
		if _, ok := byID[edge.ToNodeID]; !ok {
			return fmt.Errorf("边 %d 引用不存在节点 %s", edge.ID, edge.ToNodeID)
		}
		if !addGraphArc(adjacency, seenDirected, edge.FromNodeID, edge.ToNodeID, edge.MoveTime) {
			return fmt.Errorf("地图 %s 存在重复边 %s -> %s", gameMap.ID, edge.FromNodeID, edge.ToNodeID)
		}
		if edge.Bidirectional && !addGraphArc(adjacency, seenDirected, edge.ToNodeID, edge.FromNodeID, edge.MoveTime) {
			return fmt.Errorf("地图 %s 存在重复反向边 %s -> %s", gameMap.ID, edge.ToNodeID, edge.FromNodeID)
		}
	}
	for nodeID := range adjacency {
		sort.SliceStable(adjacency[nodeID], func(i, j int) bool {
			if adjacency[nodeID][i].nodeID == adjacency[nodeID][j].nodeID {
				return adjacency[nodeID][i].moveTime < adjacency[nodeID][j].moveTime
			}
			return adjacency[nodeID][i].nodeID < adjacency[nodeID][j].nodeID
		})
	}

	pointIDs := make(map[string]bool, len(points))
	activePoints := 0
	for _, point := range points {
		if point.ID == "" || point.MapID != gameMap.ID || pointIDs[point.ID] {
			return fmt.Errorf("撤离点 %s 的地图引用或 ID 无效", point.ID)
		}
		pointIDs[point.ID] = true
		if point.AnchorNodeID == "" {
			return fmt.Errorf("撤离点 %s 缺少锚点节点", point.ID)
		}
		if _, ok := byID[point.AnchorNodeID]; !ok {
			return fmt.Errorf("撤离点 %s 引用不存在锚点 %s", point.ID, point.AnchorNodeID)
		}
		if point.TravelTime <= 0 {
			return fmt.Errorf("撤离点 %s 的旅行时间必须大于 0", point.ID)
		}
		if point.Enabled {
			activePoints++
		}
	}
	if activePoints == 0 {
		return fmt.Errorf("地图 %s 没有启用的撤离点", gameMap.ID)
	}

	reachable := reachableNodes(gameMap.StartNodeID, adjacency)
	for _, point := range points {
		if point.Enabled && !reachable[point.AnchorNodeID] {
			return fmt.Errorf("撤离点 %s 的锚点不可达", point.ID)
		}
	}
	return nil
}

// addGraphArc 向邻接表追加一条有向边，重复的 from->to 返回 false。
func addGraphArc(adjacency map[string][]graphNeighbor, seen map[string]bool, from, to string, moveTime int) bool {
	key := from + "\x00" + to
	if seen[key] {
		return false
	}
	seen[key] = true
	adjacency[from] = append(adjacency[from], graphNeighbor{nodeID: to, moveTime: moveTime})
	return true
}

// reachableNodes 用 BFS 求起点可达的节点集合。
func reachableNodes(start string, adjacency map[string][]graphNeighbor) map[string]bool {
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if seen[next.nodeID] {
				continue
			}
			seen[next.nodeID] = true
			queue = append(queue, next.nodeID)
		}
	}
	return seen
}

// PlanRoute 以受限 DFS 枚举无重复节点路线，随后按行动风格选出一条固定路线。
func PlanRoute(snapshot ScenarioSnapshot, style string, rng *rand.Rand, options RoutePlannerOptions) (RoutePlan, error) {
	if err := ValidateMapGraph(snapshot.Map, snapshot.Nodes, snapshot.Edges, snapshot.ExtractionPoints); err != nil {
		return RoutePlan{}, err
	}
	policy := stylePolicy(snapshot.Styles, style)
	if options.MaxRouteNodes <= 0 || options.MaxRouteNodes > len(snapshot.Nodes) {
		options.MaxRouteNodes = len(snapshot.Nodes)
	}
	if options.MaxCandidates <= 0 {
		options.MaxCandidates = 512
	}
	adjacency := buildAdjacency(snapshot.Edges)
	nodesByID := make(map[string]Node, len(snapshot.Nodes))
	for _, node := range snapshot.Nodes {
		nodesByID[node.ID] = node
	}

	activePoints := make([]ExtractionPoint, 0, len(snapshot.ExtractionPoints))
	for _, point := range snapshot.ExtractionPoints {
		if point.Enabled {
			activePoints = append(activePoints, point)
		}
	}
	sort.SliceStable(activePoints, func(i, j int) bool { return activePoints[i].ID < activePoints[j].ID })

	// 出生点：从非撤离锚点的节点中确定式随机挑选；rng 缺省时退回地图默认起点。
	startNodeID := snapshot.Map.StartNodeID
	if rng != nil {
		anchorNodeIDs := make(map[string]struct{}, len(activePoints))
		for _, point := range activePoints {
			anchorNodeIDs[point.AnchorNodeID] = struct{}{}
		}
		spawnCandidates := make([]string, 0, len(snapshot.Nodes))
		for _, node := range snapshot.Nodes {
			if _, isAnchor := anchorNodeIDs[node.ID]; isAnchor {
				continue
			}
			spawnCandidates = append(spawnCandidates, node.ID)
		}
		if len(spawnCandidates) > 0 {
			sort.Strings(spawnCandidates)
			startNodeID = spawnCandidates[rng.Intn(len(spawnCandidates))]
		}
	}
	if _, ok := nodesByID[startNodeID]; !ok {
		return RoutePlan{}, fmt.Errorf("起点节点 %s 不存在", startNodeID)
	}

	candidates := make([]routeCandidate, 0, options.MaxCandidates)
	for _, point := range activePoints {
		path := []string{startNodeID}
		visited := map[string]bool{startNodeID: true}
		enumerateRoutes(startNodeID, point.AnchorNodeID, path, visited, adjacency, nodesByID, point, options, &candidates)
		if len(candidates) >= options.MaxCandidates {
			break
		}
	}
	if len(candidates) == 0 {
		return RoutePlan{}, fmt.Errorf("地图 %s 没有满足限制的可行路线", snapshot.Map.ID)
	}
	candidates = filterDetours(candidates, options.MaxDetourRatio)
	if len(candidates) == 0 {
		return RoutePlan{}, fmt.Errorf("地图 %s 的路线均超过绕行限制", snapshot.Map.ID)
	}
	normalizeCandidateScores(candidates, policy)
	bestScore := candidates[0].score
	for _, candidate := range candidates[1:] {
		if candidate.score > bestScore {
			bestScore = candidate.score
		}
	}
	best := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.score == bestScore {
			best = append(best, candidate)
		}
	}
	selected := 0
	if rng != nil && len(best) > 1 {
		selected = rng.Intn(len(best))
	}
	return best[selected].plan, nil
}

// buildAdjacency 由边列表构建邻接表，并对每个节点的邻接排序保证确定性。
func buildAdjacency(edges []MapEdge) map[string][]graphNeighbor {
	adjacency := make(map[string][]graphNeighbor)
	for _, edge := range edges {
		adjacency[edge.FromNodeID] = append(adjacency[edge.FromNodeID], graphNeighbor{nodeID: edge.ToNodeID, moveTime: edge.MoveTime})
		if edge.Bidirectional {
			adjacency[edge.ToNodeID] = append(adjacency[edge.ToNodeID], graphNeighbor{nodeID: edge.FromNodeID, moveTime: edge.MoveTime})
		}
	}
	for nodeID := range adjacency {
		sort.SliceStable(adjacency[nodeID], func(i, j int) bool {
			if adjacency[nodeID][i].nodeID == adjacency[nodeID][j].nodeID {
				return adjacency[nodeID][i].moveTime < adjacency[nodeID][j].moveTime
			}
			return adjacency[nodeID][i].nodeID < adjacency[nodeID][j].nodeID
		})
	}
	return adjacency
}

// enumerateRoutes 受限 DFS 枚举从起点到锚点的无重复节点路径，并累计各维度代价。
func enumerateRoutes(current, anchor string, path []string, visited map[string]bool, adjacency map[string][]graphNeighbor, nodesByID map[string]Node, point ExtractionPoint, options RoutePlannerOptions, candidates *[]routeCandidate) {
	if len(*candidates) >= options.MaxCandidates || len(path) > options.MaxRouteNodes {
		return
	}
	if current == anchor {
		candidate := routeCandidate{plan: RoutePlan{NodeIDs: append([]string(nil), path...), ExtractionID: point.ID, AnchorNodeID: anchor}, length: len(path)}
		for _, nodeID := range path {
			node := nodesByID[nodeID]
			candidate.value += node.ValueTier
			candidate.risk += nodeRisk(node)
			candidate.exploreTime += node.ExploreTime
		}
		for index := 1; index < len(path); index++ {
			candidate.moveTime += edgeMoveTime(adjacency[path[index-1]], path[index])
		}
		candidate.moveTime += point.TravelTime
		*candidates = append(*candidates, candidate)
		return
	}
	for _, next := range adjacency[current] {
		if visited[next.nodeID] {
			continue
		}
		visited[next.nodeID] = true
		enumerateRoutes(next.nodeID, anchor, append(path, next.nodeID), visited, adjacency, nodesByID, point, options, candidates)
		delete(visited, next.nodeID)
		if len(*candidates) >= options.MaxCandidates {
			return
		}
	}
}

// filterDetours 以同一撤离点的最短耗时做基准，剔除移动耗时超比例的绕行路线。
func filterDetours(candidates []routeCandidate, ratio float64) []routeCandidate {
	if ratio <= 0 || math.IsInf(ratio, 0) || math.IsNaN(ratio) {
		return candidates
	}
	minimum := make(map[string]int)
	for _, candidate := range candidates {
		if current, ok := minimum[candidate.plan.ExtractionID]; !ok || candidate.moveTime < current {
			minimum[candidate.plan.ExtractionID] = candidate.moveTime
		}
	}
	// 以每个撤离点的最短移动耗时作为基准线，保留未超过 ratio 倍的路线。
	filtered := make([]routeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if float64(candidate.moveTime) <= float64(minimum[candidate.plan.ExtractionID])*ratio {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

// normalizeCandidateScores 按行动风格权重把各维度归一化后合并成路线得分。
func normalizeCandidateScores(candidates []routeCandidate, policy StylePolicy) {
	maxValue, maxRisk, maxMove, maxExplore, maxLength := 1, 1, 1, 1, 1
	for _, candidate := range candidates {
		maxValue = maxInt(maxValue, candidate.value)
		maxRisk = maxInt(maxRisk, candidate.risk)
		maxMove = maxInt(maxMove, candidate.moveTime)
		maxExplore = maxInt(maxExplore, candidate.exploreTime)
		maxLength = maxInt(maxLength, candidate.length)
	}
	for index := range candidates {
		candidate := &candidates[index]
		// 各维度先按最大值归一到 0-1000 再按权重加权，避免量纲差异主导评分。
		value := int64(candidate.value * 1000 / maxValue)
		risk := int64(candidate.risk * 1000 / maxRisk)
		move := int64(candidate.moveTime * 1000 / maxMove)
		explore := int64(candidate.exploreTime * 1000 / maxExplore)
		length := int64(candidate.length * 1000 / maxLength)
		candidate.score = value*int64(policy.ValueWeight) - risk*int64(policy.RiskWeight) -
			move*int64(policy.MoveTimeWeight) - explore*int64(policy.ExploreTimeWeight) - length*int64(policy.LengthWeight)
	}
}

// nodeRisk 按遭遇角色返回风险等级，用于路线评分惩罚高危节点。
func nodeRisk(node Node) int {
	switch node.EncounterRole {
	case "elite":
		return 5
	case "extraction":
		return 4
	case "guard":
		return 3
	case "sniper":
		return 4
	case "patrol":
		return 2
	default:
		return 1
	}
}

// edgeMoveTime 查邻接表取两点间的移动耗时，不存在边时返回 0。
func edgeMoveTime(neighbors []graphNeighbor, target string) int {
	for _, neighbor := range neighbors {
		if neighbor.nodeID == target {
			return neighbor.moveTime
		}
	}
	return 0
}

// extractionPointByID 按 ID 查找撤离点，未找到时返回 false。
func extractionPointByID(points []ExtractionPoint, id string) (ExtractionPoint, bool) {
	for _, point := range points {
		if point.ID == id {
			return point, true
		}
	}
	return ExtractionPoint{}, false
}

// ValidateRoutePlan 校验一条已规划路线仍然属于当前快照图。
func ValidateRoutePlan(snapshot ScenarioSnapshot, plan RoutePlan) error {
	if len(plan.NodeIDs) == 0 {
		return fmt.Errorf("路线为空")
	}
	// v6 起点从非锚点节点中随机选择，这里只约束"起点不得是撤离锚点"。
	anchorNodeIDs := make(map[string]struct{}, len(snapshot.ExtractionPoints))
	for _, point := range snapshot.ExtractionPoints {
		if point.Enabled {
			anchorNodeIDs[point.AnchorNodeID] = struct{}{}
		}
	}
	if _, forbidden := anchorNodeIDs[plan.NodeIDs[0]]; forbidden {
		return fmt.Errorf("路线起点不得是撤离锚点 %s", plan.NodeIDs[0])
	}
	if plan.ExtractionID == "" || plan.AnchorNodeID == "" {
		return fmt.Errorf("路线缺少撤离点或锚点")
	}
	points := make(map[string]ExtractionPoint, len(snapshot.ExtractionPoints))
	for _, point := range snapshot.ExtractionPoints {
		points[point.ID] = point
	}
	point, ok := points[plan.ExtractionID]
	if !ok || !point.Enabled || point.AnchorNodeID != plan.AnchorNodeID {
		return fmt.Errorf("路线引用的撤离点无效 %s", plan.ExtractionID)
	}
	if plan.NodeIDs[len(plan.NodeIDs)-1] != plan.AnchorNodeID {
		return fmt.Errorf("路线终点不是撤离锚点 %s", plan.AnchorNodeID)
	}
	seen := make(map[string]bool, len(plan.NodeIDs))
	for _, nodeID := range plan.NodeIDs {
		if seen[nodeID] {
			return fmt.Errorf("路线重复经过节点 %s", nodeID)
		}
		seen[nodeID] = true
	}
	adjacency := buildAdjacency(snapshot.Edges)
	for index := 1; index < len(plan.NodeIDs); index++ {
		if edgeMoveTime(adjacency[plan.NodeIDs[index-1]], plan.NodeIDs[index]) <= 0 {
			return fmt.Errorf("路线缺少边 %s -> %s", plan.NodeIDs[index-1], plan.NodeIDs[index])
		}
	}
	return nil
}
