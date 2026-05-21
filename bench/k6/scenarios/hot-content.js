import { stressStages, stageOptions } from "../config.js";
import { runHotContent, setupWorkload } from "../lib/workload.js";

export const options = stageOptions(stressStages);

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runHotContent(state);
}
