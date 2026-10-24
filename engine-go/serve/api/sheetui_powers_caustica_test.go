package api

import (
	"context"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// A CÁUSTICA PÕE +1d4 DE ÁCIDO NOS ATAQUES DO PORTADOR POR UMA CENA (p338, ALE-460).
//
//	"Você recebe redução de ácido 10 e pode gastar uma ação de movimento e 2 PM
//	para fazer o item gotejar ácido. Se fizer isso, seus ataques causam +1d4 de
//	dano de ácido até o fim da cena."
//
// A redução já entrava; o +1d4 era a metade calada da ALE-459. O trilho é o
// mesmo do En Garde (p48) e da Armadura de Allihanna (p21): o gesto grava um
// efeito ativo de cena, e o efeito volta ao MESMO coletor dos itens.
//
// O caso é de INTEGRAÇÃO de propósito — o gesto atravessa o `UsePower`, a
// gravação, a releitura da ficha e o coletor —, e é o caminho inteiro que
// precisa estar ligado para o dado chegar à carta. Cada pedaço já tinha teste
// isolado; o que faltava provar era a COMPOSIÇÃO.
func TestTheCausticGestureAddsAcidDiceToTheBearersAttacks(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	armor := itemSemeia(t, f, id, "armadura-couro", "Armadura de couro", "vested")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), armor, []string{"encanto-caustica"},
	); err != nil {
		t.Fatalf("encantar a armadura: %v", err)
	}

	// SEM O GESTO, nada: é o controle, e é ele que impede o caso de passar
	// sobre um dado que valesse sempre.
	if extra := acidOnTheSword(t, f, id); len(extra) != 0 {
		t.Fatalf("o ácido desceu ANTES do gesto: %+v", extra)
	}

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/item.encanto-caustica", ""); refusal != "" {
		t.Fatalf("usar a Cáustica foi recusado: %q", refusal)
	}
	// O PREÇO DO LIVRO: 2 PM. Em delta, pela razão do caso da Dançarina.
	if after := pm(t, f, id); before-after != 2 {
		t.Errorf("o gesto cobrou %d PM (%d → %d), e a p338 diz 2", before-after, before, after)
	}

	extra := acidOnTheSword(t, f, id)
	if len(extra) != 1 {
		t.Fatalf("depois do gesto a carta da espada traz %d parcelas de dado, quer 1: %+v",
			len(extra), extra)
	}
	if extra[0].Dice != "1d4" || extra[0].Type != "ácido" {
		t.Errorf("o gesto pôs %q de %q, e a p338 dá 1d4 de ácido", extra[0].Dice, extra[0].Type)
	}
}

// acidOnTheSword lê a carta da arma empunhada pelo caminho de produção.
func acidOnTheSword(t *testing.T, f sceneFixture, id int64) []engine.ExtraDamage {
	t.Helper()
	row, err := f.s.sceneCore().Queries().GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler o personagem: %v", err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar o personagem: %v", err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		t.Fatalf("converter para o motor: %v", err)
	}
	cards := dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals))
	// A espada e, depois dela, a carta do ataque desarmado, que é de todo mundo
	// (p143, ALE-486).
	if len(cards) != 2 || cards[1].Name != engine.UnarmedName {
		t.Fatalf("esperava a espada e o desarmado, e vieram %d cartas", len(cards))
	}
	return cards[0].ExtraDamage
}
