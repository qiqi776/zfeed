import { stressStages, stageOptions } from "../config.js";
import { runMixed, setupWorkload } from "../lib/workload.js";

export const options = stageOptions(stressStages);

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runMixed(state, 0.4);
}

