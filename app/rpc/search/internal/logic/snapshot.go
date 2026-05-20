package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"zfeed/app/rpc/search/internal/querynorm"
	"zfeed/app/rpc/search/internal/repositories"
	"zfeed/app/rpc/search/internal/svc"
	"zfeed/pkg/errorx"
)

const searchSnapshotVersion = 1

type searchSnapshot struct {
	Version     int                             `json:"version"`
	Entity      string                          `json:"entity"`
	Mode        string                          `json:"mode"`
	QueryHash   string                          `json:"query_hash"`
	CreatedAt   int64                           `json:"created_at"`
	UserRows    []repositories.SearchUserRow    `json:"user_rows,omitempty"`
	ContentRows []repositories.SearchContentRow `json:"content_rows,omitempty"`
}

func createUserSnapshot(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	mode string,
	query querynorm.Query,
	rows []repositories.SearchUserRow,
) (string, error) {
	return createSearchSnapshot(ctx, svcCtx, searchSnapshot{
		Version:   searchSnapshotVersion,
		Entity:    searchEntityUsers,
		Mode:      mode,
		QueryHash: query.Hash,
		CreatedAt: time.Now().Unix(),
		UserRows:  rows,
	})
}

func createContentSnapshot(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	mode string,
	query querynorm.Query,
	rows []repositories.SearchContentRow,
) (string, error) {
	return createSearchSnapshot(ctx, svcCtx, searchSnapshot{
		Version:     searchSnapshotVersion,
		Entity:      searchEntityContents,
		Mode:        mode,
		QueryHash:   query.Hash,
		CreatedAt:   time.Now().Unix(),
		ContentRows: rows,
	})
}

func createSearchSnapshot(ctx context.Context, svcCtx *svc.ServiceContext, snapshot searchSnapshot) (string, error) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return "", errorx.NewMsg("搜索快照不可用")
	}

	snapshotID := uuid.NewString()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}

	if err := svcCtx.Redis.SetexCtx(ctx, searchSnapshotKey(snapshotID), string(data), snapshotTTLSeconds(svcCtx)); err != nil {
		return "", err
	}

	return snapshotID, nil
}

func loadSearchSnapshot(ctx context.Context, svcCtx *svc.ServiceContext, snapshotID string, entity string, mode string, query querynorm.Query) (searchSnapshot, error) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return searchSnapshot{}, errorx.NewMsg("搜索快照不可用")
	}

	raw, err := svcCtx.Redis.GetCtx(ctx, searchSnapshotKey(snapshotID))
	if err != nil {
		return searchSnapshot{}, err
	}
	if raw == "" {
		return searchSnapshot{}, errorx.NewBadRequest("搜索快照已过期，请重新搜索")
	}

	var snapshot searchSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return searchSnapshot{}, errorx.NewBadRequest("搜索快照无效，请重新搜索")
	}
	if snapshot.Version != searchSnapshotVersion ||
		snapshot.Entity != entity ||
		snapshot.Mode != mode ||
		snapshot.QueryHash != query.Hash {
		return searchSnapshot{}, errorx.NewBadRequest("分页令牌与当前查询不匹配")
	}

	return snapshot, nil
}

func searchSnapshotKey(snapshotID string) string {
	return fmt.Sprintf("search:snapshot:v1:%s", snapshotID)
}
