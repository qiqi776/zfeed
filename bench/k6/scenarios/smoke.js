import { smokeOptions } from "../config.js";
import { runSmoke, setupWorkload } from "../lib/workload.js";

export const options = smokeOptions();

export function setup() {
  return setupWorkload();
}

export default function (state) {
  runSmoke(state);
}

