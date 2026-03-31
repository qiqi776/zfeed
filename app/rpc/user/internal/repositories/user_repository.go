package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/user/internal/do"
	"zfeed/app/rpc/user/internal/model"
)

type UserRepository interface {
	GetByMobile(mobile string) (*do.UserDO, error)
	GetByID(userID int64) (*do.UserDO, error)
	BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error)
	Create(userDO *do.UserDO) (int64, error)
}

type userRepositoryImpl struct {
	ctx context.Context
	db  *gorm.DB
	logx.Logger
}

func NewUserRepository(ctx context.Context, db *gorm.DB) UserRepository {
	return &userRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *userRepositoryImpl) GetByMobile(mobile string) (*do.UserDO, error) {
	var row model.ZfeedUser
	err := r.db.WithContext(r.ctx).
		Where("mobile = ? AND is_deleted = 0", mobile).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return modelToDO(&row), nil
}

func (r *userRepositoryImpl) GetByID(userID int64) (*do.UserDO, error) {
	if userID <= 0 {
		return nil, nil
	}

	var row model.ZfeedUser
	err := r.db.WithContext(r.ctx).
		Where("id = ? AND is_deleted = 0", userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return modelToDO(&row), nil
}

func (r *userRepositoryImpl) BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error) {
	if len(userIDs) == 0 {
		return map[int64]*do.UserDO{}, nil
	}

	var rows []model.ZfeedUser
	if err := r.db.WithContext(r.ctx).
		Where("id IN ? AND is_deleted = 0", userIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[int64]*do.UserDO, len(rows))
	for i := range rows {
		row := rows[i]
		result[row.ID] = modelToDO(&row)
	}

	return result, nil
}

func (r *userRepositoryImpl) Create(userDO *do.UserDO) (int64, error) {
	row := &model.ZfeedUser{
		Username:     userDO.Username,
		Nickname:     userDO.Nickname,
		Avatar:       userDO.Avatar,
		Bio:          userDO.Bio,
		Mobile:       userDO.Mobile,
		Email:        userDO.Email,
		PasswordHash: userDO.PasswordHash,
		PasswordSalt: userDO.PasswordSalt,
		Gender:       userDO.Gender,
		Birthday:     userDO.Birthday,
		Status:       userDO.Status,
		CreatedBy:    userDO.CreatedBy,
		UpdatedBy:    userDO.UpdatedBy,
	}

	if err := r.db.WithContext(r.ctx).Create(row).Error; err != nil {
		return 0, err
	}

	return row.ID, nil
}

func modelToDO(row *model.ZfeedUser) *do.UserDO {
	if row == nil {
		return nil
	}

	return &do.UserDO{
		ID:           row.ID,
		Username:     row.Username,
		Nickname:     row.Nickname,
		Avatar:       row.Avatar,
		Bio:          row.Bio,
		Mobile:       row.Mobile,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		PasswordSalt: row.PasswordSalt,
		Gender:       row.Gender,
		Birthday:     row.Birthday,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy,
		UpdatedBy:    row.UpdatedBy,
	}
}
