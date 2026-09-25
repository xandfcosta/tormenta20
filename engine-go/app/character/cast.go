package character

import (
	"context"
	"fmt"

	"t20engine/domain/catalog"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS TRÊS ESCRITAS DO GRIMÓRIO — aprender, esquecer e alternar o preparo.
//
// Elas desceram da cena na ALE-350, e a do meio é a que explica por quê: o
// `TogglePrepared` LÊ o grimório, DECIDE o estado novo e grava. As três coisas
// que o guia usa para reconhecer um caso de uso, num gesto de um clique.

// LearnSpell põe uma magia do catálogo no grimório.
//
// O livro é conferido AQUI e não por quem chama: uma magia que não existe seria
// uma linha órfã no grimório, que a ficha desenha como um espaço em branco.
func (p Plays) LearnSpell(ctx context.Context, characterID int64, catalogSpellID string) error {
	if _, known := catalog.LookupSpell(catalogSpellID); !known {
		return fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	if _, err := p.queries.CreateSpell(ctx, sqlcgen.CreateSpellParams{
		Characterid: characterID, Catalogspellid: catalogSpellID,
		Prepared: 0, Learnedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("aprender %q na ficha %d: %w", catalogSpellID, characterID, err)
	}
	return nil
}

// ForgetSpell tira a magia do grimório.
func (p Plays) ForgetSpell(ctx context.Context, characterID int64, catalogSpellID string) error {
	if _, err := p.queries.DeleteSpell(ctx, sqlcgen.DeleteSpellParams{
		Characterid: characterID, Catalogspellid: catalogSpellID,
	}); err != nil {
		return fmt.Errorf("esquecer %q na ficha %d: %w", catalogSpellID, characterID, err)
	}
	return nil
}

// TogglePrepared prepara ou desprepara uma magia.
//
// Ela recebe a MAGIA e não o estado desejado: mandar "preparada" perde para o
// clique repetido e para a segunda aba aberta no mesmo personagem — quem sabe o
// que está lá é o servidor, e é ele que inverte.
func (p Plays) TogglePrepared(ctx context.Context, characterID int64, catalogSpellID string) error {
	all, err := p.queries.ListSpellsByCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("ler o grimório da ficha %d: %w", characterID, err)
	}
	for _, m := range all {
		if m.Catalogspellid != catalogSpellID {
			continue
		}
		after := int64(0)
		if m.Prepared == 0 {
			after = 1
		}
		if _, err := p.queries.SetSpellPreparedByCatalog(ctx, sqlcgen.SetSpellPreparedByCatalogParams{
			Prepared: after, CharacterId: characterID, CatalogSpellId: catalogSpellID,
		}); err != nil {
			return fmt.Errorf("gravar o preparo de %q: %w", catalogSpellID, err)
		}
		return nil
	}
	return fmt.Errorf("a magia %q não está no grimório", catalogSpellID)
}

// Cast é a conjuração INTEIRA: as recusas do livro, o custo em PM e a baixa.
//
// Ela nasceu extraída na ALE-272 (fatia 6), quando a ficha em Datastar passou a
// conjurar: escrever as recusas de novo lá daria DUAS regras para a mesma
// pergunta, e elas divergiriam no dia em que uma mudasse. É a mesma razão da
// ALE-110, que registrou o custo sendo exibido num lugar e ignorado no outro.
//
// A recusa volta como uma FRASE para um humano ler numa tela — ver por que ela
// não é tipada no `doc.go` do pacote.
func (p Plays) Cast(
	ctx context.Context, dto sheet.CharacterDTO, catalogSpellID string, augments []sheet.AugmentPick,
) error {
	spell, known := catalog.LookupSpell(catalogSpellID)
	if !known {
		return fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	learned := findSpell(dto.Spells, catalogSpellID)
	if learned == nil {
		return fmt.Errorf("%q não está no grimório desta ficha", catalogSpellID)
	}
	if sheet.RequiresPreparation(dto.Classes, dto.ClassChoices) && !learned.Prepared {
		return fmt.Errorf("prepare a magia antes de conjurá-la")
	}
	augmentPm, augErr := validateAugments(spell, augments,
		sheet.HighestCastableCircle(dto.Classes, spell.Circle))
	if augErr != "" {
		return fmt.Errorf("%s", augErr)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return err
	}
	basePm := sheet.SpellBasePmCost[spell.Circle]
	// O TRUQUE zera a conjuração INTEIRA e não só a parte dele: "reduz seu custo
	// em PM para zero" (p171). Tratado como um aprimoramento de +0 PM, o custo
	// base ficava de pé e a versão mais simples da magia saía pelo preço da
	// normal (ALE-339).
	//
	// O `augmentPm` também vai a zero, e não por precaução: o truque é
	// `Exclusive`, então o `validateAugments` já recusou qualquer companhia e não
	// há segunda parcela — zerar aqui diz isso em voz alta em vez de depender de
	// uma invariante que mora noutra função.
	if truqueEscolhido(spell, augments) {
		basePm, augmentPm = 0, 0
	}
	// OS CONDICIONAIS LIGADOS entram nas três contas. Aqui iam três conjuntos
	// VAZIOS, e o custo, o mínimo e o teto saíam de um personagem que não é o
	// que está conjurando (ALE-357).
	active := sheet.ToStringSet(dto.Conditionals)
	totalPm := dto.Ruleset.SpellPmCostFor(ec, basePm, augmentPm, active)
	minPm := dto.Ruleset.SpellPmCostFor(ec, basePm, 0, active)
	limit := dto.Ruleset.SpellPmLimitFor(ec, active, spell.Classes)
	if spell.Circle > 0 && totalPm > limit && totalPm > minPm {
		return fmt.Errorf("o custo de %d PM passa do limite de %d por magia", totalPm, limit)
	}
	if int64(totalPm) > dto.MpCurrent {
		return fmt.Errorf("faltam PM: a magia custa %d e restam %d", totalPm, dto.MpCurrent)
	}
	if totalPm == 0 {
		return nil
	}
	_, err = sheet.ApplyToLoadedPools(ctx, p.queries, &dto,
		func(pools sheet.Pools) (sheet.Pools, error) {
			pools.MpCurrent -= int64(totalPm)
			return pools, nil
		})
	return err
}

// truqueEscolhido diz se um dos aprimoramentos pedidos é um truque.
//
// Índice fora de faixa é ignorado de propósito: quem recusa o pedido malformado é
// o `validateAugments`, que roda antes, e repetir a checagem aqui daria duas
// respostas para a mesma pergunta.
func truqueEscolhido(spell catalog.Spell, picks []sheet.AugmentPick) bool {
	for _, p := range picks {
		if p.AugmentIndex >= 0 && p.AugmentIndex < len(spell.Augments) &&
			spell.Augments[p.AugmentIndex].Cantrip {
			return true
		}
	}
	return false
}

func findSpell(spells []sheet.SpellDTO, catalogSpellID string) *sheet.SpellDTO {
	for i := range spells {
		if spells[i].CatalogSpellID == catalogSpellID {
			return &spells[i]
		}
	}
	return nil
}

// validateAugments confere os aprimoramentos escolhidos e devolve o PM deles,
// ou a frase da recusa.
//
// É regra PURA do livro e mesmo assim não desce para o `domain/sheet`: ela
// recebe um `catalog.Spell`, e o guarda daquele pacote recusa o `catalog` por
// escrito — *"catalog (que é o arquivo cru)"*. O caminho para descer é unificar
// `catalog.Spell` com o `book.Spell` tipado, que é trabalho próprio e mexe nos
// dois lados. Até lá ela fica com o único chamador que tem.
//
// O `castableCircle` fechou uma FRONTEIRA que estava aberta (ALE-272, fatia 6):
// 126 dos 486 aprimoramentos do catálogo exigem um círculo mínimo, e até aqui
// esse limite existia só na tela. A tabela que o decide vivia só no TypeScript,
// então o servidor nem tinha como perguntar — e um pedido montado à mão
// conjurava o que a regra não permite. Travar na UI é UX; a fronteira é aqui.
func validateAugments(spell catalog.Spell, picks []sheet.AugmentPick, castableCircle int) (int, string) {
	if len(picks) == 0 {
		return 0, ""
	}
	if spell.Circle == 0 {
		return 0, "truque não recebe aprimoramento"
	}
	seen := map[int]bool{}
	total := 0
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(spell.Augments) {
			return 0, fmt.Sprintf("o aprimoramento %d não existe nesta magia", p.AugmentIndex)
		}
		if seen[p.AugmentIndex] {
			return 0, fmt.Sprintf("o aprimoramento %d veio duas vezes: junte as repetições num pedido só", p.AugmentIndex)
		}
		seen[p.AugmentIndex] = true
		if p.Stacks < 1 {
			return 0, fmt.Sprintf("o aprimoramento %d pede um número de repetições de 1 para cima (veio %d)", p.AugmentIndex, p.Stacks)
		}
		a := spell.Augments[p.AugmentIndex]
		// O "MUDA" não se repete: "Mudanças na mesma característica da magia
		// nunca se acumulam" (p171).
		if a.Kind == "muda" && p.Stacks > 1 {
			return 0, fmt.Sprintf("o aprimoramento %d muda a magia, e mudança não se acumula", p.AugmentIndex)
		}
		// O EXCLUSIVO recusa companhia: "truques não podem ser usados em conjunto
		// com outros aprimoramentos" (p171), e o livro repete a frase à mão na
		// esfera da Invisibilidade (p195) e na Luz permanente (p197).
		//
		// A pergunta é feita DENTRO do laço, sobre cada pedido, e não sobre o
		// primeiro: a ordem da lista é do cliente, e um ramo que olhasse só
		// `picks[0]` deixaria passar a mesma combinação escrita ao contrário.
		if a.Exclusive && len(picks) > 1 {
			return 0, fmt.Sprintf(
				"o aprimoramento %d não pode ser usado em conjunto com outros: escolha só ele",
				p.AugmentIndex)
		}
		if a.RequiresCircle != nil && *a.RequiresCircle > castableCircle {
			return 0, fmt.Sprintf(
				"aprimoramento %d exige o %dº círculo e este personagem alcança o %dº",
				p.AugmentIndex, *a.RequiresCircle, castableCircle)
		}
		total += a.PmCost * p.Stacks
	}
	return total, ""
}
