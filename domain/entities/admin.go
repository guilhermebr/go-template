package entities

// Admin Dashboard Stats
type DashboardStats struct {
	TotalUsers     int64 `json:"total_users"`
	AdminUsers     int64 `json:"admin_users"`
	ActiveSessions int64 `json:"active_sessions"`
	SystemAlerts   int64 `json:"system_alerts"`
}

// User List Response
type UserListResponse struct {
	Users      []User `json:"users"`
	Total      int64  `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}
