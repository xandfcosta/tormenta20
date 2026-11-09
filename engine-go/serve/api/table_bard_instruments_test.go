package api

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// OS INSTRUMENTOS QUE MUDAM UM PODER (p158).
//
//	"Alaúde Élfico. […] Enquanto empunha este item, você pode usar a habilidade Inspiração como uma ação de
//	 movimento."
//	"Tambor das Profundezas. […] Enquanto empunha este item, o alcance da habilidade Inspiração e de
//	 qualquer Música de Bardo é dobrado."
//
// INTEGRAÇÃO porque a ação de um poder decide três coisas em lugares diferentes — o que se cobra da vez, o
// grupo do chip na superfície Ações e a linha da aba Poderes —, e o que pode quebrar é um deles ler o
// catálogo cru enquanto os outros leem o item.

// bardWithInstrumentOnTurn é um bardo de 5º nível na vez dele, com `item` na mão — ou guardado, com `wear` vazio.
func bardWithInstrumentOnTurn(t *testing.T, item, wear string) (sceneFixture, int64) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)

	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Menestrel", Origin: "artista", Level: 5,
		Strength: 0, Dexterity: 2, Constitution: 1, Intelligence: 1, Wisdom: 0, Charisma: 4,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o bardo: %v", err)
	}
	seedClasse(t, s, id, "Bardo", 5)
	seedMember(t, s, campaignID, id)
	f := sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}

	if _, err := s.queries.CreateItem(context.Background(), sqlcgen.CreateItemParams{
		Characterid: id, Catalogid: sql.NullString{String: item, Valid: true},
		Name: item, Quantity: 1, Slots: 1,
		Equipped:     sql.NullString{String: wear, Valid: wear != ""},
		Improvements: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("pôr %s na mochila: %v", item, err)
	}
	if refused := powerCommand(t, f, id, "escolhe/class.bardo.musica-balada-fascinante", ""); refused != "" {
		t.Fatalf("escolher a Balada Fascinante: %s", refused)
	}
	startActionSceneOnTurnOf(t, f, "Menestrel")
	return f, id
}

// COM O ALAÚDE NA MÃO, A INSPIRAÇÃO CUSTA MOVIMENTO, e a ação padrão sobra para o resto da vez.
//
// O CONTROLE é o mesmo alaúde guardado: a Inspiração volta a custar a padrão (p44). Sem ele, um motor que
// cobrasse movimento de toda postura passaria.
func TestTheElvenLuteMakesInspirationAMovementAction(t *testing.T) {
	for _, c := range []struct {
		name, wear                 string
		standardLeft, movementLeft bool
	}{
		{"com o alaúde empunhado", "wielded", true, false},
		{"com o alaúde guardado", "", false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, bard := bardWithInstrumentOnTurn(t, "alaude-elfico", c.wear)
			if refused := assumes(t, f, bard, "inspiracao"); refused != "" {
				t.Fatalf("entrar na Inspiração: %s", refused)
			}
			scene := stateOf(t, f.s.sessions, f.sessionID).Scene
			if scene.StandardLeft != c.standardLeft || scene.MovementLeft != c.movementLeft {
				t.Errorf("sobrou padrão %v e movimento %v; o livro deixa padrão %v e movimento %v",
					scene.StandardLeft, scene.MovementLeft, c.standardLeft, c.movementLeft)
			}
		})
	}
}

// O CHIP SEGUE A AÇÃO QUE O ITEM DÁ: com o alaúde na mão, a Inspiração está na ação de movimento da
// superfície Ações, e não na padrão.
func TestTheElvenLuteMovesTheInspirationChipToTheMovementGroup(t *testing.T) {
	f, _ := bardWithInstrumentOnTurn(t, "alaude-elfico", "wielded")
	scene := acoesDaMesa(t, f)
	standard := entre(t, scene, "Ação padrão", "Ação de movimento")
	movement := entre(t, scene, "Ação de movimento", "Ação completa")
	if !strings.Contains(movement, "Inspiração") {
		t.Errorf("com o alaúde, a Inspiração não está na ação de movimento:\n%s", recorte(movement))
	}
	if strings.Contains(standard, "Inspiração") {
		t.Errorf("com o alaúde, a Inspiração continua na ação padrão:\n%s", recorte(standard))
	}
}

// COM O TAMBOR NA MÃO, A INSPIRAÇÃO E A MÚSICA AVISAM O ALCANCE DOBRADO. O motor não guarda alcance de poder
// (ele mora no texto), e por isso o que a ficha dá é o aviso na linha — decisão do dono.
//
// Pela LINHA de cada poder, e não por contagem na aba: a folha de cada poder repete a linha, e uma
// contagem crua mediria quantas vezes a tela desenha o poder.
func TestTheDeepDrumWarnsThatTheRangeIsDoubled(t *testing.T) {
	const warning = "alcance dobrado (Tambor das Profundezas)"
	for _, c := range []struct {
		name, wear string
		warns      bool
	}{
		{"com o tambor empunhado", "wielded", true},
		{"com o tambor guardado", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, bard := bardWithInstrumentOnTurn(t, "tambor-das-profundezas", c.wear)
			powers := sheetTab(t, f, bard, "abilities")
			for _, power := range []string{"Inspiração", "Música: Balada Fascinante"} {
				line := regexp.MustCompile(`<span class="ds-row-title">` + regexp.QuoteMeta(power) +
					`</span> <span class="ds-row-meta">([^<]*)<`).FindStringSubmatch(powers)
				if line == nil {
					t.Fatalf("a aba Poderes não tem a linha de %s", power)
				}
				if strings.Contains(line[1], warning) != c.warns {
					t.Errorf("a linha de %s diz %q; o aviso do tambor tinha de estar lá: %v", power, line[1], c.warns)
				}
			}
		})
	}
}

// A CD DE CADA MAGIA SOMA O QUE SOMA NA CD DA FICHA: a superfície Ações escreve "CD N" na linha de conjurar,
// e com a Flauta Mística na mão o N sobe 1, como a CD da aba Magias (p158).
//
// Pela DIFERENÇA entre a flauta empunhada e a guardada, e não por um número fixo: o que se mede é o +1
// chegar à linha, e não a conta de base, que é do motor e tem caso próprio.
func TestTheMysticFluteRaisesTheDCOnTheSpellRow(t *testing.T) {
	spellDC := func(wear string) int {
		f, bard := bardWithInstrumentOnTurn(t, "flauta-mistica", wear)
		// A Luz é evocação, e o bardo só tem a CD das magias das escolas dele (p44).
		if refusal := powerCommand(t, f, bard, bardSchoolsPath+"evocacao", ""); refusal != "" {
			t.Fatalf("escolher evocação: %q", refusal)
		}
		learnSpell(t, f, "luz")
		standard := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
		found := regexp.MustCompile(`Conjurar · Luz.*?CD (\d+)`).FindStringSubmatch(standard)
		if found == nil {
			t.Fatalf("a linha de conjurar a Luz não diz a CD:\n%s", recorte(standard))
		}
		dc, _ := strconv.Atoi(found[1])
		return dc
	}
	if held, stored := spellDC("wielded"), spellDC(""); held-stored != 1 {
		t.Errorf("a linha da Luz diz CD %d com a flauta na mão e %d com ela guardada; o livro pede +1", held, stored)
	}
}
