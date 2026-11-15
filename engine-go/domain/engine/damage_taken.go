package engine

import (
	"slices"
	"strings"
)

// O QUE O CORPO FAZ COM O DANO QUE SOFRE (ALE-650).
//
// Cinco habilidades do capítulo 1 mudam o dano que CHEGA ao personagem, e não o que ele causa: o +1 por dado
// de um tipo (Ossos Frágeis, p28; Sangue Frio, p31), a imunidade a um tipo e a cura pelo dano mágico dele
// (Fonte Elemental, p27), e o teste de Reflexos que zera a metade (Evasão e Evasão Aprimorada, p47, p48, p73,
// p74).
//
// # Por que isto é uma leitura, e não um número da ficha
//
// Quase todo dano chega ao personagem pela MÃO do mestre: NPC não ataca pelo app (só ficha empunha carta), e o
// mestre tira PV na linha da fila com um número sem tipo. O app não tem como aplicar ali o que não sabe — de
// que tipo era o dano, quantos dados, se era mágico, se o efeito permitia Reflexos. O que ele faz é DIZER ao
// mestre, no lugar em que o mestre aplica o dano, como ele atinge esta ficha. Onde o app SABE tipo e dados — o
// golpe de uma ficha em outra —, o golpe aplica sozinho (`theTargetsBodyTakesIt`).

// Os três alvos do catálogo, todos com o tipo de dano no `DamageType`.
const (
	// DamageTakenPerDieTarget: "+N por dado de dano deste tipo" que o personagem sofre.
	DamageTakenPerDieTarget = "damageTakenPerDie"
	// DamageImmunityTarget: imune a dano deste tipo.
	DamageImmunityTarget = "damageImmunity"
	// MagicDamageHealsTarget: o dano MÁGICO deste tipo cura metade em vez de ferir.
	MagicDamageHealsTarget = "magicDamageHeals"
)

// As duas flags da Evasão. A aprimorada contém a outra, e por isso a leitura devolve uma só.
const (
	EvasionFlag         = "evasao"
	ImprovedEvasionFlag = "evasao-aprimorada"
)

// DamageTakenLine é uma linha do que o corpo faz: o tipo de dano (ou a flag, na Evasão), quanto, e quem dá.
type DamageTakenLine struct {
	Type   string `json:"type"`
	Amount int    `json:"amount"`
	Source string `json:"source"`
}

// DamageTaken é o que o corpo do personagem faz com o dano que sofre. Vazio é o comum.
type DamageTaken struct {
	PerDie        []DamageTakenLine
	Immune        []DamageTakenLine
	HealedByMagic []DamageTakenLine
	// Evasion é a Evasão que vale agora — a aprimorada, quando há as duas —, ou o zero.
	Evasion DamageTakenLine
}

// DamageTakenOf é como o dano atinge a ficha, pelos efeitos que valem agora: a Evasão de armadura pesada não
// aparece, porque não vale.
//
//	ruleset.DamageTakenOf(ch, sheet.ToStringSet(dto.Conditionals)).PerDie // → [{impacto 1 Ossos Frágeis}]
func (r *Ruleset) DamageTakenOf(ch Character, switches map[string]bool) DamageTaken {
	return DamageTakenFrom(r.EffectsOf(ch, switches))
}

// DamageTakenFrom é a mesma leitura sobre efeitos já resolvidos, para quem já os tem.
func DamageTakenFrom(effects ItemEffects) DamageTaken {
	out := DamageTaken{
		PerDie:        typedDamageLines(effects, DamageTakenPerDieTarget),
		Immune:        typedDamageLines(effects, DamageImmunityTarget),
		HealedByMagic: typedDamageLines(effects, MagicDamageHealsTarget),
	}
	for _, flag := range []string{ImprovedEvasionFlag, EvasionFlag} {
		if effects.Flags[flag] {
			out.Evasion = DamageTakenLine{Type: flag, Amount: 1, Source: effects.FlagSources[flag]}
			break
		}
	}
	return out
}

// typedDamageLines são as linhas de um dos três alvos, uma por tipo de dano, na ordem do tipo.
func typedDamageLines(effects ItemEffects, target string) []DamageTakenLine {
	var out []DamageTakenLine
	for key, stat := range effects.ByTarget {
		damageType, found := strings.CutPrefix(key, target+":")
		if !found || stat.Total <= 0 {
			continue
		}
		sources := make([]string, 0, len(stat.Contributions))
		for _, c := range stat.Contributions {
			// A RAÇA é uma fonte só ("Raça: Kliren"), e o nome da habilidade viaja na nota do modificador.
			name := c.Source
			if c.Note != "" {
				name = c.Note
			}
			sources = append(sources, name)
		}
		out = append(out, DamageTakenLine{Type: damageType, Amount: stat.Total, Source: strings.Join(sources, ", ")})
	}
	slices.SortFunc(out, func(a, b DamageTakenLine) int { return strings.Compare(a.Type, b.Type) })
	return out
}
