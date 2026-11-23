package engine

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func hasDuplicates(s []string) bool {
	seen := map[string]bool{}
	for _, x := range s {
		if seen[x] {
			return true
		}
		seen[x] = true
	}
	return false
}

func toSet(s []string) map[string]bool {
	out := make(map[string]bool, len(s))
	for _, x := range s {
		out[x] = true
	}
	return out
}
