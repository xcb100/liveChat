package models

import (
	"strconv"
	"strings"
)

var CustomConfigs []Config

type Config struct {
	ID        uint   `gorm:"primary_key" json:"id"`
	ConfName  string `json:"conf_name"`
	ConfKey   string `json:"conf_key"`
	ConfValue string `json:"conf_value"`
	UserId    string `json:"user_id"`
}

func UpdateConfig(userid interface{}, key string, value string) {
	config := FindConfigByUserId(userid, key)
	if config.ID != 0 {
		config.ConfValue = value
		DB.Model(&Config{}).Where("user_id = ? and conf_key = ?", userid, key).Update(config)
	} else {
		newConfig := &Config{
			ID:        0,
			ConfName:  "",
			ConfKey:   key,
			ConfValue: value,
			UserId:    userid.(string),
		}
		DB.Create(newConfig)
	}

}
func FindConfigs() []Config {
	var config []Config
	if !IsDatabaseReachable() {
		return config
	}
	DB.Find(&config)
	return config
}
func FindConfigsByUserId(userid interface{}) []Config {
	var config []Config
	if !IsDatabaseReachable() {
		return config
	}
	DB.Where("user_id = ?", userid).Find(&config)
	return config
}

func FindConfig(key string) string {
	for _, config := range CustomConfigs {
		if key == config.ConfKey {
			return config.ConfValue
		}
	}
	return ""
}
func FindConfigByUserId(userId interface{}, key string) Config {
	var config Config
	if !IsDatabaseReachable() {
		return config
	}
	DB.Where("user_id = ? and conf_key = ?", userId, key).Find(&config)
	return config
}

func GetUserRoutingSkills(userID string) []string {
	config := FindConfigByUserId(userID, "RoutingSkills")
	return NormalizeSkillList(config.ConfValue)
}

func GetUserKefuPresenceStatus(userID string) string {
	config := FindConfigByUserId(userID, "KefuPresenceStatus")
	normalizedValue := strings.ToLower(strings.TrimSpace(config.ConfValue))
	switch normalizedValue {
	case "online", "away", "busy":
		return normalizedValue
	default:
		return "online"
	}
}

func GetUserKefuAcceptingSessions(userID string) bool {
	config := FindConfigByUserId(userID, "KefuAcceptingSessions")
	if strings.TrimSpace(config.ConfValue) == "" {
		return true
	}
	parsedValue, parseError := strconv.ParseBool(strings.TrimSpace(config.ConfValue))
	if parseError != nil {
		return true
	}
	return parsedValue
}

func NormalizeSkillList(rawValue string) []string {
	parts := strings.Split(rawValue, ",")
	skills := make([]string, 0, len(parts))
	seenSkills := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		normalizedSkill := strings.ToLower(strings.TrimSpace(part))
		if normalizedSkill == "" {
			continue
		}
		if _, exists := seenSkills[normalizedSkill]; exists {
			continue
		}
		seenSkills[normalizedSkill] = struct{}{}
		skills = append(skills, normalizedSkill)
	}
	return skills
}
