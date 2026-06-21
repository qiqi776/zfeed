import { benchConfig } from "../config.js";

function parseCSV(raw) {
  const lines = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"));
  if (lines.length === 0) {
    return [];
  }

  const headers = lines[0].split(",").map((header) => header.trim());
  return lines.slice(1).map((line) => {
    const values = line.split(",");
    const row = {};
    headers.forEach((header, index) => {
      row[header] = (values[index] || "").trim();
    });
    return row;
  });
}

const users = parseCSV(open(`${benchConfig.dataDir}/users.csv`));
const contentIDs = parseCSV(open(`${benchConfig.dataDir}/content_ids.csv`));
const followEdges = parseCSV(open(`${benchConfig.dataDir}/follow_edges.csv`));
const searchTerms = parseCSV(open(`${benchConfig.dataDir}/search_terms.csv`));

export function loadUsers() {
  return users;
}

export function loadContentIDs() {
  return contentIDs;
}

export function loadFollowEdges() {
  return followEdges;
}

export function loadSearchTerms() {
  return searchTerms;
}

export function pick(values) {
  if (!values || values.length === 0) {
    return undefined;
  }
  return values[Math.floor(Math.random() * values.length)];
}

export function pickUser() {
  return pick(users);
}

export function pickContent() {
  return pick(contentIDs);
}

export function pickFollowEdge() {
  return pick(followEdges);
}

export function pickSearchTerm() {
  return pick(searchTerms);
}
