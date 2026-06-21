package commentlogic

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/user/client/userservice"
)

func TestNormalizeCommentPage(t *testing.T) {
	tests := []struct {
		name      string
		pageSize  uint32
		want      uint32
		wantError bool
	}{
		{name: "rejects zero", pageSize: 0, wantError: true},
		{name: "accepts one", pageSize: 1, want: 1},
		{name: "accepts max", pageSize: maxCommentPageSize, want: maxCommentPageSize},
		{name: "rejects over max", pageSize: maxCommentPageSize + 1, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCommentPage(tt.pageSize)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeCommentPage returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeCommentPage() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTrimCommentPage(t *testing.T) {
	comments := []*do.CommentDO{
		{ID: 11},
		{ID: 10},
		{ID: 9},
	}

	tests := []struct {
		name           string
		comments       []*do.CommentDO
		pageSize       uint32
		wantIDs        []int64
		wantNextCursor int64
		wantHasMore    bool
	}{
		{
			name:     "returns all comments when page is not full",
			comments: comments[:2],
			pageSize: 3,
			wantIDs:  []int64{11, 10},
		},
		{
			name:           "trims one lookahead comment",
			comments:       comments,
			pageSize:       2,
			wantIDs:        []int64{11, 10},
			wantNextCursor: 10,
			wantHasMore:    true,
		},
		{
			name:     "keeps empty page stable",
			comments: nil,
			pageSize: 2,
			wantIDs:  []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, nextCursor, hasMore := trimCommentPage(tt.comments, tt.pageSize)
			if ids := commentDOIDs(got); !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Fatalf("trimmed ids = %v, want %v", ids, tt.wantIDs)
			}
			if nextCursor != tt.wantNextCursor {
				t.Fatalf("nextCursor = %d, want %d", nextCursor, tt.wantNextCursor)
			}
			if hasMore != tt.wantHasMore {
				t.Fatalf("hasMore = %v, want %v", hasMore, tt.wantHasMore)
			}
		})
	}
}

func TestCollectCommentUserIDs(t *testing.T) {
	comments := []*do.CommentDO{
		nil,
		{ID: 1, UserID: 0},
		{ID: 2, UserID: 10},
		{ID: 3, UserID: 10},
		{ID: 4, UserID: 11, IsDeleted: 1},
		{ID: 5, UserID: 12, Status: repositories.CommentStatusDeleted},
		{ID: 6, UserID: 13},
	}

	got := collectCommentUserIDs(comments)
	want := []int64{10, 13}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectCommentUserIDs() = %v, want %v", got, want)
	}
}

func TestBuildCommentItems(t *testing.T) {
	createdAt := time.Unix(1770000000, 0)
	comments := []*do.CommentDO{
		nil,
		{
			ID:            1,
			ContentID:     101,
			UserID:        201,
			ReplyToUserID: 301,
			ParentID:      401,
			RootID:        501,
			Comment:       "visible",
			Status:        repositories.CommentStatusNormal,
			ReplyCount:    2,
			CreatedAt:     createdAt,
		},
		{
			ID:        2,
			ContentID: 102,
			UserID:    202,
			Comment:   "hidden",
			Status:    repositories.CommentStatusDeleted,
			CreatedAt: createdAt.Add(time.Second),
		},
	}
	userMap := map[int64]*userservice.UserInfo{
		201: {UserId: 201, Nickname: "alice", Avatar: "alice.png"},
		202: {UserId: 202, Nickname: "deleted-user", Avatar: "deleted.png"},
	}

	got := buildCommentItems(comments, userMap)
	if len(got) != 2 {
		t.Fatalf("items len = %d, want 2", len(got))
	}

	visible := got[0]
	if visible.GetCommentId() != 1 ||
		visible.GetContentId() != 101 ||
		visible.GetUserId() != 201 ||
		visible.GetReplyToUserId() != 301 ||
		visible.GetParentId() != 401 ||
		visible.GetRootId() != 501 ||
		visible.GetComment() != "visible" ||
		visible.GetCreatedAt() != createdAt.Unix() ||
		visible.GetStatus() != repositories.CommentStatusNormal ||
		visible.GetReplyCount() != 2 ||
		visible.GetUserName() != "alice" ||
		visible.GetUserAvatar() != "alice.png" {
		t.Fatalf("visible item = %+v", visible)
	}

	tombstone := got[1]
	if tombstone.GetCommentId() != 2 ||
		tombstone.GetUserId() != 0 ||
		tombstone.GetComment() != "该评论已删除" ||
		tombstone.GetStatus() != repositories.CommentStatusDeleted ||
		tombstone.GetUserName() != "" ||
		tombstone.GetUserAvatar() != "" {
		t.Fatalf("tombstone item = %+v", tombstone)
	}
}

func TestHelperDirtyData(t *testing.T) {
	ctx := context.Background()
	userMap, err := batchLoadCommentUsers(ctx, &fakeCommentUserService{
		users: map[int64]*userservice.UserInfo{
			201: {UserId: 201, Nickname: "valid-user"},
		},
		extraUsers: []*userservice.UserInfo{
			nil,
			{UserId: 0, Nickname: "invalid-user"},
		},
	}, []*do.CommentDO{
		nil,
		{ID: 1, UserID: 201},
	})
	if err != nil {
		t.Fatalf("batchLoadCommentUsers returned error: %v", err)
	}
	if len(userMap) != 1 || userMap[201].GetNickname() != "valid-user" {
		t.Fatalf("user map = %+v, want only valid user 201", userMap)
	}

	if isDeletedComment(nil) {
		t.Fatal("nil comment should not be treated as deleted")
	}
	if items := buildCommentItems([]*do.CommentDO{nil}, nil); len(items) != 0 {
		t.Fatalf("items from nil comment = %+v, want empty", items)
	}

	filtered := filterOrderedCommentItems(
		[]int64{1, 2},
		[]*interaction.CommentItem{nil, {CommentId: 2, Comment: "two"}},
		nil,
	)
	if got := commentItemIDs(filtered); !reflect.DeepEqual(got, []int64{2}) {
		t.Fatalf("filtered ids = %v, want [2]", got)
	}
}

func TestCommentItemCollections(t *testing.T) {
	items := []*interaction.CommentItem{
		nil,
		{CommentId: 2, Comment: "two"},
		{CommentId: 0, Comment: "zero"},
		{CommentId: 1, Comment: "one"},
	}

	itemMap := map[int64]*interaction.CommentItem{}
	mergeCommentItems(items, itemMap)
	if got := sortedCommentMapKeys(itemMap); !reflect.DeepEqual(got, []int64{1, 2}) {
		t.Fatalf("merged keys = %v, want [1 2]", got)
	}

	ordered := filterOrderedCommentItems(
		[]int64{0, 3, 2, 2, 1, 4},
		[]*interaction.CommentItem{
			{CommentId: 1, Comment: "one"},
			{CommentId: 2, Comment: "two"},
			{CommentId: 4, Comment: "four"},
		},
		[]int64{3, 4},
	)
	if got := commentItemIDs(ordered); !reflect.DeepEqual(got, []int64{2, 1}) {
		t.Fatalf("ordered ids = %v, want [2 1]", got)
	}
}

func TestUniqueCommentIDs(t *testing.T) {
	got := uniqueCommentIDs([]int64{0, -1, 3, 2, 3, 1, 2})
	want := []int64{3, 2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueCommentIDs() = %v, want %v", got, want)
	}
}

func TestLoadCommentItemsByIDs(t *testing.T) {
	createdAt := time.Unix(1770000100, 0)
	repo := &fakeCommentRepository{
		comments: map[int64]*do.CommentDO{
			1: {
				ID:        1,
				ContentID: 101,
				UserID:    201,
				Comment:   "one",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: createdAt,
			},
			2: {
				ID:        2,
				ContentID: 102,
				UserID:    202,
				Comment:   "deleted",
				Status:    repositories.CommentStatusDeleted,
				CreatedAt: createdAt.Add(time.Second),
			},
			3: {
				ID:        3,
				ContentID: 103,
				UserID:    203,
				Comment:   "three",
				Status:    repositories.CommentStatusNormal,
				CreatedAt: createdAt.Add(2 * time.Second),
			},
		},
	}
	userRPC := &fakeCommentUserService{
		users: map[int64]*userservice.UserInfo{
			201: {UserId: 201, Nickname: "u201", Avatar: "201.png"},
			203: {UserId: 203, Nickname: "u203", Avatar: "203.png"},
		},
	}

	items, missIDs, err := loadCommentItemsByIDs(context.Background(), repo, userRPC, []int64{0, 3, 2, 3, 99, 1})
	if err != nil {
		t.Fatalf("loadCommentItemsByIDs returned error: %v", err)
	}
	if got := commentItemIDs(items); !reflect.DeepEqual(got, []int64{3, 2, 1}) {
		t.Fatalf("item ids = %v, want [3 2 1]", got)
	}
	if !reflect.DeepEqual(missIDs, []int64{99}) {
		t.Fatalf("missIDs = %v, want [99]", missIDs)
	}
	if items[0].GetUserName() != "u203" || items[2].GetUserName() != "u201" {
		t.Fatalf("user names = [%q %q], want [u203 u201]", items[0].GetUserName(), items[2].GetUserName())
	}
	if items[1].GetUserId() != 0 || items[1].GetComment() != "该评论已删除" {
		t.Fatalf("deleted item = %+v", items[1])
	}
}

func TestLoadCommentItemsByIDsError(t *testing.T) {
	items, missIDs, err := loadCommentItemsByIDs(context.Background(), &fakeCommentRepository{}, nil, []int64{0, -1})
	if err != nil {
		t.Fatalf("empty load returned error: %v", err)
	}
	if len(items) != 0 || len(missIDs) != 0 {
		t.Fatalf("empty load = items:%v miss:%v, want empty", items, missIDs)
	}

	repoErr := errors.New("repository unavailable")
	_, _, err = loadCommentItemsByIDs(context.Background(), &fakeCommentRepository{batchErr: repoErr}, nil, []int64{1})
	if !errors.Is(err, repoErr) {
		t.Fatalf("load error = %v, want %v", err, repoErr)
	}
}

type fakeCommentRepository struct {
	comments          map[int64]*do.CommentDO
	getByID           map[int64]*do.CommentDO
	includeDeleted    map[int64]*do.CommentDO
	listRoot          []*do.CommentDO
	listReplies       []*do.CommentDO
	children          map[int64]bool
	deletedIDs        []int64
	createdID         int64
	createCalls       int
	listRootCalls     int
	listRepliesCalls  int
	getErr            error
	getErrByID        map[int64]error
	includeDeletedErr error
	createErr         error
	batchErr          error
	listRootErr       error
	listRepliesErr    error
	incErr            error
	hasChildrenErr    error
	markErr           error
	decErr            error
	deleteErr         error
}

func (r *fakeCommentRepository) WithTx(*gorm.DB) repositories.CommentRepository {
	return r
}

func (r *fakeCommentRepository) Create(*do.CommentDO) (int64, error) {
	r.createCalls++
	if r.createErr != nil {
		return 0, r.createErr
	}
	if r.createdID > 0 {
		return r.createdID, nil
	}
	return 0, errors.New("unexpected Create call")
}

func (r *fakeCommentRepository) GetByID(commentID int64) (*do.CommentDO, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if r.getErrByID != nil {
		if err := r.getErrByID[commentID]; err != nil {
			return nil, err
		}
	}
	if r.getByID != nil {
		comment := r.getByID[commentID]
		if comment == nil {
			return nil, nil
		}
		cloned := *comment
		return &cloned, nil
	}
	return nil, errors.New("unexpected GetByID call")
}

func (r *fakeCommentRepository) GetByIDIncludeDeleted(commentID int64) (*do.CommentDO, error) {
	if r.includeDeletedErr != nil {
		return nil, r.includeDeletedErr
	}
	if r.includeDeleted != nil {
		comment := r.includeDeleted[commentID]
		if comment == nil {
			return nil, nil
		}
		cloned := *comment
		return &cloned, nil
	}
	return nil, errors.New("unexpected GetByIDIncludeDeleted call")
}

func (r *fakeCommentRepository) ListRootComments(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, errors.New("unexpected ListRootComments call")
}

func (r *fakeCommentRepository) ListRootCommentsIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	r.listRootCalls++
	if r.listRootErr != nil {
		return nil, r.listRootErr
	}
	if r.listRoot != nil {
		return cloneCommentDOs(r.listRoot), nil
	}
	return nil, errors.New("unexpected ListRootCommentsIncludeDeleted call")
}

func (r *fakeCommentRepository) ListReplies(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, errors.New("unexpected ListReplies call")
}

func (r *fakeCommentRepository) ListRepliesIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	r.listRepliesCalls++
	if r.listRepliesErr != nil {
		return nil, r.listRepliesErr
	}
	if r.listReplies != nil {
		return cloneCommentDOs(r.listReplies), nil
	}
	return nil, errors.New("unexpected ListRepliesIncludeDeleted call")
}

func (r *fakeCommentRepository) BatchGetByIDs([]int64) (map[int64]*do.CommentDO, error) {
	return nil, errors.New("unexpected BatchGetByIDs call")
}

func (r *fakeCommentRepository) BatchGetByIDsIncludeDeleted(commentIDs []int64) (map[int64]*do.CommentDO, error) {
	if r.batchErr != nil {
		return nil, r.batchErr
	}
	result := make(map[int64]*do.CommentDO, len(commentIDs))
	for _, commentID := range commentIDs {
		if comment := r.comments[commentID]; comment != nil {
			cloned := *comment
			result[commentID] = &cloned
		}
	}
	return result, nil
}

func (r *fakeCommentRepository) IncReplyCount(int64) error {
	if r.incErr != nil {
		return r.incErr
	}
	return errors.New("unexpected IncReplyCount call")
}

func (r *fakeCommentRepository) DecReplyCount(int64) error {
	if r.decErr != nil {
		return r.decErr
	}
	return nil
}

func (r *fakeCommentRepository) MarkDeleted(int64, int64) error {
	if r.markErr != nil {
		return r.markErr
	}
	return nil
}

func (r *fakeCommentRepository) DeleteByID(commentID int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deletedIDs = append(r.deletedIDs, commentID)
	return nil
}

func (r *fakeCommentRepository) HasChildren(commentID int64) (bool, error) {
	if r.hasChildrenErr != nil {
		return false, r.hasChildrenErr
	}
	if r.children != nil {
		return r.children[commentID], nil
	}
	return false, errors.New("unexpected HasChildren call")
}

func commentDOIDs(comments []*do.CommentDO) []int64 {
	ids := make([]int64, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		ids = append(ids, comment.ID)
	}
	return ids
}

func cloneCommentDOs(comments []*do.CommentDO) []*do.CommentDO {
	cloned := make([]*do.CommentDO, 0, len(comments))
	for _, comment := range comments {
		if comment == nil {
			cloned = append(cloned, nil)
			continue
		}
		item := *comment
		cloned = append(cloned, &item)
	}
	return cloned
}

func commentItemIDs(items []*interaction.CommentItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.GetCommentId())
	}
	return ids
}

func sortedCommentMapKeys(itemMap map[int64]*interaction.CommentItem) []int64 {
	keys := make([]int64, 0, len(itemMap))
	for key := range itemMap {
		keys = append(keys, key)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

func newCommentCacheRedis(t *testing.T) (*miniredis.Miniredis, *gzredis.Redis) {
	t.Helper()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})
	return store, client
}

func newErrorCommentRedis(t *testing.T) *gzredis.Redis {
	t.Helper()

	store, client := newCommentCacheRedis(t)
	store.SetError("ERR redis unavailable")
	return client
}

type fakeContentRepository struct {
	authorID int64
	err      error
}

func (r *fakeContentRepository) GetAuthorID(int64) (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.authorID, nil
}

var _ repositories.CommentRepository = (*fakeCommentRepository)(nil)
var _ repositories.ContentRepository = (*fakeContentRepository)(nil)
var _ userservice.UserService = (*fakeCommentUserService)(nil)
var _ = grpc.EmptyCallOption{}
