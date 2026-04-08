package ws

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"goflylivechat/models"
	"goflylivechat/tools"
)

const (
	autoReplyCacheKeyPrefix = "livechat:auto_reply:"
	autoReplyCacheDuration  = 12 * time.Hour
)

type AutoReplyRule struct {
	Keywords []string `json:"keywords"`
	Content  string   `json:"content"`
}

// MatchAutoReplyContent 输入客服账号和访客消息，输出为命中的自动回复内容，目的在于优先通过 Redis 缓存完成关键词命中。
func MatchAutoReplyContent(kefuName string, visitorContent string) string {
	normalizedContent := normalizeAutoReplyText(visitorContent)
	if kefuName == "" || normalizedContent == "" {
		return ""
	}
	autoReplyRules := loadAutoReplyRules(kefuName)
	return findMatchedAutoReplyContent(autoReplyRules, normalizedContent)
}

// InvalidateAutoReplyCache 输入客服账号，输出为缓存失效结果，目的在于在回复变更后清理 Redis 中的旧索引。
func InvalidateAutoReplyCache(kefuName string) {
	cache := tools.GetCache()
	if cache == nil || kefuName == "" {
		return
	}
	_ = cache.Del(tools.Ctx, autoReplyCacheKeyPrefix+kefuName)
}

// loadAutoReplyRules 输入客服账号，输出为自动回复规则列表，目的在于优先从 Redis 读取规则并在缺失时回源数据库。
func loadAutoReplyRules(kefuName string) []AutoReplyRule {
	cache := tools.GetCache()
	cacheKey := autoReplyCacheKeyPrefix + kefuName
	if cache != nil {
		cachedRules, cacheError := cache.Get(tools.Ctx, cacheKey)
		if cacheError == nil && cachedRules != "" {
			var autoReplyRules []AutoReplyRule
			if unmarshalError := json.Unmarshal([]byte(cachedRules), &autoReplyRules); unmarshalError == nil {
				return autoReplyRules
			}
		}
	}

	autoReplyRules := buildAutoReplyRules(models.FindReplyByUserId(kefuName))
	if cache != nil {
		payload, marshalError := json.Marshal(autoReplyRules)
		if marshalError == nil {
			_ = cache.Set(tools.Ctx, cacheKey, string(payload), autoReplyCacheDuration)
		}
	}
	return autoReplyRules
}

// buildAutoReplyRules 输入回复分组列表，输出为排序后的规则列表，目的在于生成可直接执行的关键词匹配索引。
func buildAutoReplyRules(replyGroups []*models.ReplyGroup) []AutoReplyRule {
	autoReplyRules := make([]AutoReplyRule, 0)
	for _, replyGroup := range replyGroups {
		for _, replyItem := range replyGroup.Items {
			keywords := splitAutoReplyKeywords(replyItem.ItemName)
			if len(keywords) == 0 || strings.TrimSpace(replyItem.Content) == "" {
				continue
			}
			sort.SliceStable(keywords, func(leftIndex, rightIndex int) bool {
				return len(keywords[leftIndex]) > len(keywords[rightIndex])
			})
			autoReplyRules = append(autoReplyRules, AutoReplyRule{
				Keywords: keywords,
				Content:  replyItem.Content,
			})
		}
	}
	sort.SliceStable(autoReplyRules, func(leftIndex, rightIndex int) bool {
		return len(autoReplyRules[leftIndex].Keywords[0]) > len(autoReplyRules[rightIndex].Keywords[0])
	})
	return autoReplyRules
}

// splitAutoReplyKeywords 输入原始关键词字符串，输出为归一化后的关键词列表，目的在于支持多关键词配置。
func splitAutoReplyKeywords(rawKeywords string) []string {
	replacer := strings.NewReplacer("，", ",", "；", ",", ";", ",", "|", ",", "\n", ",", "\r", ",")
	segments := strings.Split(replacer.Replace(rawKeywords), ",")
	keywords := make([]string, 0, len(segments))
	keywordSet := make(map[string]struct{}, len(segments))
	for _, segment := range segments {
		normalizedKeyword := normalizeAutoReplyText(segment)
		if normalizedKeyword == "" {
			continue
		}
		if _, exists := keywordSet[normalizedKeyword]; exists {
			continue
		}
		keywordSet[normalizedKeyword] = struct{}{}
		keywords = append(keywords, normalizedKeyword)
	}
	return keywords
}

// normalizeAutoReplyText 输入任意文本，输出为归一化文本，目的在于消除大小写和空白差异带来的匹配偏差。
func normalizeAutoReplyText(rawText string) string {
	joinedText := strings.Join(strings.Fields(strings.TrimSpace(rawText)), " ")
	return strings.ToLower(joinedText)
}

// findMatchedAutoReplyContent 输入规则列表和归一化消息内容，输出为命中的回复内容，目的在于按精确优先、最长关键词次之完成匹配。
func findMatchedAutoReplyContent(autoReplyRules []AutoReplyRule, normalizedContent string) string {
	for _, autoReplyRule := range autoReplyRules {
		for _, keyword := range autoReplyRule.Keywords {
			if normalizedContent == keyword {
				return autoReplyRule.Content
			}
		}
	}
	for _, autoReplyRule := range autoReplyRules {
		for _, keyword := range autoReplyRule.Keywords {
			if strings.Contains(normalizedContent, keyword) {
				return autoReplyRule.Content
			}
		}
	}
	return ""
}
