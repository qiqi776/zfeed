package handler

import (
	"net/http"

	"zfeed/app/admin-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers registers all admin API routes.
func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// Public admin login endpoint (no auth required)
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: AdminLoginHandler(serverCtx),
			},
		},
		rest.WithPrefix("/v1/admin"),
	)

	// Protected admin endpoints (require admin authentication)
	adminRoutes := []rest.Route{
		{
			Method:  http.MethodPost,
			Path:    "/dashboard",
			Handler: AdminDashboardHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/users/list",
			Handler: AdminListUsersHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/users/status",
			Handler: AdminUpdateUserStatusHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/contents/list",
			Handler: AdminListContentsHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/contents/status",
			Handler: AdminUpdateContentStatusHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/contents/batch-status",
			Handler: AdminBatchContentStatusHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/comments/list",
			Handler: AdminListCommentsHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/comments/status",
			Handler: AdminUpdateCommentStatusHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/settings",
			Handler: AdminGetSettingsHandler(serverCtx),
		},
		{
			Method:  http.MethodPost,
			Path:    "/settings/update",
			Handler: AdminUpdateSettingsHandler(serverCtx),
		},
	}
	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.AdminAuthMiddleware},
			adminRoutes...,
		),
		rest.WithPrefix("/v1/admin"),
	)
}
