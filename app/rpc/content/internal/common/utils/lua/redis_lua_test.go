package lua

import (
	"strings"
	"testing"
)

func TestEmbeddedScriptsAvoidDeprecatedSortedSetRangeCommands(t *testing.T) {
	scripts := map[string]string{
		"QueryHotFeedZSetScript":      QueryHotFeedZSetScript,
		"RebuildHotSnapshotScript":    RebuildHotSnapshotScript,
		"QueryFollowInboxZSetScript":  QueryFollowInboxZSetScript,
		"QueryUserPublishZSetScript":  QueryUserPublishZSetScript,
		"QueryUserFavoriteZSetScript": QueryUserFavoriteZSetScript,
	}

	for name, script := range scripts {
		if strings.Contains(script, "'ZREVRANGEBYSCORE'") {
			t.Fatalf("%s still uses deprecated ZREVRANGEBYSCORE", name)
		}
		if strings.Contains(script, "'ZREVRANGE'") {
			t.Fatalf("%s still uses deprecated ZREVRANGE", name)
		}
	}
}
