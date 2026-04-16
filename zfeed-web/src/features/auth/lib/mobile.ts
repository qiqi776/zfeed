export function normalizeMobileInput(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }

  if (/^1\d{10}$/.test(trimmed)) {
    return `+86${trimmed}`;
  }

  if (/^86\d{11}$/.test(trimmed)) {
    return `+${trimmed}`;
  }

  return trimmed;
}

export function isLikelyE164Mobile(value: string) {
  return /^\+[1-9]\d{7,15}$/.test(value.trim());
}
