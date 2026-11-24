package character

import (
	"context"
	"errors"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS TRÊS ESCRITAS DO GRIMÓRIO — aprender, esquecer e alternar a memorização.
//
// Elas desceram da cena na ALE-350, e a do meio é a que explica por quê: o
// `ToggleMemorized` LÊ o grimório, DECIDE o estado novo e grava. As três coisas
// que o guia usa para reconhecer um caso de uso, num gesto de um clique.

// LearnSpell põe uma magia do catálogo no grimório.
//
// O livro é conferido AQUI e não por quem chama: uma magia que não existe seria
// uma linha órfã no grimório, que a ficha desenha como um espaço em branco. E o
// círculo também: aprende-se a magia "de um tipo e círculo que possa lançar com
// aquela classe" (p170), e a acima dele não se aprende (o conjurador, `sheet.CasterOf`). A magia de fora da
// lista das classes se aprende, porque um poder pode ensiná-la; a ficha só avisa.
func (p Plays) LearnSpell(ctx context.Context, dto sheet.CharacterDTO, catalogSpellID string) error {
	if _, known := book.SpellByID(catalogSpellID); !known {
		return fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	if refusal := sheet.CasterOf(dto).Spell(catalogSpellID).AboveTheCircle; refusal != "" {
		return errors.New(refusal)
	}
	if _, err := p.queries.CreateSpell(ctx, sqlcgen.CreateSpellParams{
		Characterid: dto.ID, Catalogspellid: catalogSpellID,
		Prepared: 0, Learnedat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("aprender %q na ficha %d: %w", catalogSpellID, dto.ID, err)
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

// ToggleMemorized memoriza ou desmemoriza uma magia do mago (p37).
//
// Ela recebe a MAGIA e não o estado desejado: mandar "memorizada" perde para o
// clique repetido e para a segunda aba aberta no mesmo personagem — quem sabe o
// que está lá é o servidor, e é ele que inverte. Memorizar pergunta ao conjurador
// (`MemorizeRefusal`): só o mago memoriza, e até metade do que conhece.
// Desmemorizar sempre se pode.
func (p Plays) ToggleMemorized(ctx context.Context, dto sheet.CharacterDTO, catalogSpellID string) error {
	all, err := p.queries.ListSpellsByCharacter(ctx, dto.ID)
	if err != nil {
		return fmt.Errorf("ler o grimório da ficha %d: %w", dto.ID, err)
	}
	for _, m := range all {
		if m.Catalogspellid != catalogSpellID {
			continue
		}
		after := int64(0)
		if m.Prepared == 0 {
			if refusal := sheet.CasterOf(dto).MemorizeRefusal(catalogSpellID); refusal != "" {
				return errors.New(refusal)
			}
			after = 1
		}
		if _, err := p.queries.SetSpellPreparedByCatalog(ctx, sqlcgen.SetSpellPreparedByCatalogParams{
			Prepared: after, CharacterId: dto.ID, CatalogSpellId: catalogSpellID,
		}); err != nil {
			return fmt.Errorf("gravar a memorização de %q: %w", catalogSpellID, err)
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
	augments []sheet.AugmentPick, powerAugments []sheet.PowerAugmentPick, fromTheTable []int,
) (SpellCast, error) {
	onSheet := sheet.CasterOf(dto).Spell(catalogSpellID)
	if !onSheet.Known {
		return SpellCast{}, fmt.Errorf("a magia %q não existe no livro", catalogSpellID)
	}
	spell := onSheet.Spell
	if findSpell(dto.Spells, catalogSpellID) == nil {
		return SpellCast{}, fmt.Errorf("%q não está no grimório desta ficha", catalogSpellID)
	}
	// A RECUSA QUE A FICHA RESPONDE SOZINHA é do conjurador, a mesma que tranca o
	// Conjurar nas abas: a magia acima do círculo — a gravada antes da regra fica no
	// grimório, e quem joga a esquece e aprende outra —, a do mago não memorizada e a
	// segunda sustentada, que depende do aprimoramento escolhido (p227).
	if refusal := onSheet.CastRefusal(augments); refusal != "" {
		return SpellCast{}, errors.New(refusal)
	}
	// O APRIMORAMENTO é do conjurador, o da magia e o poder que é aprimoramento (p131): o preço e a recusa do
	// livro (p170-171), a mesma resposta que tranca o cadeado da folha.
	augmentPm, augErr := onSheet.AugmentPrice(augments, powerAugments)
	if augErr != "" {
		return SpellCast{}, fmt.Errorf("%s", augErr)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return SpellCast{}, err
	}
	if !engine.CastsInTheShape(ec) {
		return SpellCast{}, fmt.Errorf("%s", ShapeSilence)
	}
	basePm := sheet.SpellBasePmCost[spell.Circle]
	// O TRUQUE zera a conjuração INTEIRA e não só a parte dele: "reduz seu custo
	// em PM para zero" (p171). Tratado como um aprimoramento de +0 PM, o custo
	// base ficava de pé e a versão mais simples da magia saía pelo preço da
	// normal (ALE-339).
	//
	// O `augmentPm` também vai a zero, e não por precaução: o truque é
	// `Exclusive`, então o `AugmentPrice` já recusou qualquer companhia e não
	// há segunda parcela — zerar aqui diz isso em voz alta em vez de depender de
	// uma invariante que mora noutra função.
	if onSheet.ChoosesTheCantrip(augments) {
		basePm, augmentPm = 0, 0
	}
	// OS CONDICIONAIS LIGADOS entram nas três contas. Aqui iam três conjuntos
	// VAZIOS, e o custo, o mínimo e o teto saíam de um personagem que não é o
	// que está conjurando (ALE-357).
	active := sheet.ToStringSet(dto.Conditionals)
	// QUEM LANÇA a magia é a pergunta das duas contas, e não a lista do verbete: o bardo de evocação lança a
	// Bola de Fogo, que lista só o Arcanista (p44).
	at := onSheet.At()
	totalPm := dto.Ruleset.SpellPmCostFor(ec, basePm, augmentPm, active, at)
	minPm := dto.Ruleset.SpellPmCostFor(ec, basePm, 0, active, at)
	limit := dto.Ruleset.SpellPmLimitFor(ec, active, onSheet.Casting, at)
	if spell.Circle > 0 && totalPm > limit && totalPm > minPm {
		return SpellCast{}, fmt.Errorf("o custo de %d PM passa do limite de %d por magia", totalPm, limit)
	}
	// O MANA DISPONÍVEL inclui a poça temporária, e as duas metades da p106
	// andam juntas: se o gasto drena a poça primeiro, a poça CONTA para decidir
	// se o gasto cabe. Com `dto.MpCurrent` aqui, o bardo de poço vazio e poça
	// cheia seria recusado — a poça existiria só para ser ignorada.
	if available := dto.ManaAvailable(); int64(totalPm) > available {
		return SpellCast{}, fmt.Errorf("faltam PM: a magia custa %d e restam %d", totalPm, available)
	}
	// O DADO é rolado depois das recusas e antes da cobrança: o total da mesa que os dados não tiram é recusado
	// sem gastar nada.
	cast, err := p.rollTheCast(dto, onSheet, augments, fromTheTable)
	if err != nil {
		return SpellCast{}, err
	}
	ritual, sustains := active[sheet.RitualSwitch], onSheet.Sustains(augments)
	// O DADO DA VEZ rola de novo enquanto a magia dura, com os aprimoramentos desta conjuração (ver GLOSSARY,
	// **dado da vez**): ela abre a linha em vigor mesmo sem manutenção (a Poeira da Podridão dura a cena), e os
	// grava junto.
	turns := spell.TurnRoll != nil
	// AS AÇÕES DA MAGIA também rolam depois com os aprimoramentos desta conjuração (ver GLOSSARY, **ação da
	// magia**), e os dardos da Lança Ígnea abrem a linha de cena com as cargas contadas.
	acts, charges := onSheet.Actions(augments) != nil, onSheet.Charges(augments)
	if !ritual && !sustains && !turns && !acts {
		if err := sheet.SpendLoadedMana(ctx, p.queries, &dto, totalPm); err != nil {
			return SpellCast{}, err
		}
		return cast, nil
	}
	// O RITUAL e A SUSTENTADA valem para esta conjuração: é aqui e não no gesto porque ela pode ser recusada
	// acima, e a recusa não gasta o interruptor nem abre a manutenção. E saem na mesma transação que o PM: se
	// a segunda escrita falhasse com o PM já cobrado, a tela diria erro, e tentar de novo cobraria outra vez.
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return SpellCast{}, fmt.Errorf("abrir a transação de %s: %w", spell.Name, err)
	}
	defer func() { _ = tx.Rollback() }()
	q := p.queries.WithTx(tx)
	if err := sheet.SpendLoadedMana(ctx, q, &dto, totalPm); err != nil {
		return SpellCast{}, err
	}
	if ritual {
		if err := q.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: dto.ID, Conditionalid: sheet.RitualSwitch,
		}); err != nil {
			return SpellCast{}, fmt.Errorf("desligar o ritual da ficha %d: %w", dto.ID, err)
		}
	}
	if sustains || turns || acts {
		scope := engine.Duration{Kind: engine.DurationSustained}.Stored()
		if !sustains {
			scope = engine.Duration{Kind: engine.DurationScene}.Stored()
		}
		effectID, err := writeSpellEffect(ctx, q, dto.ID, catalogSpellID, spellEffectWrite{
			Scope: scope, Cast: true, Augments: augments,
		})
		if err != nil {
			return SpellCast{}, err
		}
		if charges > 0 {
			if err := q.SetEffectCharges(ctx, sqlcgen.SetEffectChargesParams{Effectid: effectID, Remaining: int64(charges)}); err != nil {
				return SpellCast{}, fmt.Errorf("gravar as cargas de %s: %w", spell.Name, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return SpellCast{}, err
	}
	return cast, nil
}

func findSpell(spells []sheet.SpellDTO, catalogSpellID string) *sheet.SpellDTO {
	for i := range spells {
		if spells[i].CatalogSpellID == catalogSpellID {
			return &spells[i]
		}
	}
	return nil
}
