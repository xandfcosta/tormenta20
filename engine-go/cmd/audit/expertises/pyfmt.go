package main

// pyOptionalStr é o `str()` de uma str que pode ser `None` — aqui, "".
func pyOptionalStr(s string) string {
	if s == "" {
		return "None"
	}
	return s
}
