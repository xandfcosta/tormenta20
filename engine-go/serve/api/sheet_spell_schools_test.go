package api

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// AS ESCOLAS DO BARDO E DO DRUIDA (p44, p61): "Escolha três escolas de magia. Uma vez feita, essa escolha
// não pode ser mudada. Você pode lançar magias arcanas [divinas, no druida] de 1º círculo que pertençam a
// essas escolas."
//
// A escolha mora no poder Magias (1º círculo) da classe, como a escolha contada do Especialista, e decide
// de que magias a ficha diz a CD — decisão do dono.

// schoolCaster é um personagem de 1º nível de `class`, o ÚNICO do jogador na mesa: a mesa desenha o primeiro
// personagem de quem olha, e um segundo faria a superfície Ações mostrar outra ficha.
func schoolCaster(t *testing.T, class string) (sceneFixture, int64) {
	t.Helper()
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t.com")
	player := seedUser(t, s, "jogador@t.com")
	campaignID := seedCampaign(t, s, gm)
	sessionID := seedSession(t, s, campaignID)
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: player, Name: "Conjurador de escola", Origin: "artista", Level: 1,
		Charisma: 3, Wisdom: 3,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear o %s: %v", class, err)
	}
	seedClasse(t, s, id, class, 1)
	seedMember(t, s, campaignID, id)
	return sceneFixture{s: s, gm: gm, player: player,
		campaignID: campaignID, sessionID: sessionID, charID: id}, id
}

const (
	bardSchoolsPath  = "escolha/class.bardo.magias-1-circulo/"
	druidSchoolsPath = "escolha/class.druida.magias-1-circulo/"
)

// TRÊS ESCOLAS, distintas, e a ficha avisa enquanto faltam.
func TestTheBardChoosesThreeSchools(t *testing.T) {
	f, id := schoolCaster(t, "Bardo")
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Bardo · Magias (1° círculo)"); step.Tag != "0 de 3" {
		t.Errorf("sem escola escolhida, o passo das escolas diz %q", step.Tag)
	}
	// O BLOCO diz quantas de quantas e o que o livro diz da escolha, e não o rodapé das perícias do
	// Especialista, de quem ele empresta a forma.
	page := powerScreen(t, f, id)
	for _, want := range []string{`aria-label="Magias (1° círculo): Evocação"`, ">0 de 3<", "não pode ser mudada"} {
		if !strings.Contains(page, want) {
			t.Errorf("o bloco das escolas não tem %q", want)
		}
	}
	if strings.Contains(page, "Só as perícias treinadas") {
		t.Error("o bloco das escolas diz o rodapé das perícias do Especialista")
	}
	for _, school := range []string{"evocacao", "ilusao", "abjuracao"} {
		if refusal := powerCommand(t, f, id, bardSchoolsPath+school, ""); refusal != "" {
			t.Fatalf("escolher %s foi recusado: %q", school, refusal)
		}
	}
	if refusal := powerCommand(t, f, id, bardSchoolsPath+"necromancia", ""); !strings.Contains(refusal, "3 escolas") {
		t.Errorf("a quarta escola foi aceita, ou a recusa não diz o limite: %q", refusal)
	}
	if got := storedPowerChoices(t, f, id); got != `{"class.bardo.magias-1-circulo":["evocacao","ilusao","abjuracao"]}` {
		t.Errorf("powerChoices gravado = %s", got)
	}
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Bardo · Magias (1° círculo)"); step.Tag != "feito" {
		t.Errorf("com as três escolhidas, o passo das escolas diz %q", step.Tag)
	}
}

// O druida escolhe as dele pelo mesmo gesto (p61).
func TestTheDruidChoosesThreeSchools(t *testing.T) {
	f, id := schoolCaster(t, "Druida")
	if step := choiceStepTitled(t, powerScreen(t, f, id), "Druida · Magias (1° círculo)"); step.Tag != "0 de 3" {
		t.Errorf("sem escola escolhida, o passo das escolas do druida diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, druidSchoolsPath+"abjuracao", ""); refusal != "" {
		t.Fatalf("escolher abjuração foi recusado: %q", refusal)
	}
}

// spellDCOnTheTable é o que a linha de conjurar diz da CD da magia `name` na superfície Ações: o número, ou
// "—".
func spellDCOnTheTable(t *testing.T, f sceneFixture, name string) string {
	t.Helper()
	standard := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	found := regexp.MustCompile(`Conjurar · ` + regexp.QuoteMeta(name) + `.*?CD (\d+|—)`).FindStringSubmatch(standard)
	if found == nil {
		t.Fatalf("a linha de conjurar %s não diz a CD:\n%s", name, recorte(standard))
	}
	return found[1]
}

// A CD SEGUE AS ESCOLAS: a Explosão de Chamas é arcana de evocação, e o verbete a lista só para o
// Arcanista. O bardo que escolheu evocação a lança e tem a CD dela; o que não escolheu, não.
func TestTheBardsSpellDCFollowsHisSchools(t *testing.T) {
	for _, c := range []struct {
		name    string
		schools []string
		hasDC   bool
	}{
		{"com evocação", []string{"evocacao"}, true},
		{"sem evocação", []string{"ilusao"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := schoolCaster(t, "Bardo")
			for _, school := range c.schools {
				if refusal := powerCommand(t, f, id, bardSchoolsPath+school, ""); refusal != "" {
					t.Fatalf("escolher %s: %q", school, refusal)
				}
			}
			learnSpell(t, f, "explosao-de-chamas")
			if got := spellDCOnTheTable(t, f, "Explosão de Chamas"); (got != "—") != c.hasDC {
				t.Errorf("a Explosão de Chamas diz CD %s; a CD tinha de estar lá: %v", got, c.hasDC)
			}
		})
	}
}

// O DRUIDA LANÇA DIVINAS das escolas dele: o Santuário é divino de abjuração, e o verbete o lista para o
// Clérigo e o Paladino.
func TestTheDruidsSpellDCFollowsHisSchools(t *testing.T) {
	f, id := schoolCaster(t, "Druida")
	if refusal := powerCommand(t, f, id, druidSchoolsPath+"abjuracao", ""); refusal != "" {
		t.Fatalf("escolher abjuração: %q", refusal)
	}
	learnSpell(t, f, "santuario")
	if got := spellDCOnTheTable(t, f, "Santuário"); got == "—" {
		t.Errorf("o druida de abjuração não tem a CD do Santuário")
	}
}

// A ESCOLA SOZINHA NÃO BASTA: o bardo lança ARCANAS (p44), e o Santuário é divino de abjuração. O bardo de
// abjuração não tem a CD dele.
func TestTheBardDoesNotCastDivineSpellsOfHisSchools(t *testing.T) {
	f, id := schoolCaster(t, "Bardo")
	if refusal := powerCommand(t, f, id, bardSchoolsPath+"abjuracao", ""); refusal != "" {
		t.Fatalf("escolher abjuração: %q", refusal)
	}
	learnSpell(t, f, "santuario")
	if got := spellDCOnTheTable(t, f, "Santuário"); got != "—" {
		t.Errorf("o bardo de abjuração tem CD %s no Santuário, que é divino", got)
	}
}

// TIRAR UMA CLASSE PERGUNTA A MESMA COISA: a magia que outra classe da ficha lança não segura ninguém. O
// bardo de evocação lança a Bola de Fogo, que o verbete lista só para o Arcanista, e por isso o Guerreiro
// que ele tem ao lado sai.
func TestTheBardsSchoolSpellDoesNotHoldAnotherClass(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Menestrel guerreiro", "Bardo", 3, 0, 0)
	if refusal := powerCommand(t, f, id, bardSchoolsPath+"evocacao", ""); refusal != "" {
		t.Fatalf("escolher evocação: %q", refusal)
	}
	if refusal := enterClass(t, f, id, "Guerreiro"); refusal != "" {
		t.Fatalf("entrar no Guerreiro: %q", refusal)
	}
	learn(t, f, id, "bola-de-fogo")
	if refusal := stepClass(t, f, id, "Guerreiro", -1); refusal != "" {
		t.Errorf("a Bola de Fogo, que o bardo lança, segurou o Guerreiro: %q", refusal)
	}
}

// O CONTROLE: o arcanista lança toda arcana (p37), e o Especialista em Escola, que também escolhe escola,
// não o torna um conjurador por escola. Sem este caso, uma regra que pegasse todo poder com escolha de
// escola tiraria a CD de todo arcanista, e o resto da suíte passaria.
func TestTheArcanistKeepsTheDCOfEveryArcaneSpell(t *testing.T) {
	f, _ := schoolCaster(t, "Arcanista")
	learnSpell(t, f, "explosao-de-chamas")
	if got := spellDCOnTheTable(t, f, "Explosão de Chamas"); got == "—" {
		t.Error("o arcanista perdeu a CD da Explosão de Chamas")
	}
}
