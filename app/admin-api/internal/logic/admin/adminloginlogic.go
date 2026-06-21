package admin

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	"zfeed/app/rpc/user/user"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Login Logic ====================

type AdminLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminLoginLogic {
	return &AdminLoginLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminLoginReq struct {
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
}

type AdminLoginRes struct {
	Token     string `json:"token"`
	ExpiredAt int64  `json:"expired_at"`
	UserID    int64  `json:"user_id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Role      int32  `json:"role"`
}

func (l *AdminLoginLogic) AdminLogin(req *AdminLoginReq) (*AdminLoginRes, error) {
	if req.Mobile == "" || req.Password == "" {
		return nil, errorx.NewBadRequest("参数错误")
	}

	// Reuse existing user-rpc Login
	loginResp, err := l.svcCtx.UserRpc.Login(l.ctx, &user.LoginReq{
		Mobile:   req.Mobile,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}

	userID := loginResp.GetUserId()

	// Fetch role from DB since user proto doesn't have role field yet
	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	var role int32
	if err := db.QueryRow("SELECT role FROM zfeed_user WHERE id = ?", userID).Scan(&role); err != nil {
		role = 0
	}

	if role < 1 {
		return nil, errorx.NewForbidden("无管理员权限")
	}

	// Write role to Redis cache for AdminAuthMiddleware
	userRoleKey := "user:role:" + strconv.FormatInt(userID, 10)
	roleTTL := int(l.svcCtx.Config.SessionTTL)
	if roleTTL <= 0 {
		roleTTL = 7 * 24 * 60 * 60 // default 7 days
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, userRoleKey, strconv.Itoa(int(role)), roleTTL); err != nil {
		l.Logger.Errorf("cache admin role failed: userID=%d, role=%d, err=%v", userID, role, err)
		return nil, errorx.NewInternal("登录状态保存失败，请稍后重试")
	}

	return &AdminLoginRes{
		Token:     loginResp.GetToken(),
		ExpiredAt: loginResp.GetExpiredAt(),
		UserID:    userID,
		Nickname:  loginResp.GetNickname(),
		Avatar:    loginResp.GetAvatar(),
		Role:      role,
	}, nil
}

// ==================== Helper: extract admin user id ====================

func getAdminUserID(ctx context.Context) (int64, error) {
	uid, ok := ctx.Value("admin_user_id").(int64)
	if !ok || uid <= 0 {
		return 0, errorx.NewUnauthorized("管理员未登录")
	}
	return uid, nil
}
