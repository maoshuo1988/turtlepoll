package services

import "sort"

// PredictTagService 标签统计服务（轻量，不落库）。
var PredictTagService = newPredictTagService()

func newPredictTagService() *predictTagService {
	return &predictTagService{}
}

type predictTagService struct{}

type HotTag struct {
	Tag  string `json:"tag"`
	Heat int64  `json:"heat"`
}

// TopN 将 tag->heat 的 map 输出为按 heat 倒序的 TopN。
// 当 heat 相同，按 tag 字典序升序稳定排序。
func (s *predictTagService) TopN(tagHeat map[string]int64, n int) []HotTag {
	if n <= 0 {
		return []HotTag{}
	}
	list := make([]HotTag, 0, len(tagHeat))
	for tag, heat := range tagHeat {
		list = append(list, HotTag{Tag: tag, Heat: heat})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Heat == list[j].Heat {
			return list[i].Tag < list[j].Tag
		}
		return list[i].Heat > list[j].Heat
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}
