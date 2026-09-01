package sheet

// Os dois auxiliares da CRIAÇÃO: as proficiências que a classe concede e a
// conversão de lista para conjunto.
//
// Eles viajam com a `CreateBody` porque é ela que os consome — quem monta um
// herói precisa dos dois, e a forja monta heróis.

func ToStringSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
