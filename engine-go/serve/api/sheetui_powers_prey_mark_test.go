package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/app/character"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A MARCA DA PRESA (p50, ALE-487).
//
//	"Você pode gastar uma ação de movimento e 1 PM para analisar uma criatura em
//	alcance curto. Até o fim da cena, você recebe +1d4 nas rolagens de dano
//	contra essa criatura. A cada quatro níveis, você pode gastar +1 PM para
//	aumentar o bônus de dano (veja a tabela da classe)."
//
// A tabela é 1d4, 1d8, 1d12, 2d8, 2d10: o degrau ESCOLHE a linha, não soma um
// dado. O app não guarda QUEM foi marcado (decisão do dono): o gesto cobra e
// grava o dado, e a ficha ganha o interruptor "contra a criatura marcada", que
// o jogador liga no golpe contra ela.
func TestThePreyMarkBuysTheTableDieAndAddsItAgainstTheMarkedCreature(t *testing.T) {
	f, id := hunterFixture(t, 5)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", `{"stance_degrees":1}`); refusal != "" {
		t.Fatalf("marcar com 1 degrau no 5º nível foi recusado: %q", refusal)
	}
	if after := pm(t, f, id); before-after != 2 {
		t.Errorf("a Marca com 1 degrau cobrou %d PM, e a p50 diz 1 + 1", before-after)
	}

	// SEM O INTERRUPTOR, nada: o dado é contra a criatura marcada, e não contra
	// todo mundo.
	if extra := extraDiceOnTheFirstCard(t, f, id); len(extra) != 0 {
		t.Fatalf("o dado da Marca desceu sem o interruptor: %+v", extra)
	}
	key := preyMarkSwitch(t, f, id)
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao?tab=abilities", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target,
		fmt.Sprintf(`{"conditional":%q}`, key)).Body.String()); refusal != "" {
		t.Fatalf("ligar %q foi recusado: %q", key, refusal)
	}
	if extra := extraDiceOnTheFirstCard(t, f, id); len(extra) != 1 || extra[0].Dice != "1d8" {
		t.Errorf("com 1 degrau a Marca pôs %+v, e a tabela do caçador dá 1d8 no 5º", extra)
	}

	// MARCAR DE NOVO reescreve o dado — o livro não empilha a fonte consigo.
	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("marcar de novo sem degrau foi recusado: %q", refusal)
	}
	if extra := extraDiceOnTheFirstCard(t, f, id); len(extra) != 1 || extra[0].Dice != "1d4" {
		t.Errorf("a segunda Marca, sem degrau, deixou %+v; want só 1d4", extra)
	}

	// O TETO É DO NÍVEL: o 5º abre um degrau, e dois são recusados sem cobrar.
	held := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", `{"stance_degrees":2}`); refusal == "" {
		t.Error("2 degraus no 5º nível passaram — o segundo é do 9º")
	}
	if after := pm(t, f, id); after != held {
		t.Errorf("a Marca recusada cobrou %d PM", held-after)
	}
}

// MARCAR CUSTA A AÇÃO DE MOVIMENTO (p50), e na vez ela sai do turno — o gesto
// instantâneo pergunta e cobra a ação como a postura já fazia.
func TestThePreyMarkSpendsTheMovementAction(t *testing.T) {
	f, id := hunterFixture(t, 1)
	f.charID = id
	startActionSceneOnTurnOf(t, f, "Caçadora")

	if refusal := powerCommand(t, f, id, "usa/class.cacador.marca-da-presa-1d4", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("marcar na vez foi recusado: %q", refusal)
	}
	scene := stateOf(t, f.s.sessions, f.sessionID).Scene
	if scene.MovementLeft {
		t.Error("marcar não gastou a ação de movimento")
	}
	if !scene.StandardLeft {
		t.Error("marcar comeu a ação padrão junto")
	}
}

// hunterFixture semeia um caçador com o poço cheio.
func hunterFixture(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id, err := f.s.sceneCore().Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Caçadora", Origin: "batedor", Level: level,
		Strength: 3, Dexterity: 3, Constitution: 2, Intelligence: 0, Wisdom: 2, Charisma: 0,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a caçadora: %v", err)
	}
	seedClasse(t, f.s, id, "Caçador", level)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent, pools.MpCurrent = pools.HpMax, pools.MpMax
		return pools, nil
	})
	seedPericia(t, f.s, id, "Luta", "strength", true)
	seedMember(t, f.s, f.campaignID, id)
	return f, id
}

// extraDiceOnTheFirstCard são os dados extras da primeira carta de arma, lidos
// pelo caminho de produção.
func extraDiceOnTheFirstCard(t *testing.T, f sceneFixture, id int64) []engine.ExtraDamage {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	cards := dto.Ruleset.ComputeWeaponCards(ec, sheet.ToStringSet(dto.Conditionals))
	if len(cards) == 0 {
		t.Fatal("a ficha não tem carta de arma")
	}
	return cards[0].ExtraDamage
}

// preyMarkSwitch é a chave do interruptor "contra a criatura marcada" que a
// ficha oferece.
func preyMarkSwitch(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	dto, ec := loadedCharacter(t, f, id)
	groups := character.SituationalGroupsOf(engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Conditional)
	labels := []string{}
	for _, g := range groups {
		for _, m := range g.Members {
			if strings.Contains(m.Note, "criatura marcada") {
				return g.Key
			}
			labels = append(labels, m.Note)
		}
	}
	t.Fatalf("a ficha não oferece o interruptor da criatura marcada; oferece %q", labels)
	return ""
}

func loadedCharacter(t *testing.T, f sceneFixture, id int64) (sheet.CharacterDTO, engine.Character) {
	t.Helper()
	ctx := context.Background()
	row, err := f.s.sceneCore().Queries().GetCharacter(ctx, id)
	if err != nil {
		t.Fatalf("ler o personagem %d: %v", id, err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(ctx, row)
	if err != nil {
		t.Fatalf("carregar o personagem %d: %v", id, err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		t.Fatalf("converter o personagem %d: %v", id, err)
	}
	return dto, ec
}

// O DEGRAU QUE SOBROU DE OUTRO DIÁLOGO NÃO RECUSA o poder que não vende degrau.
// O sinal é o mesmo da postura e vive na página inteira; lido num poder sem
// escala, ele viraria "o nível permite até zero degraus" num gesto legítimo.
func TestAPowerWithoutDegreesIgnoresALeftoverDegreeSignal(t *testing.T) {
	f, id := fighterFixture(t)
	armor := itemSemeia(t, f, id, "armadura-couro", "Armadura de couro", "vested")
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), armor, []string{"encanto-caustica"},
	); err != nil {
		t.Fatalf("encantar a armadura: %v", err)
	}
	if refusal := powerCommand(t, f, id, "usa/item.encanto-caustica", `{"stance_degrees":3}`); refusal != "" {
		t.Errorf("a Cáustica foi recusada por um degrau que ela não vende: %q", refusal)
	}
}
