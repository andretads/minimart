package application

import "strings"

func NormalizeUsernames(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	out := make([]string, 0)
	for p := range strings.SplitSeq(input, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
