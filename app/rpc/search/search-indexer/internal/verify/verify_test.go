package verify

import (
	"context"
	"testing"

	"zfeed/app/rpc/search/internal/common/indexdoc"
)

func TestRunDetectsCountMismatch(t *testing.T) {
	repo := &fakeRepo{contentCount: 2}
	idx := &fakeIndexer{contentCount: 1}

	result, err := Run(context.Background(), repo, idx, Options{
		Entity:       EntityContent,
		ContentIndex: "contents_v2",
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.OK() || result.ContentCountOK {
		t.Fatalf("expected count mismatch, got %+v", result)
	}
}

func TestSwitchAliasAfterVerifySkipsAliasWhenVerificationFails(t *testing.T) {
	repo := &fakeRepo{contentCount: 2}
	idx := &fakeIndexer{contentCount: 1}

	_, err := SwitchAliasAfterVerify(context.Background(), repo, idx, Options{
		Entity:       EntityContent,
		ContentIndex: "contents_v2",
	}, "zfeed_content", "")
	if err == nil {
		t.Fatal("expected verification error")
	}
	if len(idx.switchedAliases) != 0 {
		t.Fatalf("alias switches = %+v, want none", idx.switchedAliases)
	}
}

func TestRunComputesTopNOverlap(t *testing.T) {
	repo := &fakeRepo{
		contentCount: 2,
		mysqlContentIDs: map[string][]int64{
			"growth": {1, 2},
		},
	}
	idx := &fakeIndexer{
		contentCount: 2,
		indexContentIDs: map[string][]int64{
			"growth": {1, 3},
		},
	}

	result, err := Run(context.Background(), repo, idx, Options{
		Entity:       EntityContent,
		ContentIndex: "contents_v2",
		TopQueries:   []string{"growth"},
		TopN:         2,
		MinOverlap:   0.6,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.TopNOK || result.MinOverlapObserved != 0.5 {
		t.Fatalf("unexpected topN result: %+v", result)
	}
}

type fakeRepo struct {
	contentCount    int64
	userCount       int64
	contentSamples  []indexdoc.ContentDocument
	userSamples     []indexdoc.UserDocument
	mysqlContentIDs map[string][]int64
	mysqlUserIDs    map[string][]int64
}

func (r *fakeRepo) CountContentDocuments(context.Context) (int64, error) { return r.contentCount, nil }
func (r *fakeRepo) CountUserDocuments(context.Context) (int64, error)    { return r.userCount, nil }
func (r *fakeRepo) SampleContentDocuments(context.Context, int) ([]indexdoc.ContentDocument, error) {
	return r.contentSamples, nil
}
func (r *fakeRepo) SampleUserDocuments(context.Context, int) ([]indexdoc.UserDocument, error) {
	return r.userSamples, nil
}
func (r *fakeRepo) SearchContentIDs(_ context.Context, query string, _ string, _ int) ([]int64, error) {
	return r.mysqlContentIDs[query], nil
}
func (r *fakeRepo) SearchUserIDs(_ context.Context, query string, _ int) ([]int64, error) {
	return r.mysqlUserIDs[query], nil
}

type fakeIndexer struct {
	contentCount    int64
	userCount       int64
	contentDocs     map[int64]indexdoc.ContentDocument
	userDocs        map[int64]indexdoc.UserDocument
	indexContentIDs map[string][]int64
	indexUserIDs    map[string][]int64
	switchedAliases []string
}

func (i *fakeIndexer) Count(_ context.Context, index string) (int64, error) {
	if index == "users_v2" {
		return i.userCount, nil
	}
	return i.contentCount, nil
}
func (i *fakeIndexer) GetContentDocuments(context.Context, string, []int64) (map[int64]indexdoc.ContentDocument, error) {
	if i.contentDocs == nil {
		return map[int64]indexdoc.ContentDocument{}, nil
	}
	return i.contentDocs, nil
}
func (i *fakeIndexer) GetUserDocuments(context.Context, string, []int64) (map[int64]indexdoc.UserDocument, error) {
	if i.userDocs == nil {
		return map[int64]indexdoc.UserDocument{}, nil
	}
	return i.userDocs, nil
}
func (i *fakeIndexer) SearchContentIDs(_ context.Context, _ string, query string, _ string, _ int) ([]int64, error) {
	return i.indexContentIDs[query], nil
}
func (i *fakeIndexer) SearchUserIDs(_ context.Context, _ string, query string, _ int) ([]int64, error) {
	return i.indexUserIDs[query], nil
}
func (i *fakeIndexer) SwitchAlias(_ context.Context, alias string, targetIndex string) error {
	i.switchedAliases = append(i.switchedAliases, alias+"->"+targetIndex)
	return nil
}
