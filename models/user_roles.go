package models

import (
	"strconv"
)

type User_role struct {
	ID     uint   `gorm:"primary_key" json:"id"`
	UserId string `json:"user_id"`
	RoleId uint   `json:"role_id"`
}

func FindRoleByUserId(userId interface{}) User_role {
	var uRole User_role
	DB.Where("user_id = ?", userId).First(&uRole)
	return uRole
}

func CreateUserRole(userId uint, roleId uint) {
	uRole := &User_role{
		UserId: strconv.Itoa(int(userId)),
		RoleId: roleId,
	}
	DB.Create(uRole)
}

func ReplaceUserRole(userId uint, roleId uint) {
	DeleteRoleByUserId(strconv.Itoa(int(userId)))
	CreateUserRole(userId, roleId)
}

func AssignRoleToUser(userId uint, roleName string) bool {
	role := FindRoleByName(roleName)
	if role.ID == 0 {
		return false
	}
	ReplaceUserRole(userId, role.ID)
	return true
}

func FindRoleNameByUserId(userId uint) string {
	userRole := FindRoleByUserId(strconv.Itoa(int(userId)))
	if userRole.ID == 0 || userRole.RoleId == 0 {
		return ""
	}
	return FindRole(userRole.RoleId).Name
}

func DeleteRoleByUserId(userId interface{}) {
	DB.Where("user_id = ?", userId).Delete(User_role{})
}
