package engine

// tormentaPowerIDs — the poderes da Tormenta catalog ids (tormenta.ts). Used
// to detect which powerIds / deformidade swaps are real Tormenta powers.
var tormentaPowerIDs = map[string]bool{
	"anatomia-insana": true, "antenas": true, "armamento-aberrante": true,
	"articulacoes-flexiveis": true, "asas-insetoides": true, "carapaca": true,
	"corpo-aberrante": true, "cuspir-enxame": true, "dentes-afiados": true,
	"desprezar-a-realidade": true, "empunhadura-rubra": true, "fome-de-mana": true,
	"larva-explosiva": true, "legiao-aberrante": true, "maos-membranosas": true,
	"membros-estendidos": true, "membros-extras": true, "mente-aberrante": true,
	"olhos-vermelhos": true, "pele-corrompida": true, "sangue-acido": true,
	"visco-rubro": true,
}

// carismaLossFromPowers mirrors tormenta.ts: the N-th power costs
// 1 + floor((N-1)/2) Carisma. Sequence: 1, 2, 4, 6, 9, 12…
func carismaLossFromPowers(powerCount int) int {
	total := 0
	for k := 1; k <= powerCount; k++ {
		total += 1 + (k-1)/2
	}
	return total
}
