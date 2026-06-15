package recommend

import contentconfig "zfeed/app/rpc/content/internal/config"

const (
	defaultCandidateLimit   = 500
	defaultTimeoutMs        = 120
	defaultSnapshotTTL      = 10 * 60
	defaultCandidateTTL     = 5 * 60
	defaultHotLimit         = 300
	defaultHotWeight        = 0.55
	defaultNewContentLimit  = 100
	defaultNewContentWeight = 0.2
	defaultInterestLimit    = 200
	defaultInterestWeight   = 0.25
	defaultInterestTopTags  = 8
	defaultInterestMinTags  = 3
	defaultProfileTTL       = 30 * 24 * 3600
	defaultContentTagTTL    = 30 * 24 * 3600
	defaultTagIndexTTL      = 7 * 24 * 3600
	defaultCoarseLimit      = 200
	defaultAlphaHot         = 0.45
	defaultBetaInterest     = 0.30
	defaultGammaFresh       = 0.20
	defaultDeltaQuality     = 0.05
	defaultSeenPenalty      = 0.30
	defaultRepeatedSeen     = 2
	defaultColdMetaTTL      = 7 * 24 * 3600
	defaultSeenTTL          = 7 * 24 * 3600
	defaultAuthorWindow     = 5
	defaultMaxSameAuthor    = 1
	defaultTypeWindow       = 6
	defaultMaxSameType      = 4
	defaultNewContentTopN   = 20
	defaultNewContentMin    = 2
)

func NormalizeConfig(cfg contentconfig.RecommendConfig) contentconfig.RecommendConfig {
	if cfg.CandidateLimit <= 0 {
		cfg.CandidateLimit = defaultCandidateLimit
	}
	if cfg.TimeoutMs <= 0 {
		cfg.TimeoutMs = defaultTimeoutMs
	}
	if cfg.SnapshotTTL <= 0 {
		cfg.SnapshotTTL = defaultSnapshotTTL
	}
	if cfg.CandidateTTL <= 0 {
		cfg.CandidateTTL = defaultCandidateTTL
	}
	if cfg.ColdStartMetaTTL <= 0 {
		cfg.ColdStartMetaTTL = defaultColdMetaTTL
	}
	if cfg.SeenTTL <= 0 {
		cfg.SeenTTL = defaultSeenTTL
	}
	if !cfg.Hot.Enabled && cfg.Hot.Weight == 0 && cfg.Hot.Limit == 0 {
		cfg.Hot.Enabled = true
	}
	if cfg.Hot.Weight <= 0 {
		cfg.Hot.Weight = defaultHotWeight
	}
	if cfg.Hot.Limit <= 0 {
		cfg.Hot.Limit = defaultHotLimit
	}
	if cfg.NewContent.Limit <= 0 {
		cfg.NewContent.Limit = defaultNewContentLimit
	}
	if cfg.NewContent.Weight <= 0 {
		cfg.NewContent.Weight = defaultNewContentWeight
	}
	if cfg.Interest.Limit <= 0 {
		cfg.Interest.Limit = defaultInterestLimit
	}
	if cfg.Interest.Weight <= 0 {
		cfg.Interest.Weight = defaultInterestWeight
	}
	if cfg.Interest.TopTags <= 0 {
		cfg.Interest.TopTags = defaultInterestTopTags
	}
	if cfg.Interest.MinTags <= 0 {
		cfg.Interest.MinTags = defaultInterestMinTags
	}
	if cfg.Interest.ProfileTTL <= 0 {
		cfg.Interest.ProfileTTL = defaultProfileTTL
	}
	if cfg.Interest.ContentTagTTL <= 0 {
		cfg.Interest.ContentTagTTL = defaultContentTagTTL
	}
	if cfg.Interest.TagIndexTTL <= 0 {
		cfg.Interest.TagIndexTTL = defaultTagIndexTTL
	}
	if cfg.Rank.CoarseLimit <= 0 {
		cfg.Rank.CoarseLimit = defaultCoarseLimit
	}
	if cfg.Rank.AlphaHot <= 0 {
		cfg.Rank.AlphaHot = defaultAlphaHot
	}
	if cfg.Rank.BetaInterest <= 0 {
		cfg.Rank.BetaInterest = defaultBetaInterest
	}
	if cfg.Rank.GammaFresh <= 0 {
		cfg.Rank.GammaFresh = defaultGammaFresh
	}
	if cfg.Rank.DeltaQuality <= 0 {
		cfg.Rank.DeltaQuality = defaultDeltaQuality
	}
	if cfg.Rank.SeenPenalty <= 0 {
		cfg.Rank.SeenPenalty = defaultSeenPenalty
	}
	if cfg.Rank.RepeatedSeenFilterN <= 0 {
		cfg.Rank.RepeatedSeenFilterN = defaultRepeatedSeen
	}
	if cfg.Diversity.AuthorWindow <= 0 {
		cfg.Diversity.AuthorWindow = defaultAuthorWindow
	}
	if cfg.Diversity.MaxSameAuthor <= 0 {
		cfg.Diversity.MaxSameAuthor = defaultMaxSameAuthor
	}
	if cfg.Diversity.TypeWindow <= 0 {
		cfg.Diversity.TypeWindow = defaultTypeWindow
	}
	if cfg.Diversity.MaxSameType <= 0 {
		cfg.Diversity.MaxSameType = defaultMaxSameType
	}
	if cfg.Diversity.NewContentTopN <= 0 {
		cfg.Diversity.NewContentTopN = defaultNewContentTopN
	}
	if cfg.Diversity.NewContentMinCount <= 0 {
		cfg.Diversity.NewContentMinCount = defaultNewContentMin
	}
	return cfg
}
