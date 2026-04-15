package logic

import (
	"zfeed/app/rpc/user/internal/do"
	"zfeed/app/rpc/user/user"
)

func buildPrivateUserInfo(userDO *do.UserDO) *user.UserInfo {
	if userDO == nil {
		return nil
	}

	info := &user.UserInfo{
		UserId:   userDO.ID,
		Username: userDO.Username,
		Mobile:   userDO.Mobile,
		Nickname: userDO.Nickname,
		Avatar:   userDO.Avatar,
		Bio:      userDO.Bio,
		Gender:   user.Gender(userDO.Gender),
		Status:   user.UserStatus(userDO.Status),
		Email:    userDO.Email,
	}
	if userDO.Birthday != nil {
		info.Birthday = userDO.Birthday.Unix()
	}

	return info
}
