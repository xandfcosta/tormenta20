package character

import (
	"context"
	"fmt"

	"t20engine/domain/catalog"
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
	ctx context.Context, row sqlcgen.Character, qual string, passo int,
) error {
	if qual != "pv" && qual != "pm" {
		return fmt.Errorf("vital %q não existe: são 'pv' e 'pm'", qual)
	}
	_, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, row,
		func(pocos sheet.Pools) (sheet.Pools, error) {
			if qual == "pv" {
				pocos.HpCurrent += int64(passo)
			} else {
				pocos.MpCurrent += int64(passo)
			}
			return pocos, nil
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
	ctx context.Context, row sqlcgen.Character, condicao string,
) error {
	if !catalog.IsCondition(condicao) {
		return fmt.Errorf("%q não é uma condição do livro", condicao)
	}
	depois := []string{}
	tinha := false
	for _, c := range sheet.UnmarshalStrings(row.Activeconditions) {
		if c == condicao {
			tinha = true
			continue
		}
		depois = append(depois, c)
	}
	if !tinha {
		depois = append(depois, condicao)
	}
	if err := p.queries.UpdateConditions(ctx, sqlcgen.UpdateConditionsParams{
		ActiveConditions: sheet.MarshalStrings(&depois),
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
func (p Plays) EndAppliedEffect(ctx context.Context, characterID, efeitoID int64) error {
	meta, err := p.queries.GetActiveEffectMeta(ctx, efeitoID)
	if err != nil || meta.Characterid != characterID {
		return fmt.Errorf("o efeito %d não é desta ficha", efeitoID)
	}
	if err := p.queries.DeleteEffectByID(ctx, efeitoID); err != nil {
		return fmt.Errorf("encerrar o efeito %d: %w", efeitoID, err)
	}
	return nil
}

// ToggleSituational liga ou desliga um condicional de contexto.
func (p Plays) ToggleSituational(ctx context.Context, characterID int64, chave string) error {
	if chave == "" {
		return fmt.Errorf("o gesto não disse qual efeito situacional alternar")
	}
	atuais, err := p.queries.ListCharacterConditionals(ctx, characterID)
	if err != nil {
		return fmt.Errorf("ler os condicionais da ficha %d: %w", characterID, err)
	}
	for _, c := range atuais {
		if c != chave {
			continue
		}
		if err := p.queries.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
			Characterid: characterID, Conditionalid: chave,
		}); err != nil {
			return fmt.Errorf("desligar o condicional %q: %w", chave, err)
		}
		return nil
	}
	if err := p.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
		Characterid: characterID, Conditionalid: chave,
	}); err != nil {
		return fmt.Errorf("ligar o condicional %q: %w", chave, err)
	}
	return nil
}
