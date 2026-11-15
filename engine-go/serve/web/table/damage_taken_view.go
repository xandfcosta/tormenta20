package table

import (
	"context"
	"fmt"
	"log"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// COMO O DANO ATINGE A FICHA, dito ao mestre onde ele aplica o dano (ALE-650).
//
// Quase todo dano chega ao personagem pela mão do mestre — NPC não ataca pelo app —, num número sem tipo. O
// app não tem como aplicar ali o +1 por dado dos Ossos Frágeis, a imunidade da Fonte Elemental ou a Evasão, e
// por isso os DIZ: na folha do combatente, ao lado do editor de PV, e na faixa da bola de fogo que caiu nele.
// É do mestre e só dele: o jogador não lê a ficha de outra entidade.

// damageTakenSentences são as frases de `engine.DamageTaken`, uma por linha, ou nenhuma.
func damageTakenSentences(taken engine.DamageTaken) []string {
	var out []string
	for _, line := range taken.PerDie {
		out = append(out, fmt.Sprintf("%s por dado de %s (%s)", signed(int64(line.Amount)), line.Type, line.Source))
	}
	for _, line := range taken.Immune {
		out = append(out, fmt.Sprintf("imune a %s (%s)", line.Type, line.Source))
	}
	for _, line := range taken.HealedByMagic {
		out = append(out, fmt.Sprintf("o dano mágico de %s cura metade, em vez de ferir (%s)", line.Type, line.Source))
	}
	switch taken.Evasion.Type {
	case engine.EvasionFlag:
		out = append(out, fmt.Sprintf("no Reflexos que reduz à metade, passar zera o dano (%s)", taken.Evasion.Source))
	case engine.ImprovedEvasionFlag:
		out = append(out, fmt.Sprintf("no Reflexos que reduz à metade, passar zera o dano e falhar sofre metade (%s)",
			taken.Evasion.Source))
	}
	return out
}

// damageTakenOf lê, de cada ficha na mesa, como o dano a atinge. Só entra quem tem alguma frase.
//
// Melhor esforço, como a Defesa do Grupo: uma ficha que não monta fica sem as frases, e a mesa continua de pé —
// a iniciativa é o assunto da tela.
func (s Scene) damageTakenOf(ctx context.Context, ids []int64) map[int64][]string {
	out := map[int64][]string{}
	if s.deps.Catalogs() == nil {
		return out
	}
	for _, id := range ids {
		row, err := s.deps.Queries().GetCharacter(ctx, id)
		if err != nil {
			log.Printf("mesa: ler a ficha %d para o dano sofrido falhou (%v)", id, err)
			continue
		}
		dto, err := sheet.Load(ctx, s.deps.Queries(), s.deps.Catalogs(), row)
		if err != nil {
			log.Printf("mesa: montar a ficha %d para o dano sofrido falhou (%v)", id, err)
			continue
		}
		ec, err := sheet.EngineCharacterFrom(dto)
		if err != nil {
			log.Printf("mesa: a ficha %d não vira personagem do motor (%v)", id, err)
			continue
		}
		if sentences := damageTakenSentences(dto.Ruleset.DamageTakenOf(ec, sheet.ToStringSet(dto.Conditionals))); len(sentences) > 0 {
			out[id] = sentences
		}
	}
	return out
}
