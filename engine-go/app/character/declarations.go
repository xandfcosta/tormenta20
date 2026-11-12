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

// O USAR DE UMA DECLARAÇÃO — ou o gesto da Forma Ágil, que não tem poder próprio — liga a flag dela, e o
// "encerrar" da aba Efeitos a desliga (ver `book.Declaration`). Não há PM, uso por cena nem efeito ativo a gravar: o que dura até o próximo
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
	// A FORMA ÁGIL é a única que tem as duas armas naturais (p63). O -2 dela mora no poder da Forma
	// Selvagem, que todo druida com a forma tem, e por isso o condicional é oferecido em qualquer forma:
	// quem diz se há o que declarar é a forma em que ele está.
	if flag == engine.WildAgileBothFlag && (ec.WildShape == nil || ec.WildShape.Form != engine.WildAgile) {
		return false
	}
	offered := dto.Ruleset.EffectsOf(ec, sheet.ToStringSet(dto.Conditionals)).Conditional
	return slices.ContainsFunc(offered, func(c engine.ConditionalEffect) bool { return c.Flag == flag })
}

// Declare liga a declaração pela flag dela — o gesto da Forma Ágil, que não tem poder próprio para ter
// um Usar (`book.DeclarationFor`).
//
//	plays.Declare(ctx, dto, engine.WildAgileBothFlag)
func (p Plays) Declare(ctx context.Context, dto sheet.CharacterDTO, flag string) error {
	d, declared := book.DeclarationFor(flag)
	if !declared {
		return fmt.Errorf("%q não é uma declaração; esperava a flag de uma, como %q", flag, engine.WildAgileBothFlag)
	}
	return p.declare(ctx, dto, d.Name, flag)
}

// declare liga a flag da declaração.
func (p Plays) declare(ctx context.Context, dto sheet.CharacterDTO, name, flag string) error {
	if !OffersTheDeclaration(dto, flag) {
		return fmt.Errorf("%s: a ficha não tem o que declarar agora (%s)", name, whyNothingToDeclare(flag))
	}
	if err := p.queries.AddCharacterConditional(ctx, sqlcgen.AddCharacterConditionalParams{
		Characterid: dto.ID, Conditionalid: engine.FlagGroupID(flag),
	}); err != nil {
		return fmt.Errorf("declarar %s na ficha %d: %w", name, dto.ID, err)
	}
	return nil
}

// whyNothingToDeclare é o caso que a recusa explica.
func whyNothingToDeclare(flag string) string {
	if flag == engine.WildAgileBothFlag {
		return "só a Forma Ágil tem as duas armas naturais, p63"
	}
	return "com o Estilo de Duas Armas, a Ambidestria só tira o -2, p125"
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
