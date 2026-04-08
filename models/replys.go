package models

type ReplyItem struct {
	Id       string `json:"item_id"`
	Content  string `json:"item_content"`
	GroupId  string `json:"group_id"`
	ItemName string `json:"item_name"`
	UserId   string `json:"user_id"`
}

type ReplyGroup struct {
	Id        string       `json:"group_id"`
	GroupName string       `json:"group_name"`
	UserId    string       `json:"user_id"`
	Items     []*ReplyItem `json:"items"`
}

// attachReplyItemsToGroups 输入回复分组和回复条目列表，输出为组装后的分组列表，目的在于安全地将条目挂载到对应分组。
func attachReplyItemsToGroups(replyGroups []*ReplyGroup, replyItems []*ReplyItem) []*ReplyGroup {
	groupIndex := make(map[string]*ReplyGroup, len(replyGroups))
	for _, replyGroup := range replyGroups {
		replyGroup.Items = make([]*ReplyItem, 0)
		groupIndex[replyGroup.Id] = replyGroup
	}
	for _, replyItem := range replyItems {
		replyGroup, hasGroup := groupIndex[replyItem.GroupId]
		if !hasGroup {
			continue
		}
		replyGroup.Items = append(replyGroup.Items, replyItem)
	}
	return replyGroups
}

// filterReplyGroupsWithItems 输入回复分组列表，输出为包含回复内容的分组列表，目的在于清理空分组搜索结果。
func filterReplyGroupsWithItems(replyGroups []*ReplyGroup) []*ReplyGroup {
	filteredReplyGroups := make([]*ReplyGroup, 0, len(replyGroups))
	for _, replyGroup := range replyGroups {
		if len(replyGroup.Items) == 0 {
			continue
		}
		filteredReplyGroups = append(filteredReplyGroups, replyGroup)
	}
	return filteredReplyGroups
}

// FindReplyItemByUserIdTitle 输入客服标识和标题，输出为单条快捷回复，目的在于兼容原有精确匹配场景。
func FindReplyItemByUserIdTitle(userId interface{}, title string) ReplyItem {
	var reply ReplyItem
	DB.Where("user_id = ? and item_name = ?", userId, title).Find(&reply)
	return reply
}

// FindReplyByUserId 输入客服标识，输出为快捷回复分组列表，目的在于获取完整回复树。
func FindReplyByUserId(userId interface{}) []*ReplyGroup {
	var replyGroups []*ReplyGroup
	var replyItems []*ReplyItem
	DB.Where("user_id = ?", userId).Find(&replyGroups)
	DB.Where("user_id = ?", userId).Find(&replyItems)
	return attachReplyItemsToGroups(replyGroups, replyItems)
}

// FindReplyTitleByUserId 输入客服标识，输出为仅包含标题的快捷回复分组列表，目的在于提供轻量回复索引。
func FindReplyTitleByUserId(userId interface{}) []*ReplyGroup {
	var replyGroups []*ReplyGroup
	var replyItems []*ReplyItem
	DB.Where("user_id = ?", userId).Find(&replyGroups)
	DB.Select("item_name,group_id").Where("user_id = ?", userId).Find(&replyItems)
	return attachReplyItemsToGroups(replyGroups, replyItems)
}

// CreateReplyGroup 输入分组名称和客服标识，输出为数据库创建结果，目的在于新增快捷回复分组。
func CreateReplyGroup(groupName string, userId string) {
	replyGroup := &ReplyGroup{
		GroupName: groupName,
		UserId:    userId,
	}
	DB.Create(replyGroup)
}

// CreateReplyContent 输入分组标识、客服标识、回复内容和关键词，输出为数据库创建结果，目的在于新增快捷回复内容。
func CreateReplyContent(groupId string, userId string, content, itemName string) {
	replyItem := &ReplyItem{
		GroupId:  groupId,
		UserId:   userId,
		Content:  content,
		ItemName: itemName,
	}
	DB.Create(replyItem)
}

// UpdateReplyContent 输入回复标识、客服标识、标题和内容，输出为数据库更新结果，目的在于更新快捷回复内容。
func UpdateReplyContent(id, userId, title, content string) {
	replyItem := &ReplyItem{
		ItemName: title,
		Content:  content,
	}
	DB.Model(&ReplyItem{}).Where("user_id = ? and id = ?", userId, id).Update(replyItem)
}

// DeleteReplyContent 输入回复标识和客服标识，输出为数据库删除结果，目的在于删除单条快捷回复。
func DeleteReplyContent(id string, userId string) {
	DB.Where("user_id = ? and id = ?", userId, id).Delete(ReplyItem{})
}

// DeleteReplyGroup 输入分组标识和客服标识，输出为数据库删除结果，目的在于删除分组及其下属回复。
func DeleteReplyGroup(id string, userId string) {
	DB.Where("user_id = ? and id = ?", userId, id).Delete(ReplyGroup{})
	DB.Where("user_id = ? and group_id = ?", userId, id).Delete(ReplyItem{})
}

// FindReplyBySearch 输入客服标识和搜索词，输出为命中的快捷回复分组列表，目的在于同时支持按关键词和内容搜索。
func FindReplyBySearch(userId interface{}, search string) []*ReplyGroup {
	var replyGroups []*ReplyGroup
	var replyItems []*ReplyItem
	likePattern := "%" + search + "%"
	DB.Where("user_id = ?", userId).Find(&replyGroups)
	DB.Where("user_id = ? AND (content LIKE ? OR item_name LIKE ?)", userId, likePattern, likePattern).Find(&replyItems)
	return filterReplyGroupsWithItems(attachReplyItemsToGroups(replyGroups, replyItems))
}

// FindReplyBySearcch 输入客服标识和搜索词，输出为命中的快捷回复分组列表，目的在于兼容历史调用中的旧函数名。
func FindReplyBySearcch(userId interface{}, search string) []*ReplyGroup {
	return FindReplyBySearch(userId, search)
}
