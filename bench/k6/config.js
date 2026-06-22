export const benchConfig = {
  baseURL: __ENV.BASE_URL || "http://127.0.0.1:18080",
  dataDir: __ENV.DATA_DIR || "../../data/small",
  defaultPassword: __ENV.BENCH_PASSWORD || "123456Aa!",
};

export const defaultThresholds = {
  http_req_failed: ["rate<0.005"],
  http_req_duration: ["p(95)<500", "p(99)<1500"],
  checks: ["rate>0.995"],
};

export const taggedThresholds = {
  "http_req_duration{name:feed_recommend}": ["p(95)<300", "p(99)<1000"],
  "http_req_duration{name:content_detail}": ["p(95)<400", "p(99)<1200"],
  "http_req_duration{name:interaction_like}": ["p(95)<300", "p(99)<800"],
  "http_req_duration{name:interaction_favorite}": ["p(95)<300", "p(99)<800"],
  "http_req_duration{name:search_contents}": ["p(95)<500", "p(99)<1500"],
};

export function smokeOptions() {
  return {
    summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
    thresholds: { ...defaultThresholds, ...taggedThresholds },
    scenarios: {
      smoke: {
        executor: "shared-iterations",
        vus: Number(__ENV.VUS || 1),
        iterations: Number(__ENV.ITERATIONS || 1),
        maxDuration: __ENV.MAX_DURATION || "5m",
      },
    },
  };
}

export function stageOptions(stages) {
  return {
    summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
    thresholds: { ...defaultThresholds, ...taggedThresholds },
    stages,
  };
}

export const loadStages = [
  { duration: "2m", target: 10 },
  { duration: "5m", target: 25 },
  { duration: "5m", target: 50 },
  { duration: "2m", target: 0 },
];

export const stressStages = [
  { duration: "2m", target: 10 },
  { duration: "5m", target: 25 },
  { duration: "5m", target: 50 },
  { duration: "2m", target: 0 },
];

export const spikeStages = [
  { duration: "2m", target: 10 },
  { duration: "1m", target: 400 },
  { duration: "5m", target: 10 },
  { duration: "1m", target: 800 },
  { duration: "10m", target: 10 },
  { duration: "2m", target: 0 },
];

export const soakStages = [
  { duration: "5m", target: Number(__ENV.SOAK_VUS || 50) },
  { duration: __ENV.SOAK_DURATION || "2h", target: Number(__ENV.SOAK_VUS || 50) },
  { duration: "5m", target: 0 },
];
