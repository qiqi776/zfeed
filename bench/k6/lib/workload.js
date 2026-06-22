import { sleep } from "k6";
import { benchConfig } from "../config.js";
import { registerAndLogin } from "./auth.js";
import { loadContentIDs, loadFollowEdges, loadSearchTerms, loadUsers, pick } from "./data.js";
import { check2xx, deleteJSON, getJSON, jsonField, postJSON } from "./http.js";

export function setupWorkload() {
  const users = loadUsers();
  const contents = loadContentIDs();
  const followEdges = loadFollowEdges();
  const terms = loadSearchTerms();
  const author = registerAndLogin(benchConfig.baseURL, users[0] || {}, 1);
  const viewer = registerAndLogin(benchConfig.baseURL, users[1] || users[0] || {}, 2);
  const title = `bench_article_${Date.now()}`;

  const publishRes = postJSON(
    benchConfig.baseURL,
    "/v1/content/article/publish",
    {
      title,
      description: "benchmark smoke article",
      cover: "https://example.com/bench/cover.png",
      content: "hello benchmark",
      visibility: 10,
    },
    author.token,
    { name: "content_publish_article", module: "content", kind: "write", auth: "required" },
  );
  check2xx(publishRes, "content_publish_article");

  const fallbackContent = contents[0] || {};
  const contentID = Number(jsonField(publishRes, "content_id", fallbackContent.content_id || 0));

  return {
    author,
    viewer,
    contentID,
    contentUserID: author.userId || Number(fallbackContent.author_id || 0),
    fixtureContents: contents,
    fixtureUsers: users,
    followEdges,
    searchQuery: title,
    searchTerms: terms,
  };
}

export function runSmoke(state) {
  getMe(state);
  queryProfile(state);
  recommendFeed(state);
  contentDetail(state);
  likeContent(state);
  queryLikeInfo(state);
  favoriteContent(state);
  queryFavoriteInfo(state);
  searchContents(state);
  searchUsers(state);
}

export function runMixed(state, writeRatio) {
  if (Math.random() < writeRatio) {
    pick([
      likeContent,
      unlikeContent,
      favoriteContent,
      commentContent,
      followAuthor,
      unfollowAuthor,
    ])(state);
    sleep(0.2);
    return;
  }

  pick([
    getMe,
    queryProfile,
    recommendFeed,
    contentDetail,
    queryLikeInfo,
    queryFavoriteInfo,
    commentList,
    followFeed,
    searchContents,
    searchUsers,
    userPublishFeed,
    userFavoriteFeed,
    queryFollowers,
  ])(state);
  sleep(0.2);
}

export function runSearch(state) {
  if (Math.random() < 0.7) {
    searchContents(state);
    return;
  }
  searchUsers(state);
}

export function runHotContent(state) {
  likeContent(state);
  queryLikeInfo(state);
}

function activeToken(state) {
  return state.viewer.token || state.author.token;
}

function activeUserID(state) {
  return state.viewer.userId || state.author.userId;
}

function useFixtureIDs() {
  return __ENV.BENCH_USE_FIXTURE_IDS === "1";
}

function liveContent(state) {
  return {
    contentID: Number(state.contentID || 0),
    contentUserID: Number(state.contentUserID || state.author.userId || activeUserID(state) || 0),
    scene: "ARTICLE",
  };
}

function sampledContent(state) {
  if (!useFixtureIDs()) {
    return liveContent(state);
  }

  const content = pick(state.fixtureContents);
  if (!content) {
    return liveContent(state);
  }

  const contentID = Number(content.content_id || 0);
  const contentUserID = Number(content.author_id || 0);
  if (contentID === 0 || contentUserID === 0) {
    return liveContent(state);
  }

  return {
    contentID,
    contentUserID,
    scene: content.scene || "ARTICLE",
  };
}

function sampledAuthorID(state) {
  if (!useFixtureIDs()) {
    return Number(state.author.userId || activeUserID(state));
  }

  const user = pick(state.fixtureUsers);
  return Number((user && user.user_id) || state.author.userId || activeUserID(state));
}

function sampledFolloweeID(state) {
  if (!useFixtureIDs()) {
    return Number(state.author.userId || activeUserID(state));
  }

  const edge = pick(state.followEdges);
  return Number((edge && edge.followee_id) || state.author.userId || activeUserID(state));
}

function getMe(state) {
  const res = getJSON(benchConfig.baseURL, "/v1/users/me", activeToken(state), {
    name: "user_get_me",
    module: "user",
    kind: "read",
    auth: "required",
  });
  check2xx(res, "user_get_me");
}

function queryProfile(state) {
  const res = getJSON(benchConfig.baseURL, `/v1/user/profile/${sampledAuthorID(state)}`, activeToken(state), {
    name: "user_profile",
    module: "user",
    kind: "read",
    auth: "required",
  });
  check2xx(res, "user_profile");
}

function recommendFeed(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/feed/recommend",
    { cursor: "0", page_size: 10 },
    activeToken(state),
    { name: "feed_recommend", module: "feed", kind: "read", auth: "optional" },
  );
  check2xx(res, "feed_recommend");
}

function userPublishFeed(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/feed/user/publish",
    { user_id: String(sampledAuthorID(state)), cursor: "0", page_size: 10 },
    activeToken(state),
    { name: "feed_user_publish", module: "feed", kind: "read", auth: "optional" },
  );
  check2xx(res, "feed_user_publish");
}

function followFeed(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/feed/follow",
    { cursor: "0", page_size: 10 },
    activeToken(state),
    { name: "feed_follow", module: "feed", kind: "read", auth: "required" },
  );
  check2xx(res, "feed_follow");
}

function userFavoriteFeed(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/feed/user/favorite",
    { user_id: String(activeUserID(state)), cursor: "0", page_size: 10 },
    activeToken(state),
    { name: "feed_user_favorite", module: "feed", kind: "read", auth: "required" },
  );
  check2xx(res, "feed_user_favorite");
}

function contentDetail(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/content/detail",
    { content_id: String(content.contentID) },
    activeToken(state),
    { name: "content_detail", module: "content", kind: "read", auth: "optional" },
  );
  check2xx(res, "content_detail");
}

function likeContent(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/like",
    { content_id: String(content.contentID), content_user_id: String(content.contentUserID), scene: content.scene },
    activeToken(state),
    { name: "interaction_like", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_like");
}

function unlikeContent(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/unlike",
    { content_id: String(content.contentID), scene: content.scene },
    activeToken(state),
    { name: "interaction_unlike", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_unlike");
}

function queryLikeInfo(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/like/info",
    { content_id: String(content.contentID), scene: content.scene },
    activeToken(state),
    { name: "interaction_like_info", module: "interaction", kind: "read", auth: "optional" },
  );
  check2xx(res, "interaction_like_info");
}

function favoriteContent(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/favorite",
    { content_id: String(content.contentID), scene: content.scene },
    activeToken(state),
    { name: "interaction_favorite", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_favorite");
}

function queryFavoriteInfo(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/favorite/info",
    { content_id: String(content.contentID), scene: content.scene },
    activeToken(state),
    { name: "interaction_favorite_info", module: "interaction", kind: "read", auth: "optional" },
  );
  check2xx(res, "interaction_favorite_info");
}

function commentContent(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/comment",
    {
      content_id: String(content.contentID),
      content_user_id: String(content.contentUserID),
      scene: content.scene,
      comment: `bench comment ${Date.now()}`,
    },
    activeToken(state),
    { name: "interaction_comment", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_comment");
}

function commentList(state) {
  const content = sampledContent(state);
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/comment/list",
    { content_id: String(content.contentID), scene: content.scene, cursor: 0, page_size: 10 },
    activeToken(state),
    { name: "interaction_comment_list", module: "interaction", kind: "read", auth: "optional" },
  );
  check2xx(res, "interaction_comment_list");
}

function followAuthor(state) {
  const targetUserID = sampledAuthorID(state);
  if (targetUserID === activeUserID(state)) {
    return;
  }
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/interaction/followings",
    { target_user_id: String(targetUserID) },
    activeToken(state),
    { name: "interaction_follow", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_follow");
}

function unfollowAuthor(state) {
  const targetUserID = sampledFolloweeID(state);
  if (targetUserID === activeUserID(state)) {
    return;
  }
  const res = deleteJSON(
    benchConfig.baseURL,
    "/v1/interaction/followings",
    { target_user_id: String(targetUserID) },
    activeToken(state),
    { name: "interaction_unfollow", module: "interaction", kind: "write", auth: "required" },
  );
  check2xx(res, "interaction_unfollow");
}

function queryFollowers(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/user/followers",
    { user_id: String(sampledAuthorID(state)), cursor: 0, page_size: 10 },
    activeToken(state),
    { name: "user_followers", module: "user", kind: "read", auth: "optional" },
  );
  check2xx(res, "user_followers");
}

function searchContents(state) {
  const term = pick(state.searchTerms);
  const query = (term && term.query) || state.searchQuery || "bench";
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/search/contents",
    { query, cursor: 0, page_size: 10, mode: "latest" },
    activeToken(state),
    { name: "search_contents", module: "search", kind: "read", auth: "optional" },
  );
  check2xx(res, "search_contents");
}

function searchUsers(state) {
  const res = postJSON(
    benchConfig.baseURL,
    "/v1/search/users",
    { query: "bench", cursor: 0, page_size: 10, mode: "latest" },
    activeToken(state),
    { name: "search_users", module: "search", kind: "read", auth: "optional" },
  );
  check2xx(res, "search_users");
}
