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

// tormentaCarismaLoss mirrors character-sheet.ts tormentaCarismaLoss: counts
// real Tormenta powers picked (powerIds ∩ catalog) plus the Deformidade swap,
// then applies the scaling loss once over the total.
func tormentaCarismaLoss(in *CharacterInput) int {
	var picked []string
	for _, id := range in.PowerIDs {
		if tormentaPowerIDs[id] {
			picked = append(picked, id)
		}
	}
	held := ""
	if in.Deformidade != nil {
		held = in.Deformidade.TormentaPower
	}
	count := len(picked)
	if held != "" && !contains(picked, held) {
		count++
	}
	return carismaLossFromPowers(count)
}

// expertiseToSkill mirrors deformidade.ts: NFD-strip + lowercase, then
// match against skillIndex. The bundled table covers the accented PT names.
var expertiseToSkill = map[string]string{
	"acrobacia": "acrobacia", "adestramento": "adestramento", "atletismo": "atletismo",
	"atuacao": "atuacao", "cavalgar": "cavalgar", "conhecimento": "conhecimento",
	"cura": "cura", "diplomacia": "diplomacia", "enganacao": "enganacao",
	"fortitude": "fortitude", "furtividade": "furtividade", "guerra": "guerra",
	"iniciativa": "iniciativa", "intimidacao": "intimidacao", "intuicao": "intuicao",
	"investigacao": "investigacao", "jogatina": "jogatina", "ladinagem": "ladinagem",
	"luta": "luta", "misticismo": "misticismo", "nobreza": "nobreza", "oficio": "oficio",
	"percepcao": "percepcao", "pilotagem": "pilotagem", "pontaria": "pontaria",
	"reflexos": "reflexos", "religiao": "religiao", "sobrevivencia": "sobrevivencia",
	"vontade": "vontade",
}

// deformidadeSkillIDs returns the +2 perícia skill ids from the Deformidade
// choice (deformidade.ts deformidadeSkillIds).
func deformidadeSkillIDs(d *Deformidade) []string {
	if d == nil {
		return nil
	}
	var out []string
	for _, name := range d.Pericias {
		if id, ok := expertiseToSkill[stripAccentsLower(name)]; ok {
			out = append(out, id)
		}
	}
	return out
}
