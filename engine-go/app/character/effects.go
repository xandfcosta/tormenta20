package character

import (
	"context"
	"fmt"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// OS GESTOS DA ABA EFEITOS e o toque nos vitais.
//
// Os quatro aqui têm uma escrita cada. Entrar e sair de uma POSTURA são de outra
// natureza — várias escritas que só fazem sentido juntas — e por isso não moram
// neste arquivo.

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
func (p Plays) ToggleSituational(ctx context.Context, dto sheet.CharacterDTO, key string) error {
	if key == "" {
		return fmt.Errorf("o gesto não disse qual efeito situacional alternar")
	}
	if err := p.refuseUnofferedSituational(dto, key); err != nil {
		return err
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
		if err := p.queries.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: characterID, Conditionalid: key,
		}); err != nil {
			return fmt.Errorf("desligar o condicional %q: %w", key, err)
		}
		return nil
	}
	if err := p.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
		Characterid: characterID, Conditionalid: key,
	}); err != nil {
		return fmt.Errorf("ligar o condicional %q: %w", key, err)
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
