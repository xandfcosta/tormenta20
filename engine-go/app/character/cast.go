package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
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

// ShapeSilence é a recusa e o aviso da forma que não conjura (p62), uma frase para a tela e o servidor.
const ShapeSilence = "Na Forma Selvagem você não lança magias (p62); a Magia Natural libera."

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
	ctx context.Context, dto sheet.CharacterDTO, catalogSpellID string,
	augments []sheet.AugmentPick, powerAugments []sheet.PowerAugmentPick,
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
	powerPm, powerErr := validatePowerAugments(dto, spell, catalogSpellID, augments, powerAugments)
	if powerErr != "" {
		return fmt.Errorf("%s", powerErr)
	}
	augmentPm += powerPm
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return err
	}
	if !engine.CastsInTheShape(ec) {
		return fmt.Errorf("%s", ShapeSilence)
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
	// QUEM LANÇA a magia é a pergunta das duas contas, e não a lista do verbete: o bardo de evocação lança a
	// Bola de Fogo, que lista só o Arcanista (p44).
	casting := sheet.CastingClasses(dto, dto.Classes, spell.School, spell.Classes)
	tradition := sheet.SpellTradition(dto.Classes, casting)
	totalPm := dto.Ruleset.SpellPmCostFor(ec, basePm, augmentPm, active, tradition, catalogSpellID)
	minPm := dto.Ruleset.SpellPmCostFor(ec, basePm, 0, active, tradition, catalogSpellID)
	limit := dto.Ruleset.SpellPmLimitFor(ec, active, casting)
	if spell.Circle > 0 && totalPm > limit && totalPm > minPm {
		return fmt.Errorf("o custo de %d PM passa do limite de %d por magia", totalPm, limit)
	}
	// O MANA DISPONÍVEL inclui a poça temporária, e as duas metades da p106
	// andam juntas: se o gasto drena a poça primeiro, a poça CONTA para decidir
	// se o gasto cabe. Com `dto.MpCurrent` aqui, o bardo de poço vazio e poça
	// cheia seria recusado — a poça existiria só para ser ignorada.
	if available := dto.ManaAvailable(); int64(totalPm) > available {
		return fmt.Errorf("faltam PM: a magia custa %d e restam %d", totalPm, available)
	}
	if !active[RitualSwitch] {
		return sheet.SpendLoadedMana(ctx, p.queries, &dto, totalPm)
	}
	// O RITUAL VALE PARA ESTA CONJURAÇÃO: é aqui e não no gesto porque ela pode ser recusada acima, e a recusa
	// não gasta o interruptor. E os dois saem na mesma transação: se desligar falhasse com o PM já cobrado, a
	// tela diria erro, o interruptor ficaria ligado, e tentar de novo cobraria outra vez.
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação do ritual: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := p.queries.WithTx(tx)
	if err := sheet.SpendLoadedMana(ctx, q, &dto, totalPm); err != nil {
		return err
	}
	if err := q.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
		Characterid: dto.ID, Conditionalid: RitualSwitch,
	}); err != nil {
		return fmt.Errorf("desligar o ritual da ficha %d: %w", dto.ID, err)
	}
	return tx.Commit()
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

// validatePowerAugments confere os poderes escolhidos como aprimoramento (p131) e
// devolve o PM deles, ou a frase da recusa.
//
// O poder que falta na ficha é recusado aqui porque quem desenha a linha é a tela,
// e a fronteira é o servidor: um pedido montado à mão pagaria o aprimoramento de
// um poder que o personagem não tem.
func validatePowerAugments(
	dto sheet.CharacterDTO, spell catalog.Spell, spellID string,
	spellPicks []sheet.AugmentPick, picks []sheet.PowerAugmentPick,
) (int, string) {
	if len(picks) == 0 {
		return 0, ""
	}
	fromBook, _ := book.SpellByID(spellID)
	owned := sheet.OwnedPowers(dto)
	total := 0
	for _, pick := range picks {
		augment, known := book.PowerAugmentOf(pick.PowerID)
		if !known {
			return 0, fmt.Sprintf("o poder %q não é aprimoramento de magia", pick.PowerID)
		}
		if !slices.ContainsFunc(owned, func(p sheet.OwnedPower) bool { return p.ID == pick.PowerID }) {
			return 0, fmt.Sprintf("esta ficha não tem %s", augment.Name)
		}
		if pick.Stacks < 1 {
			return 0, fmt.Sprintf("%s pede um número de repetições de 1 para cima (veio %d)", augment.Name, pick.Stacks)
		}
		if augment.Augment.Kind == "muda" && pick.Stacks > 1 {
			return 0, fmt.Sprintf("%s muda a magia, e mudança não se acumula", augment.Name)
		}
		if bar := book.PowerAugmentBar(pick.PowerID, fromBook); bar != "" {
			return 0, fmt.Sprintf("%s nesta magia: %s", augment.Name, bar)
		}
		// "Mudanças na mesma característica da magia nunca se acumulam" (p171).
		if pick.PowerID == book.QuickenedSpellPower && CastsAsRitual(dto) {
			return 0, fmt.Sprintf("o ritual e %s mudam a execução, e mudança não se acumula", augment.Name)
		}
		for _, spellPick := range spellPicks {
			own := spell.Augments[spellPick.AugmentIndex]
			if own.Exclusive {
				return 0, fmt.Sprintf(
					"o aprimoramento %d não pode ser usado em conjunto com outros: escolha só ele", spellPick.AugmentIndex)
			}
			// "Mudanças na mesma característica da magia nunca se acumulam" (p171).
			if pick.PowerID == book.QuickenedSpellPower && spellPick.AugmentIndex < len(fromBook.Augments) &&
				book.AugmentChangesExecution(fromBook.Augments[spellPick.AugmentIndex].Description) {
				return 0, fmt.Sprintf("%s e o aprimoramento %d mudam a execução, e mudança não se acumula",
					augment.Name, spellPick.AugmentIndex)
			}
		}
		total += augment.Augment.PmCost * pick.Stacks
	}
	return total, ""
}

// RitualSwitch é o interruptor do Celebrar Ritual (p131).
var RitualSwitch = engine.FlagGroupID("ritual")

// CastsAsRitual diz se a próxima magia desta ficha é ritual.
func CastsAsRitual(dto sheet.CharacterDTO) bool {
	return sheet.ToStringSet(dto.Conditionals)[RitualSwitch]
}

// CastExecution é a execução que esta conjuração cobra do turno: a da magia, a ação livre da Magia
// Acelerada, ou a hora do ritual (p131).
//
// Quem chama pergunta ANTES de conjurar, e um pedido sem o poder na ficha só passa
// por aqui para ser recusado pelo `Cast` logo em seguida, sem gastar nada.
func CastExecution(execution string, ritual bool, powerAugments []sheet.PowerAugmentPick) string {
	if ritual {
		return string(engine.ActionLong)
	}
	for _, pick := range powerAugments {
		if pick.PowerID == book.QuickenedSpellPower {
			return string(engine.ActionFree)
		}
	}
	return execution
}
