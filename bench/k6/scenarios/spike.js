import { spikeStages, stageOptions } from "../config.js";
import { runMixed, setupWorkload } from "../lib/workload.js";

export const options = stageOptions(spikeStages);

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runMixed(state, Number(__ENV.WRITE_RATIO || 0.2));
}

