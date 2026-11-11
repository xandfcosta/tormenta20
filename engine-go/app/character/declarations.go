package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O USAR DE UMA DECLARAÇÃO liga a flag dela, e o "encerrar" da aba Efeitos a desliga (ver
// `book.DeclaredFlag`). Não há PM, uso por cena nem efeito ativo a gravar: o que dura até o próximo
// turno é o condicional da flag, e quem o desliga na volta da vez é a fila.

// OffersTheDeclaration diz se a ficha tem o que declarar com esta flag. Com o Estilo de Duas Armas,
// a Ambidestria de classe só tira o -2 (p125): o `Unless` do catálogo cala a declaração dela, e ela
// fica passiva. A tela e o caso de uso perguntam AQUI, para a tela não oferecer o que o servidor
// recusa.
//
//	character.OffersTheDeclaration(dto, "ataque-ambidestro") // false com o estilo na ficha
func OffersTheDeclaration(dto sheet.CharacterDTO, flag string) bool {
	if dto.Ruleset == nil {
		return false
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return false
	}
	offered := engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional
	return slices.ContainsFunc(offered, func(c engine.ConditionalEffect) bool { return c.Flag == flag })
}

// declare liga a flag que o poder usado declara.
func (p Plays) declare(ctx context.Context, dto sheet.CharacterDTO, spec book.Activation, flag string) error {
	if !OffersTheDeclaration(dto, flag) {
		return fmt.Errorf("%s: a ficha não tem o que declarar com ele (com o Estilo de Duas Armas, a Ambidestria só tira o -2, p125)", spec.Name)
	}
	if err := p.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
		Characterid: dto.ID, Conditionalid: engine.FlagGroupID(flag),
	}); err != nil {
		return fmt.Errorf("declarar %s na ficha %d: %w", spec.Name, dto.ID, err)
	}
	return nil
}

// EndDeclaration desfaz o uso de uma declaração antes de a vez voltar — o uso por engano.
//
// SÓ DECLARAÇÃO sai por aqui: a postura tem a própria saída, que devolve o que ela concedeu, e o
// interruptor (o Ataque Poderoso) se desliga pela chave dele.
//
//	plays.EndDeclaration(ctx, dto, "estilo-de-duas-armas")
func (p Plays) EndDeclaration(ctx context.Context, dto sheet.CharacterDTO, flag string) error {
	if !book.IsDeclaredFlag(flag) {
		return fmt.Errorf("%q não é uma declaração de usar; esperava a flag de um poder como o Estilo de Duas Armas", flag)
	}
	if err := p.queries.RemoveCharacterConditional(ctx, sqlcgen.RemoveCharacterConditionalParams{
		Characterid: dto.ID, Conditionalid: engine.FlagGroupID(flag),
	}); err != nil {
		return fmt.Errorf("encerrar %q na ficha %d: %w", flag, dto.ID, err)
	}
	return nil
}
