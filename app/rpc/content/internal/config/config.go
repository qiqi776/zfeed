package config

import (
	"strings"
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	InteractionRpcClientConf zrpc.RpcClientConf
	UserRpcClientConf        zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
	Oss                      OssConfig
	RedisConfig              redis.RedisConf
	KqProducerConf           KqProducerConf
	KqConsumerConf           kq.KqConf
	MySQL                    MySQLConf
	XxlJob                   XxlJobConfig
	Recommend                RecommendConfig
}

type KqProducerConf struct {
	Brokers    []string
	Topic      string
	MaxRetries int
}

func (c KqProducerConf) Enabled() bool {
	if strings.TrimSpace(c.Topic) == "" {
		return false
	}
	if len(c.Brokers) == 0 {
		return false
	}
	for _, broker := range c.Brokers {
		if strings.TrimSpace(broker) != "" {
			return true
		}
	}
	return false
}

type OssConfig struct {
	Provider        string
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	Endpoint        string
	UploadDir       string
	PublicHost      string
}

type MySQLConf struct {
	DataSource string
}

type XxlJobConfig struct {
	AppName          string
	Address          string
	RegistryAddress  string
	IP               string
	Port             int
	AccessToken      string
	AdminAddresses   []string
	RegistryInterval time.Duration
	HTTPTimeout      time.Duration
}

type RecommendConfig struct {
	Enabled          bool
	TimeoutMs        int
	SnapshotTTL      int
	CandidateTTL     int
	Hot              RecommendHotConfig
	NewContent       RecommendNewContentConfig
	Interest         RecommendInterestConfig
	Rank             RecommendRankConfig
	Diversity        RecommendDiversityConfig
	Experiment       RecommendExperimentConfig
	CandidateLimit   int
	FallbackToHot    bool
	ColdStartMetaTTL int
	SeenTTL          int
}

type RecommendHotConfig struct {
	Enabled bool
	Weight  float64
	Limit   int
}

type RecommendNewContentConfig struct {
	Enabled bool
	Weight  float64
	Limit   int
}

type RecommendInterestConfig struct {
	Enabled       bool
	Weight        float64
	Limit         int
	TopTags       int
	MinTags       int
	ProfileTTL    int
	ContentTagTTL int
	TagIndexTTL   int
}

type RecommendRankConfig struct {
	CoarseLimit  int
	AlphaHot     float64
	BetaInterest float64
	GammaFresh   float64
	DeltaQuality float64
	SeenPenalty  float64
}

type RecommendDiversityConfig struct {
	Enabled       bool
	AuthorWindow  int
	MaxSameAuthor int
	TypeWindow    int
	MaxSameType   int
}

type RecommendExperimentConfig struct {
	ID               string
	Enabled          bool
	Salt             string
	TrafficPermyriad int
	TrafficPercent   int
	DefaultVariant   string
	Variants         []RecommendExperimentVariantConfig
}

type RecommendExperimentVariantConfig struct {
	ID               string
	TrafficPermyriad int
	TrafficPercent   int
	Overrides        map[string]string
}
