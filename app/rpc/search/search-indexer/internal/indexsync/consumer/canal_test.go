package consumer

import (
	"context"
	"encoding/json"
	"testing"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

func TestCanalSearchConsumerReindexesArticleContent(t *testing.T) {
	repo := &fakeRepo{
		contentDocs: map[int64]*indexdoc.ContentDocument{
			4001: {ContentID: 4001, Title: "Growth"},
		},
	}
	idx := &fakeIndexer{}
	consumer := newCanalSearchConsumerForTest(context.Background(), repo, idx)

	msg := canalMessage{
		ID:    1,
		Table: tableArticle,
		Type:  "UPDATE",
		Data:  []map[string]any{{"content_id": float64(4001)}},
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}
	if len(idx.indexedContents) != 1 || idx.indexedContents[0] != 4001 {
		t.Fatalf("indexed contents = %v, want [4001]", idx.indexedContents)
	}
}

func TestCanalSearchConsumerReindexesUserAndAuthorContents(t *testing.T) {
	repo := &fakeRepo{
		userDocs: map[int64]*indexdoc.UserDocument{
			3001: {UserID: 3001, Nickname: "writer"},
		},
		authorContents: map[int64][]int64{
			3001: {4001, 4002},
		},
		contentDocs: map[int64]*indexdoc.ContentDocument{
			4001: {ContentID: 4001, AuthorID: 3001},
			4002: {ContentID: 4002, AuthorID: 3001},
		},
	}
	idx := &fakeIndexer{}
	consumer := newCanalSearchConsumerForTest(context.Background(), repo, idx)

	msg := canalMessage{
		ID:    2,
		Table: tableUser,
		Type:  "UPDATE",
		Data:  []map[string]any{{"id": "3001"}},
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	if err := consumer.Consume(context.Background(), "", string(raw)); err != nil {
		t.Fatalf("Consume returned error: %v", err)
	}
	if len(idx.indexedUsers) != 1 || idx.indexedUsers[0] != 3001 {
		t.Fatalf("indexed users = %v, want [3001]", idx.indexedUsers)
	}
	if len(idx.indexedContents) != 2 || idx.indexedContents[0] != 4001 || idx.indexedContents[1] != 4002 {
		t.Fatalf("indexed contents = %v, want [4001 4002]", idx.indexedContents)
	}
}

type fakeRepo struct {
	contentDocs    map[int64]*indexdoc.ContentDocument
	userDocs       map[int64]*indexdoc.UserDocument
	authorContents map[int64][]int64
}

func (r *fakeRepo) BuildContentDocument(_ context.Context, contentID int64) (*indexdoc.ContentDocument, bool, error) {
	doc, ok := r.contentDocs[contentID]
	return doc, ok, nil
}

func (r *fakeRepo) BuildUserDocument(_ context.Context, userID int64) (*indexdoc.UserDocument, bool, error) {
	doc, ok := r.userDocs[userID]
	return doc, ok, nil
}

func (r *fakeRepo) ListContentIDsByAuthor(_ context.Context, authorID int64, _ int) ([]int64, error) {
	return r.authorContents[authorID], nil
}

type fakeIndexer struct {
	indexedContents []int64
	deletedContents []int64
	indexedUsers    []int64
	deletedUsers    []int64
}

func (i *fakeIndexer) IndexContent(_ context.Context, doc indexdoc.ContentDocument) error {
	i.indexedContents = append(i.indexedContents, doc.ContentID)
	return nil
}

func (i *fakeIndexer) DeleteContent(_ context.Context, contentID int64) error {
	i.deletedContents = append(i.deletedContents, contentID)
	return nil
}

func (i *fakeIndexer) IndexUser(_ context.Context, doc indexdoc.UserDocument) error {
	i.indexedUsers = append(i.indexedUsers, doc.UserID)
	return nil
}

func (i *fakeIndexer) DeleteUser(_ context.Context, userID int64) error {
	i.deletedUsers = append(i.deletedUsers, userID)
	return nil
}
