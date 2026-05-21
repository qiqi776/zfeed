import { loadStages, stageOptions } from "../config.js";
import { runSearch, setupWorkload } from "../lib/workload.js";

export const options = stageOptions(loadStages);

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runSearch(state);
}

