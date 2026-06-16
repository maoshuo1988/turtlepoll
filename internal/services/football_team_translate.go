package services

import "strings"

var footballTeamZhNameByID = map[int64]string{}

var footballTeamZhNameByName = map[string]string{
	"argentina":                    "阿根廷",
	"algeria":                      "阿尔及利亚",
	"australia":                    "澳大利亚",
	"austria":                      "奥地利",
	"belgium":                      "比利时",
	"bosnia and herzegovina":       "波黑",
	"bosnia-herzegovina":           "波黑",
	"brazil":                       "巴西",
	"cameroon":                     "喀麦隆",
	"canada":                       "加拿大",
	"cape verde islands":           "佛得角",
	"cape verde":                   "佛得角",
	"cabo verde":                   "佛得角",
	"chile":                        "智利",
	"china":                        "中国",
	"colombia":                     "哥伦比亚",
	"costa rica":                   "哥斯达黎加",
	"côte d'ivoire":                "科特迪瓦",
	"cote d'ivoire":                "科特迪瓦",
	"ivory coast":                  "科特迪瓦",
	"dr congo":                     "刚果（金）",
	"congo dr":                     "刚果（金）",
	"congo, dr":                    "刚果（金）",
	"democratic republic of congo": "刚果（金）",
	"croatia":                      "克罗地亚",
	"czech republic":               "捷克",
	"czechia":                      "捷克",
	"denmark":                      "丹麦",
	"ecuador":                      "厄瓜多尔",
	"egypt":                        "埃及",
	"england":                      "英格兰",
	"france":                       "法国",
	"germany":                      "德国",
	"ghana":                        "加纳",
	"greece":                       "希腊",
	"hungary":                      "匈牙利",
	"iceland":                      "冰岛",
	"iran":                         "伊朗",
	"ir iran":                      "伊朗",
	"iraq":                         "伊拉克",
	"italy":                        "意大利",
	"japan":                        "日本",
	"jordan":                       "约旦",
	"korea republic":               "韩国",
	"korea, republic of":           "韩国",
	"south korea":                  "韩国",
	"mexico":                       "墨西哥",
	"morocco":                      "摩洛哥",
	"netherlands":                  "荷兰",
	"new zealand":                  "新西兰",
	"nigeria":                      "尼日利亚",
	"norway":                       "挪威",
	"panama":                       "巴拿马",
	"paraguay":                     "巴拉圭",
	"peru":                         "秘鲁",
	"poland":                       "波兰",
	"portugal":                     "葡萄牙",
	"qatar":                        "卡塔尔",
	"republic of ireland":          "爱尔兰",
	"romania":                      "罗马尼亚",
	"russia":                       "俄罗斯",
	"saudi arabia":                 "沙特阿拉伯",
	"scotland":                     "苏格兰",
	"senegal":                      "塞内加尔",
	"serbia":                       "塞尔维亚",
	"slovakia":                     "斯洛伐克",
	"slovenia":                     "斯洛文尼亚",
	"south africa":                 "南非",
	"spain":                        "西班牙",
	"sweden":                       "瑞典",
	"switzerland":                  "瑞士",
	"tunisia":                      "突尼斯",
	"turkey":                       "土耳其",
	"türkiye":                      "土耳其",
	"ukraine":                      "乌克兰",
	"united arab emirates":         "阿联酋",
	"united states":                "美国",
	"united states of america":     "美国",
	"usa":                          "美国",
	"uruguay":                      "乌拉圭",
	"uzbekistan":                   "乌兹别克斯坦",
	"wales":                        "威尔士",

	// 2026 host/qualification common display variants.
	"curaçao":             "库拉索",
	"curacao":             "库拉索",
	"el salvador":         "萨尔瓦多",
	"guatemala":           "危地马拉",
	"haiti":               "海地",
	"honduras":            "洪都拉斯",
	"jamaica":             "牙买加",
	"trinidad and tobago": "特立尼达和多巴哥",
	"venezuela":           "委内瑞拉",
}

func translateFootballTeamName(teamID int64, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if zh := footballTeamZhNameByID[teamID]; zh != "" {
		return zh
	}
	if zh := footballTeamZhNameByName[normalizeFootballTeamNameKey(name)]; zh != "" {
		return zh
	}
	return name
}

func translateKnownFootballTeamName(teamID int64, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if zh := footballTeamZhNameByID[teamID]; zh != "" {
		return zh, true
	}
	if zh := footballTeamZhNameByName[normalizeFootballTeamNameKey(name)]; zh != "" {
		return zh, true
	}
	return name, false
}

func normalizeFootballTeamNameKey(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.TrimSuffix(name, " national football team")
	name = strings.TrimSuffix(name, " national team")
	name = strings.TrimSuffix(name, " men's national football team")
	name = strings.TrimSuffix(name, " men's national team")
	return strings.Join(strings.Fields(name), " ")
}
