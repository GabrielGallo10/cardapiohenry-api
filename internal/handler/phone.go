package handler

import "strings"

// digitsOnlyPhone extrai apenas dígitos — deve ser o mesmo critério usado em Register ao gravar.
func digitsOnlyPhone(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// phoneLookupCandidates devolve o número normalizado e variantes comuns (DDI 55) para bater com o que está na BD.
func phoneLookupCandidates(digits string) []string {
	if digits == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(digits)
	// BR: 11 dígitos (DDD + celular) sem DDI → também procurar com 55
	if len(digits) == 11 {
		add("55" + digits)
	}
	// Valor com DDI 55 e ≥12 dígitos → também sem os dois primeiros
	if strings.HasPrefix(digits, "55") && len(digits) >= 12 {
		add(digits[2:])
	}
	return out
}
