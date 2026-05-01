package mobilex

import "strings"

func Normalize(value string) string {
	compact := compact(value)
	switch {
	case isCNWithPlus86(compact):
		return compact[3:]
	case isCNWith86(compact):
		return compact[2:]
	default:
		return compact
	}
}

func IsValid(value string) bool {
	compact := compact(value)
	return isCNRaw(compact) || isCNWith86(compact) || isCNWithPlus86(compact) || isE164(compact)
}

func LookupCandidates(value string) []string {
	compact := compact(value)
	if compact == "" {
		return nil
	}

	candidates := []string{compact}
	switch {
	case isCNRaw(compact):
		candidates = append(candidates, "86"+compact, "+86"+compact)
	case isCNWith86(compact):
		raw := compact[2:]
		candidates = append(candidates, raw, "+86"+raw)
	case isCNWithPlus86(compact):
		raw := compact[3:]
		candidates = append(candidates, raw, "86"+raw)
	}

	return unique(candidates)
}

func compact(value string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	return strings.TrimSpace(replacer.Replace(value))
}

func isCNRaw(value string) bool {
	return len(value) == 11 && value[0] == '1' && digitsOnly(value)
}

func isCNWith86(value string) bool {
	return len(value) == 13 && strings.HasPrefix(value, "86") && isCNRaw(value[2:])
}

func isCNWithPlus86(value string) bool {
	return len(value) == 14 && strings.HasPrefix(value, "+86") && isCNRaw(value[3:])
}

func isE164(value string) bool {
	if len(value) < 8 || len(value) > 16 || value[0] != '+' {
		return false
	}
	if value[1] < '1' || value[1] > '9' {
		return false
	}
	return digitsOnly(value[1:])
}

func digitsOnly(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func unique(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
