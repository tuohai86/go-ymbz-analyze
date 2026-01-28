package engine

import "sort"

// 车型常量
var (
	BET_LABELS = []string{
		"红奔驰", "绿奔驰", "黄奔驰",
		"红宝马", "绿宝马", "黄宝马",
		"红奥迪", "绿奥迪", "黄奥迪",
		"红大众", "绿大众", "黄大众",
	}

	SMALL_CARS = []string{
		"红大众", "绿大众", "黄大众",
		"红奥迪", "绿奥迪", "黄奥迪",
	}

	BIG_CARS = []string{
		"红奔驰", "绿奔驰", "黄奔驰",
		"红宝马", "绿宝马", "黄宝马",
	}

	// 真实赔率表
	REAL_ODDS = map[string]int{
		"红奔驰": 45, "绿奔驰": 38, "黄奔驰": 27,
		"红宝马": 22, "绿宝马": 16, "黄宝马": 13,
		"红奥迪": 12, "绿奥迪": 10, "黄奥迪": 6,
		"红大众": 7, "绿大众": 5, "黄大众": 4,
	}

	// 特殊奖项列表
	SPECIAL_REWARDS = []string{"大三元", "大四喜", "极速狂飙", "U型过弯", "全民送灯"}
)

// StratHot3 热门3码策略：取热度最高的3个车型
func StratHot3(scores map[string]float64) []string {
	return topN(scores, 3)
}

// StratBalanced4 均衡4码策略：1大车 + 3小车
func StratBalanced4(scores map[string]float64) []string {
	bigTop := topNFromList(scores, BIG_CARS, 1)
	smallTop := topNFromList(scores, SMALL_CARS, 3)
	return append(bigTop, smallTop...)
}

// topN 从分数中取前N个
func topN(scores map[string]float64, n int) []string {
	type kv struct {
		Key   string
		Value float64
	}

	var sorted []kv
	for k, v := range scores {
		sorted = append(sorted, kv{k, v})
	}

	// 按分数降序排序
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// 取前N个
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].Key)
	}

	return result
}

// topNFromList 从指定列表中按分数取前N个
func topNFromList(scores map[string]float64, list []string, n int) []string {
	type kv struct {
		Key   string
		Value float64
	}

	var sorted []kv
	for _, item := range list {
		if score, ok := scores[item]; ok {
			sorted = append(sorted, kv{item, score})
		}
	}

	// 按分数降序排序
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	// 取前N个
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(sorted); i++ {
		result = append(result, sorted[i].Key)
	}

	return result
}
