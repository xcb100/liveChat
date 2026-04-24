package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"goflylivechat/routing"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleManager    = "manager"
	RoleAgent      = "agent"
	RoleAuditor    = "auditor"

	RoleNameContextKey = "role_name"

	PermissionWorkbenchView     = "workbench:view"
	PermissionSessionViewAll    = "session:view_all"
	PermissionSessionViewSelf   = "session:view_self"
	PermissionSessionTake       = "session:take"
	PermissionSessionTransfer   = "session:transfer"
	PermissionSessionClose      = "session:close"
	PermissionConfigView        = "config:view"
	PermissionConfigUpdate      = "config:update"
	PermissionRoutingStatusEdit = "routing_status:update"
	PermissionReplyUpdate       = "reply:update"
	PermissionBlacklistUpdate   = "blacklist:update"
	PermissionUserView          = "user:view"
	PermissionUserManage        = "user:manage"
	PermissionRoleManage        = "role:manage"
	PermissionAuditView         = "audit:view"
)

var rolePermissionMatrix = map[string]map[string]struct{}{
	RoleSuperAdmin: {
		PermissionWorkbenchView:     {},
		PermissionSessionViewAll:    {},
		PermissionSessionViewSelf:   {},
		PermissionSessionTake:       {},
		PermissionSessionTransfer:   {},
		PermissionSessionClose:      {},
		PermissionConfigView:        {},
		PermissionConfigUpdate:      {},
		PermissionRoutingStatusEdit: {},
		PermissionReplyUpdate:       {},
		PermissionBlacklistUpdate:   {},
		PermissionUserView:          {},
		PermissionUserManage:        {},
		PermissionRoleManage:        {},
		PermissionAuditView:         {},
	},
	RoleManager: {
		PermissionWorkbenchView:     {},
		PermissionSessionViewAll:    {},
		PermissionSessionViewSelf:   {},
		PermissionSessionTake:       {},
		PermissionSessionTransfer:   {},
		PermissionSessionClose:      {},
		PermissionConfigView:        {},
		PermissionConfigUpdate:      {},
		PermissionRoutingStatusEdit: {},
		PermissionReplyUpdate:       {},
		PermissionBlacklistUpdate:   {},
		PermissionUserView:          {},
		PermissionAuditView:         {},
	},
	RoleAgent: {
		PermissionWorkbenchView:     {},
		PermissionSessionViewSelf:   {},
		PermissionSessionTake:       {},
		PermissionSessionTransfer:   {},
		PermissionSessionClose:      {},
		PermissionConfigView:        {},
		PermissionConfigUpdate:      {},
		PermissionRoutingStatusEdit: {},
		PermissionReplyUpdate:       {},
		PermissionBlacklistUpdate:   {},
		PermissionUserView:          {},
	},
	RoleAuditor: {
		PermissionWorkbenchView:  {},
		PermissionSessionViewAll: {},
		PermissionConfigView:     {},
		PermissionUserView:       {},
		PermissionAuditView:      {},
	},
}

func NormalizeRoleName(roleName string) string {
	normalizedRoleName := strings.ToLower(strings.TrimSpace(roleName))
	switch normalizedRoleName {
	case RoleSuperAdmin, RoleManager, RoleAgent, RoleAuditor:
		return normalizedRoleName
	default:
		return RoleAgent
	}
}

func HasPermission(roleName string, permission string) bool {
	normalizedRoleName := NormalizeRoleName(roleName)
	rolePermissions, exists := rolePermissionMatrix[normalizedRoleName]
	if !exists {
		return false
	}
	_, hasPermission := rolePermissions[permission]
	return hasPermission
}

func HasPermissionFromContext(c *gin.Context, permission string) bool {
	roleNameValue, exists := c.Get(RoleNameContextKey)
	if !exists {
		return false
	}
	roleName, _ := roleNameValue.(string)
	return HasPermission(roleName, permission)
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasPermissionFromContext(c, permission) {
			c.Next()
			return
		}
		c.JSON(200, gin.H{
			"code": 403,
			"msg":  "没有权限:" + permission,
		})
		c.Abort()
	}
}

func CanAccessSession(c *gin.Context, ownerID string, routeStatus string) bool {
	if HasPermissionFromContext(c, PermissionSessionViewAll) {
		return true
	}
	currentKefuNameValue, exists := c.Get("kefu_name")
	if !exists {
		return false
	}
	currentKefuName, _ := currentKefuNameValue.(string)
	if ownerID != "" && ownerID == currentKefuName {
		return true
	}
	return routeStatus == routing.RouteStatusPending && HasPermissionFromContext(c, PermissionSessionTake)
}
