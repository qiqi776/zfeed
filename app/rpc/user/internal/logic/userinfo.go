package logic

import (
	"strings"

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
		Avatar:   normalizeProfileAsset(userDO.Avatar),
		Bio:      userDO.Bio,
		Gender:   normalizeGender(userDO.Gender),
		Status:   user.UserStatus(userDO.Status),
		Email:    userDO.Email,
	}
	if userDO.Birthday != nil {
		info.Birthday = userDO.Birthday.Unix()
	}

	return info
}

func normalizeGender(value int32) user.Gender {
	switch user.Gender(value) {
	case user.Gender_GENDER_UNKNOWN, user.Gender_GENDER_MALE, user.Gender_GENDER_FEMALE:
		return user.Gender(value)
	default:
		return user.Gender_GENDER_UNKNOWN
	}
}

func normalizeProfileAsset(raw string) string {
	asset := strings.TrimSpace(raw)
	if asset == "" || !isAcceptedProfileAsset(asset) {
		return ""
	}
	return asset
}
