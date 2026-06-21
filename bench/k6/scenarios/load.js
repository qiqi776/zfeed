import { loadStages, stageOptions } from "../config.js";
import { runMixed, setupWorkload } from "../lib/workload.js";

export const options = stageOptions(loadStages);

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runMixed(state, Number(__ENV.WRITE_RATIO || 0.2));
}
