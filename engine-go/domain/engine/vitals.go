package engine

// classVitals mirrors CLASS_VITALS (class-vitals.ts).
type classVitals struct {
	pvInicial  int
	pvPerLevel int
	mpPerLevel int
}

var classVitalsTable = map[string]classVitals{
	"Arcanista": {8, 2, 6},
	"Bárbaro":   {24, 6, 3},
	"Bardo":     {12, 3, 4},
	"Bucaneiro": {16, 4, 3},
	"Caçador":   {16, 4, 4},
	"Cavaleiro": {20, 5, 3},
	"Clérigo":   {16, 4, 5},
	"Druida":    {16, 4, 4},
	"Guerreiro": {20, 5, 3},
	"Inventor":  {12, 3, 4},
	"Ladino":    {12, 3, 4},
	"Lutador":   {20, 5, 3},
	"Nobre":     {16, 4, 4},
	"Paladino":  {20, 5, 3},
}

// pvPoolWithCon mirrors class-vitals.ts pvPoolWithCon (p34 min-1 floor).
func pvPoolWithCon(v classVitals, level, con int) int {
	perLevel := max(1, v.pvPerLevel+con)
	return v.pvInicial + con + (level-1)*perLevel
}

// multiclassPvPool: only the first class seeds its PV inicial (p34-35).
func multiclassPvPool(classes []ClassEntry, con int) int {
	if len(classes) == 0 {
		return 0
	}
	seed, ok := classVitalsTable[classes[0].ClassName]
	if !ok {
		return 0
	}
	pv := pvPoolWithCon(seed, classes[0].Level, con)
	for _, c := range classes[1:] {
		entry, ok := classVitalsTable[c.ClassName]
		if !ok {
			continue
		}
		pv += c.Level * max(1, entry.pvPerLevel+con)
	}
	return pv
}

// multiclassMpPool sums each class's mpPerLevel*level (p35).
func multiclassMpPool(classes []ClassEntry) int {
	mp := 0
	for _, c := range classes {
		if entry, ok := classVitalsTable[c.ClassName]; ok {
			mp += entry.mpPerLevel * c.Level
		}
	}
	return mp
}

// Everything below this point used to be the MVP engine's own vital-grant
// collector, deleted with it (the catalog-driven `vitals_v2.go` is what the app
// runs). What survives above is `classVitalsTable`, `pvPoolWithCon`,
// `multiclassPvPool` and `multiclassMpPool`, which vitals_v2 still calls.

// ClassStartingVitals são o PV inicial e o PM por nível de uma classe (p34).
//
// Exportados para a FORJA: a carta de classe precisa dizer o que a escolha
// compra — "PV 20 · PM 3" — antes de existir personagem para calcular. Os dois
// números são os da tabela, sem Constituição e sem nível, porque é isso que o
// bloco da classe imprime.
func ClassStartingVitals(className string) (pvInicial, pmPorNivel int, ok bool) {
	v, ok := classVitalsTable[className]
	if !ok {
		return 0, 0, false
	}
	return v.pvInicial, v.mpPerLevel, true
}
