package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// OS GESTOS DA ABA EFEITOS e o toque nos vitais.
//
// Quase todos têm uma escrita cada. A exceção é o situacional cujo poder VENDE
// degrau (p53): ele liga a flag e grava o bônus pago no mesmo contorno, porque
// metade do gesto é uma ficha que promete +10 e soma +2.
//
// Entrar e sair de uma POSTURA continuam de outra natureza — PM cobrado,
// registro de pagamento, posturas irmãs derrubadas — e moram no `stance.go`.

// TouchVital move PV ou PM em um passo, preso entre zero e o máximo.
//
// O TETO é o máximo do personagem: curar além dele não é PV temporário, que é
// outra regra e tem dono no motor. Quem prende é o funil, para os dois vitais e
// para todos os gestos — aqui só se diz QUAL vital anda e quanto.
func (p Plays) TouchVital(
	ctx context.Context, row sqlcgen.Character, which string, step int,
) error {
	if which != "pv" && which != "pm" {
		return fmt.Errorf("vital %q não existe: são 'pv' e 'pm'", which)
	}
	_, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, row,
		func(pools sheet.Pools) (sheet.Pools, error) {
			if which == "pv" {
				pools.HpCurrent += int64(step)
			} else {
				pools.MpCurrent += int64(step)
			}
			return pools, nil
		})
	return err
}

// ToggleBookCondition liga ou desliga UMA condição do livro (p394-395).
//
// Ela recebe a CONDIÇÃO e não a lista, pela razão de sempre: mandar a lista
// perde para o clique repetido e para a segunda aba aberta.
//
// Quem AVISA a mesa é quem chama, e de propósito: o motor deriva Defesa e
// perícias da condição, então uma condição aplicada sem aviso faz o jogador e o
// mestre verem números diferentes do mesmo personagem. O aviso depende do
// barramento do processo, que é do hospedeiro — este método devolve, e a cena
// avisa DEPOIS, nunca antes: avisar sobre algo que ainda pode falhar faria a
// mesa buscar o estado velho e acreditar nele.
func (p Plays) ToggleBookCondition(
	ctx context.Context, row sqlcgen.Character, condition string,
) error {
	if !catalog.IsCondition(condition) {
		return fmt.Errorf("%q não é uma condição do livro", condition)
	}
	after := []string{}
	had := false
	for _, c := range sheet.UnmarshalStrings(row.Activeconditions) {
		if c == condition {
			had = true
			continue
		}
		after = append(after, c)
	}
	if !had {
		after = append(after, condition)
	}
	if err := p.queries.UpdateConditions(ctx, sqlcgen.UpdateConditionsParams{
		ActiveConditions: sheet.MarshalStrings(&after),
		UpdatedAt:        dbvalue.NowISO(),
		ID:               row.ID,
	}); err != nil {
		return fmt.Errorf("gravar as condições da ficha %d: %w", row.ID, err)
	}
	return nil
}

// EndAppliedEffect encerra um efeito em curso.
//
// A POSSE é conferida AQUI, e a consulta não a confere por nós: o
// `DeleteEffectByID` apaga por id e mais nada, então sem esta leitura um pedido
// montado à mão encerraria o efeito de OUTRO personagem. Ela desceu junto com a
// escrita de propósito — separá-las daria ao `app/` um método que apaga efeito
// de qualquer ficha.
func (p Plays) EndAppliedEffect(ctx context.Context, characterID, effectID int64) error {
	meta, err := p.queries.GetActiveEffectMeta(ctx, effectID)
	if err != nil || meta.Characterid != characterID {
		return fmt.Errorf("o efeito %d não é desta ficha", effectID)
	}
	if err := p.queries.DeleteEffectByID(ctx, effectID); err != nil {
		return fmt.Errorf("encerrar o efeito %d: %w", effectID, err)
	}
	return nil
}

// ToggleSituational liga ou desliga um condicional de contexto.
//
// # A CHAVE É CONFERIDA CONTRA O QUE A FICHA OFERECE
//
// Ela vem crua de um sinal do cliente, e a tela esconder um interruptor não é
// garantia nenhuma — a fronteira é o servidor. Sem esta conferência um jogador
// entrava na Fúria postando a chave da postura: +3 em ataque e dano, PM intacto,
// nenhuma linha de postura. Jogador não muda REGRA; ele liga o que a ficha dele
// já oferece (ALE-387).
//
// O conjunto é de PERMITIDOS, e não uma lista de proibidos com as posturas
// dentro: uma lista de proibidos subconta em silêncio, e a primeira espécie de
// chave que alguém inventasse passaria por ela.
// # E O DEGRAU VIAJA COM ELE
//
// Um situacional cujo poder VENDE degrau (p53) carrega quantos foram pagos: o
// Baluarte de um cavaleiro de 20º vale +2 ou até +10, e quem escolhe é quem
// gastou o PM. Ligar com N degraus grava o bônus total num efeito ativo com a
// chave do poder; desligar leva o efeito junto. O interruptor é a DURAÇÃO —
// "até o início do seu próximo turno" não tem relógio neste app, e não ter
// relógio é o mesmo de "com torcida a seu favor".
func (p Plays) ToggleSituational(
	ctx context.Context, dto sheet.CharacterDTO, key string, steps int,
) error {
	if key == "" {
		return fmt.Errorf("o gesto não disse qual efeito situacional alternar")
	}
	if err := p.refuseUnofferedSituational(dto, key); err != nil {
		return err
	}
	offer, sold := degreeOfferFor(key)
	if sold {
		if err := refuseDegreesAboveTheCeiling(dto, offer, steps); err != nil {
			return err
		}
	}
	characterID := dto.ID
	current, err := p.queries.ListCharacterConditionals(ctx, characterID)
	if err != nil {
		return fmt.Errorf("ler os condicionais da ficha %d: %w", characterID, err)
	}
	for _, c := range current {
		if c != key {
			continue
		}
		return p.turnTheSituationalOff(ctx, characterID, key, offer, sold)
	}
	return p.turnTheSituationalOn(ctx, characterID, key, offer, sold, steps)
}

// SetSituationalDegrees liga o situacional com os degraus escolhidos, sem
// alternar.
//
// Separada do `ToggleSituational` porque o gesto é outro: o interruptor diz
// "estou nisto ou não", e o seletor diz "paguei tantos PM". Escolher um degrau
// num Baluarte já ligado tem de mudar o número, e não apagá-lo.
func (p Plays) SetSituationalDegrees(
	ctx context.Context, dto sheet.CharacterDTO, key string, steps int,
) error {
	if key == "" {
		return fmt.Errorf("o gesto não disse de qual situacional são os degraus")
	}
	if err := p.refuseUnofferedSituational(dto, key); err != nil {
		return err
	}
	offer, sold := degreeOfferFor(key)
	if !sold {
		return fmt.Errorf("este efeito situacional não vende degrau")
	}
	if err := refuseDegreesAboveTheCeiling(dto, offer, steps); err != nil {
		return err
	}
	return p.turnTheSituationalOn(ctx, dto.ID, key, offer, true, steps)
}

// turnTheSituationalOn liga o condicional e, quando o poder vende degrau, grava
// o que os degraus pagos valem.
//
// AS DUAS ESCRITAS NUM CONTORNO, ao contrário do situacional simples: o degrau
// é exatamente o que o interruptor prometeu, e um commit que ligasse a flag e
// perdesse o bônus daria um Baluarte de +2 a quem escolheu +10.
func (p Plays) turnTheSituationalOn(
	ctx context.Context, characterID int64, key string,
	offer book.DegreeOffer, sold bool, steps int,
) error {
	if !sold {
		if err := p.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
			Characterid: characterID, Conditionalid: key,
		}); err != nil {
			return fmt.Errorf("ligar o condicional %q: %w", key, err)
		}
		return nil
	}
	return p.inTx(ctx, "ligar o situacional "+key, func(q *sqlcgen.Queries) error {
		if err := q.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
			Characterid: characterID, Conditionalid: key,
		}); err != nil {
			return fmt.Errorf("ligar o condicional %q: %w", key, err)
		}
		// O DEGRAU ANTERIOR SAI PRIMEIRO, e é o que faz DESCER funcionar: o
		// efeito é gravado por chave do poder, então escolher menos degraus
		// reescreve a linha — mas escolher ZERO não escreve nada, e sem este
		// apagar o +10 de antes ficava na ficha com o seletor dizendo +2.
		// Visto na tela (ALE-454).
		if err := q.DeleteEffectsByCatalog(ctx, sqlcgen.DeleteEffectsByCatalogParams{
			Characterid: characterID, Catalogid: offer.PowerID,
		}); err != nil {
			return fmt.Errorf("tirar o degrau anterior de %q: %w", offer.PowerID, err)
		}
		return writeDegrees(ctx, q, characterID, offer, steps)
	})
}

// turnTheSituationalOff desliga o condicional e leva o degrau junto.
func (p Plays) turnTheSituationalOff(
	ctx context.Context, characterID int64, key string, offer book.DegreeOffer, sold bool,
) error {
	if !sold {
		if err := p.queries.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: characterID, Conditionalid: key,
		}); err != nil {
			return fmt.Errorf("desligar o condicional %q: %w", key, err)
		}
		return nil
	}
	return p.inTx(ctx, "desligar o situacional "+key, func(q *sqlcgen.Queries) error {
		if err := q.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: characterID, Conditionalid: key,
		}); err != nil {
			return fmt.Errorf("desligar o condicional %q: %w", key, err)
		}
		return q.DeleteEffectsByCatalog(ctx, sqlcgen.DeleteEffectsByCatalogParams{
			Characterid: characterID, Catalogid: offer.PowerID,
		})
	})
}

// writeDegrees grava o que os degraus pagos valem, num efeito ativo com a chave
// do PODER — a mesma chave e o mesmo caminho que a postura usa, pela mesma
// razão: o efeito ativo já é por onde passa "estado de jogo que vira
// modificador", e ele vem com a aba Efeitos de graça.
//
// ESCOPO CENA porque não há escopo mais curto que alguém apague: o SQL de
// expiração só conhece `scene` e `day`. Quem encerra o Baluarte é o
// interruptor, e o fim da cena é a rede embaixo dele.
func writeDegrees(
	ctx context.Context, q *sqlcgen.Queries, characterID int64,
	offer book.DegreeOffer, steps int,
) error {
	mods := book.DegreeModifiers(offer, steps)
	if len(mods) == 0 {
		return nil
	}
	blob, err := json.Marshal(mods)
	if err != nil {
		return fmt.Errorf("escrever os %d degraus de %q: %w", steps, offer.PowerID, err)
	}
	if _, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: "power", Catalogid: offer.PowerID, Scope: "scene",
		Modifiers: string(blob), Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("gravar os %d degraus de %q: %w", steps, offer.PowerID, err)
	}
	return nil
}

// degreeOfferFor acha a oferta de degrau pela chave do CONDICIONAL, que é a
// flag com o prefixo do grupo.
func degreeOfferFor(key string) (book.DegreeOffer, bool) {
	for flag, offer := range book.DegreesByFlag() {
		if engine.FlagGroupID(flag) == key {
			return offer, true
		}
	}
	return book.DegreeOffer{}, false
}

// refuseDegreesAboveTheCeiling cobra o teto que o NÍVEL NA CLASSE autoriza.
//
// A fronteira é o servidor pela mesma razão da recusa de postura (ALE-423): a
// tela desenhar só até o teto não é garantia nenhuma, e um cavaleiro de 1º
// postando `steps: 4` levaria +10 sem ter nível para comprar um degrau.
func refuseDegreesAboveTheCeiling(dto sheet.CharacterDTO, offer book.DegreeOffer, steps int) error {
	if steps < 0 {
		return fmt.Errorf("%s: %d degraus não é um número de degraus", offer.PowerName, steps)
	}
	if max := offer.Degrees.MaxDegrees(ClassPowerLevel(dto, offer.PowerID)); steps > max {
		return fmt.Errorf("%s: o nível permite comprar até %d degraus, e o gesto pediu %d",
			offer.PowerName, max, steps)
	}
	return nil
}

// refuseUnofferedSituational recusa a chave que a ficha não oferece.
func (p Plays) refuseUnofferedSituational(dto sheet.CharacterDTO, key string) error {
	if dto.Ruleset == nil {
		return fmt.Errorf("a ficha %d chegou sem o mundo dela; não dá para dizer o que ela oferece", dto.ID)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return fmt.Errorf("montar o personagem do motor (%d): %w", dto.ID, err)
	}
	offered := engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional
	for _, g := range SituationalGroupsOf(offered) {
		if g.Key == key {
			return nil
		}
	}
	return fmt.Errorf("este efeito situacional não é oferecido por esta ficha")
}
