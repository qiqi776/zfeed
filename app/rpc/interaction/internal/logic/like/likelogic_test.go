package likelogic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
)

type stubLikeRepo struct {
	isLikedFunc func(userID int64, scene int32, contentID int64) (bool, error)
}

func (r *stubLikeRepo) Upsert(*do.LikeDO) error {
	return nil
}

func (r *stubLikeRepo) CountByTarget(int32, int64) (int64, error) {
	return 0, nil
}

func (r *stubLikeRepo) CountByTargets([]repositories.LikeTarget) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (r *stubLikeRepo) IsLiked(userID int64, scene int32, contentID int64) (bool, error) {
	if r.isLikedFunc == nil {
		return false, nil
	}

	return r.isLikedFunc(userID, scene, contentID)
}

func (r *stubLikeRepo) BatchIsLiked(int64, []repositories.LikeTarget) (map[string]bool, error) {
	return map[string]bool{}, nil
}

type stubContentRepo struct {
	getAuthorIDFunc func(contentID int64) (int64, error)
}

func (r *stubContentRepo) GetAuthorID(contentID int64) (int64, error) {
	if r.getAuthorIDFunc == nil {
		return 0, nil
	}

	return r.getAuthorIDFunc(contentID)
}

type stubCommentRepo struct{}

func (r *stubCommentRepo) WithTx(tx *gorm.DB) repositories.CommentRepository {
	return r
}

func (r *stubCommentRepo) Create(*do.CommentDO) (int64, error) { return 0, nil }

func (r *stubCommentRepo) GetByID(int64) (*do.CommentDO, error) { return nil, nil }

func (r *stubCommentRepo) GetByIDIncludeDeleted(int64) (*do.CommentDO, error) { return nil, nil }

func (r *stubCommentRepo) ListRootComments(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}

func (r *stubCommentRepo) ListRootCommentsIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}

func (r *stubCommentRepo) ListReplies(int64, int64, uint32) ([]*do.CommentDO, error) { return nil, nil }

func (r *stubCommentRepo) ListRepliesIncludeDeleted(int64, int64, uint32) ([]*do.CommentDO, error) {
	return nil, nil
}

func (r *stubCommentRepo) BatchGetByIDs([]int64) (map[int64]*do.CommentDO, error) { return nil, nil }

func (r *stubCommentRepo) BatchGetByIDsIncludeDeleted([]int64) (map[int64]*do.CommentDO, error) {
	return nil, nil
}

func (r *stubCommentRepo) MarkDeleted(int64, int64) error { return nil }

func (r *stubCommentRepo) DeleteByID(int64) error { return nil }

func (r *stubCommentRepo) HasChildren(int64) (bool, error) { return false, nil }

func (r *stubCommentRepo) IncReplyCount(int64) error { return nil }

func (r *stubCommentRepo) DecReplyCount(int64) error { return nil }

type likeEventCall struct {
	userID        int64
	contentID     int64
	contentUserID int64
	scene         string
}

type stubLikeProducer struct {
	likeCalls     chan likeEventCall
	sendLikeErr   error
	sendCancelErr error
}

func (p *stubLikeProducer) SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) error {
	if p.sendLikeErr != nil {
		return p.sendLikeErr
	}
	if p.likeCalls == nil {
		return nil
	}

	p.likeCalls <- likeEventCall{
		userID:        userID,
		contentID:     contentID,
		contentUserID: contentUserID,
		scene:         scene,
	}
	return nil
}

func (p *stubLikeProducer) SendCancelLikeEvent(context.Context, int64, int64, int64, string) error {
	return p.sendCancelErr
}

func TestLikeThenUnlikeStillWorksAfterManyWrites(t *testing.T) {
	t.Parallel()

	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	svcCtx := &svc.ServiceContext{
		Redis: client,
	}

	likeLogic := NewLikeLogic(context.Background(), svcCtx)
	unlikeLogic := NewUnlikeLogic(context.Background(), svcCtx)

	const (
		userID        int64 = 1001
		firstContent  int64 = 1
		totalContents int64 = 10050
	)

	for contentID := int64(1); contentID <= totalContents; contentID++ {
		changed, err := likeLogic.processLike(userID, interaction.Scene_ARTICLE, contentID)
		if err != nil {
			t.Fatalf("processLike(%d) returned error: %v", contentID, err)
		}
		if !changed {
			t.Fatalf("processLike(%d) changed=false, want true", contentID)
		}
	}

	changed, err := unlikeLogic.processUnlike(userID, interaction.Scene_ARTICLE, firstContent)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if !changed {
		t.Fatalf("processUnlike(%d) changed=false, want true", firstContent)
	}

	changed, err = likeLogic.processLike(userID, interaction.Scene_ARTICLE, firstContent)
	if err != nil {
		t.Fatalf("processLike after unlike returned error: %v", err)
	}
	if !changed {
		t.Fatalf("processLike(%d) after unlike changed=false, want true", firstContent)
	}
}

func TestProcessUnlikeFallsBackToDBWhenCacheExpired(t *testing.T) {
	t.Parallel()

	db := newLikeLogicTestDB(t)
	client := newLikeLogicTestRedis(t)

	if err := db.Create(&likeTestRow{
		UserID:        1001,
		Scene:         int32(interaction.Scene_ARTICLE),
		ContentID:     9001,
		ContentUserID: 2001,
		Status:        10,
		IsDeleted:     0,
	}).Error; err != nil {
		t.Fatalf("seed like row: %v", err)
	}

	logic := NewUnlikeLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	changed, err := logic.processUnlike(1001, interaction.Scene_ARTICLE, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if !changed {
		t.Fatal("processUnlike changed=false, want true when DB still shows liked")
	}
}

func TestProcessUnlikeReturnsFalseWhenCacheExpiredAndDBNotLiked(t *testing.T) {
	t.Parallel()

	db := newLikeLogicTestDB(t)
	client := newLikeLogicTestRedis(t)

	logic := NewUnlikeLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   client,
	})

	changed, err := logic.processUnlike(1001, interaction.Scene_ARTICLE, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if changed {
		t.Fatal("processUnlike changed=true, want false when DB has no liked record")
	}
}

func TestProcessUnlikeSkipsDBWhenCacheShowsAlreadyUnliked(t *testing.T) {
	t.Parallel()

	client := newLikeLogicTestRedis(t)
	if err := cacheLikeState(context.Background(), client, likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9001), false); err != nil {
		t.Fatalf("seed unlike cache: %v", err)
	}

	logic := &UnlikeLogic{
		ctx:    context.Background(),
		svcCtx: &svc.ServiceContext{Redis: client},
		Logger: logx.WithContext(context.Background()),
		likeRepo: &stubLikeRepo{
			isLikedFunc: func(userID int64, scene int32, contentID int64) (bool, error) {
				t.Fatalf("unexpected DB fallback, user_id=%d, scene=%d, content_id=%d", userID, scene, contentID)
				return false, nil
			},
		},
	}

	changed, err := logic.processUnlike(1001, interaction.Scene_ARTICLE, 9001)
	if err != nil {
		t.Fatalf("processUnlike returned error: %v", err)
	}
	if changed {
		t.Fatal("processUnlike changed=true, want false when cache already marks unlike")
	}
}

func TestLikeReturnsNotFoundWhenContentDoesNotExist(t *testing.T) {
	t.Parallel()

	logic := &LikeLogic{
		ctx:    context.Background(),
		svcCtx: &svc.ServiceContext{},
		Logger: logx.WithContext(context.Background()),
		contentRepo: &stubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) {
				if contentID != 9001 {
					t.Fatalf("unexpected content_id=%d", contentID)
				}
				return 0, nil
			},
		},
		commentRepo: &stubCommentRepo{},
	}

	_, err := logic.Like(&interaction.LikeReq{
		UserId:        1001,
		ContentId:     9001,
		ContentUserId: 9999,
		Scene:         interaction.Scene_ARTICLE,
	})
	if err == nil {
		t.Fatal("Like returned nil error, want content not found")
	}
	if !strings.Contains(err.Error(), "内容不存在") {
		t.Fatalf("Like error = %v, want contains 内容不存在", err)
	}
}

func TestLikePublishesResolvedContentAuthorInsteadOfClientValue(t *testing.T) {
	t.Parallel()

	redisClient := newLikeLogicTestRedis(t)
	producer := &stubLikeProducer{likeCalls: make(chan likeEventCall, 1)}

	logic := &LikeLogic{
		ctx: context.Background(),
		svcCtx: &svc.ServiceContext{
			Redis:        redisClient,
			LikeProducer: producer,
		},
		Logger: logx.WithContext(context.Background()),
		contentRepo: &stubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) {
				if contentID != 9001 {
					t.Fatalf("unexpected content_id=%d", contentID)
				}
				return 2001, nil
			},
		},
		commentRepo: &stubCommentRepo{},
	}

	_, err := logic.Like(&interaction.LikeReq{
		UserId:        1001,
		ContentId:     9001,
		ContentUserId: 9999,
		Scene:         interaction.Scene_ARTICLE,
	})
	if err != nil {
		t.Fatalf("Like returned error: %v", err)
	}

	select {
	case call := <-producer.likeCalls:
		if call.userID != 1001 || call.contentID != 9001 {
			t.Fatalf("unexpected like event payload: %+v", call)
		}
		if call.contentUserID != 2001 {
			t.Fatalf("like event content_user_id = %d, want resolved author 2001", call.contentUserID)
		}
		if call.scene != interaction.Scene_ARTICLE.String() {
			t.Fatalf("like event scene = %s, want %s", call.scene, interaction.Scene_ARTICLE.String())
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive like event")
	}
}

func TestLikeRollsBackCacheWhenEventPersistenceFails(t *testing.T) {
	t.Parallel()

	redisClient := newLikeLogicTestRedis(t)

	logic := &LikeLogic{
		ctx: context.Background(),
		svcCtx: &svc.ServiceContext{
			Redis:        redisClient,
			LikeProducer: &stubLikeProducer{sendLikeErr: errors.New("outbox unavailable")},
		},
		Logger: logx.WithContext(context.Background()),
		contentRepo: &stubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) {
				return 2001, nil
			},
		},
		commentRepo: &stubCommentRepo{},
	}

	_, err := logic.Like(&interaction.LikeReq{
		UserId:    1001,
		ContentId: 9001,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err == nil {
		t.Fatal("Like returned nil error, want persistence failure")
	}

	values, err := redisClient.HmgetCtx(context.Background(), likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9001))
	if err != nil {
		t.Fatalf("read like cache: %v", err)
	}
	if len(values) != 1 || values[0] != likeCacheValueUnliked {
		t.Fatalf("cache value after rollback = %v, want [%s]", values, likeCacheValueUnliked)
	}
}

func TestUnlikeRollsBackCacheWhenEventPersistenceFails(t *testing.T) {
	t.Parallel()

	redisClient := newLikeLogicTestRedis(t)
	if err := cacheLikeState(context.Background(), redisClient, likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9001), true); err != nil {
		t.Fatalf("seed like cache: %v", err)
	}

	logic := &UnlikeLogic{
		ctx: context.Background(),
		svcCtx: &svc.ServiceContext{
			Redis:        redisClient,
			LikeProducer: &stubLikeProducer{sendCancelErr: errors.New("outbox unavailable")},
		},
		Logger: logx.WithContext(context.Background()),
		contentRepo: &stubContentRepo{
			getAuthorIDFunc: func(contentID int64) (int64, error) {
				return 2001, nil
			},
		},
		commentRepo: &stubCommentRepo{},
		likeRepo:    &stubLikeRepo{},
	}

	_, err := logic.Unlike(&interaction.UnlikeReq{
		UserId:    1001,
		ContentId: 9001,
		Scene:     interaction.Scene_ARTICLE,
	})
	if err == nil {
		t.Fatal("Unlike returned nil error, want persistence failure")
	}

	values, err := redisClient.HmgetCtx(context.Background(), likeCacheKey(1001), likeTargetKey(interaction.Scene_ARTICLE, 9001))
	if err != nil {
		t.Fatalf("read like cache: %v", err)
	}
	if len(values) != 1 || values[0] != likeCacheValueLiked {
		t.Fatalf("cache value after rollback = %v, want [%s]", values, likeCacheValueLiked)
	}
}

var _ repositories.ContentRepository = (*stubContentRepo)(nil)
var _ repositories.CommentRepository = (*stubCommentRepo)(nil)
