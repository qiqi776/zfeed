package consumer

import (
	"testing"

	"github.com/zeromicro/go-queue/kq"

	"zfeed/app/rpc/content/internal/config"
)

func TestRecommendConsumerConfigsIncludesUserActionTopic(t *testing.T) {
	cfg := config.Config{
		KqConsumerConf: kq.KqConf{
			Topic: "zfeed-rec-track",
			Group: "zfeed-content-rec-track-consumer",
		},
		KqUserActionConsumerConf: kq.KqConf{
			Topic: "zfeed-user-action",
			Group: "zfeed-content-user-action-consumer",
		},
	}

	got := recommendConsumerConfigs(cfg)
	if len(got) != 2 {
		t.Fatalf("consumer config count = %d, want 2", len(got))
	}
	if got[0].Topic != "zfeed-rec-track" || got[1].Topic != "zfeed-user-action" {
		t.Fatalf("consumer topics = [%q, %q], want rec-track and user-action", got[0].Topic, got[1].Topic)
	}
}
