package ws

import (
	"testing"

	"goflylivechat/models"
)

// TestSplitAutoReplyKeywords 输入测试上下文，输出为关键词拆分断言结果，目的在于验证多分隔符和去重逻辑正确。
func TestSplitAutoReplyKeywords(t *testing.T) {
	keywords := splitAutoReplyKeywords("价格，报价;价格| 报价 \n多少钱")
	if len(keywords) != 3 {
		t.Fatalf("期望得到 3 个关键词，实际得到 %d", len(keywords))
	}
	if keywords[0] != "价格" || keywords[1] != "报价" || keywords[2] != "多少钱" {
		t.Fatalf("关键词拆分结果不符合预期: %#v", keywords)
	}
}

// TestFindMatchedAutoReplyContent 输入测试上下文，输出为命中断言结果，目的在于验证精确优先与最长关键词包含匹配规则。
func TestFindMatchedAutoReplyContent(t *testing.T) {
	replyGroups := []*models.ReplyGroup{
		{
			Id:        "group-1",
			GroupName: "售前",
			Items: []*models.ReplyItem{
				{ItemName: "价格,报价", Content: "基础报价"},
				{ItemName: "价格表", Content: "完整价格表"},
			},
		},
	}
	autoReplyRules := buildAutoReplyRules(replyGroups)

	exactMatchedContent := findMatchedAutoReplyContent(autoReplyRules, normalizeAutoReplyText("报价"))
	if exactMatchedContent != "基础报价" {
		t.Fatalf("期望精确命中基础报价，实际得到 %s", exactMatchedContent)
	}

	containsMatchedContent := findMatchedAutoReplyContent(autoReplyRules, normalizeAutoReplyText("请发我最新价格表"))
	if containsMatchedContent != "完整价格表" {
		t.Fatalf("期望按最长关键词命中完整价格表，实际得到 %s", containsMatchedContent)
	}
}

// TestBuildAutoReplyRulesSkipsInvalidItems 输入测试上下文，输出为规则构建断言结果，目的在于验证空关键词和空内容不会进入缓存规则。
func TestBuildAutoReplyRulesSkipsInvalidItems(t *testing.T) {
	replyGroups := []*models.ReplyGroup{
		{
			Id:        "group-1",
			GroupName: "默认",
			Items: []*models.ReplyItem{
				{ItemName: "", Content: "无效"},
				{ItemName: "有效词", Content: ""},
				{ItemName: "售后", Content: "售后回复"},
			},
		},
	}
	autoReplyRules := buildAutoReplyRules(replyGroups)
	if len(autoReplyRules) != 1 {
		t.Fatalf("期望仅构建 1 条有效规则，实际得到 %d", len(autoReplyRules))
	}
	if autoReplyRules[0].Content != "售后回复" {
		t.Fatalf("期望保留售后回复规则，实际得到 %s", autoReplyRules[0].Content)
	}
}
