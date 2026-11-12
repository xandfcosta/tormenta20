package api

import (
	"strings"
	"testing"
)

// A FORMA SELVAGEM na ficha (p62-63): o druida escolhe a forma, o nível que tem e o tipo de dano, paga o
// PM do nível, e a ficha passa a ser a do animal até ele voltar ao normal — o que não devolve o PM.

func wildDruid(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Druida da forma", "Druida", 6, 0, 0)
	choiceCom(t, f, id, `["class.druida.forma-selvagem"]`, `[]`)
	itemSemeia(t, f, id, "adaga", "Adaga", "wielded")
	return f, id
}

func TestTheDruidTakesAWildShapeAndComesBack(t *testing.T) {
	f, id := wildDruid(t)
	if page := powerScreen(t, f, id); !strings.Contains(page, `id="wild-shape"`) || !strings.Contains(page, ">Feroz<") || !strings.Contains(page, "Transformar") {
		t.Error("o diálogo da Forma Selvagem devia oferecer as formas e o Transformar")
	}
	before := pm(t, f, id)
	shape := `{"wild_form":"feroz","wild_tier":"normal","wild_damage":"perfuracao","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", shape); refusal != "" {
		t.Fatalf("a transformação foi recusada: %q", refusal)
	}
	if spent := before - pm(t, f, id); spent != 3 {
		t.Errorf("a forma normal custa 3 PM, e cobrou %d", spent)
	}
	combat := combatScreen(t, f, id)
	if !strings.Contains(combat, "Forma Feroz") || strings.Contains(combat, ">Adaga<") {
		t.Error("na forma, a carta é a da arma da forma, e a adaga não ataca")
	}
	effects := effectScreen(t, f, id)
	if !strings.Contains(effects, "Forma Feroz") || !strings.Contains(effects, "Voltar ao normal") || !strings.Contains(effects, "3 PM pagos") {
		t.Error("a aba Efeitos devia mostrar a forma em vigor e o Voltar ao normal")
	}
	if !strings.Contains(bagScreen(t, f, id), "guardada pela Forma Selvagem") {
		t.Error("a Mochila devia dizer que a adaga está guardada pela forma")
	}

	after := pm(t, f, id)
	if refusal := sheetPost(t, f, id, "/efeitos/forma-selvagem", ""); refusal != "" {
		t.Fatalf("voltar ao normal foi recusado: %q", refusal)
	}
	if pm(t, f, id) != after {
		t.Error("voltar ao normal não devolve o PM pago")
	}
	if strings.Contains(effectScreen(t, f, id), "Voltar ao normal") || !strings.Contains(combatScreen(t, f, id), ">Adaga<") {
		t.Error("de volta ao normal, a forma some dos Efeitos e a adaga volta a atacar")
	}
}

func TestTheWildShapeRefusesATierTheDruidLacksAndABadChoice(t *testing.T) {
	f, id := wildDruid(t)
	improved := `{"wild_form":"feroz","wild_tier":"aprimorada","wild_damage":"corte","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", improved); !strings.Contains(refusal, "Forma Selvagem Aprimorada") {
		t.Errorf("sem o poder, a forma aprimorada é recusada nomeando-o: %q", refusal)
	}
	swift := `{"wild_form":"veloz","wild_tier":"normal","wild_damage":"corte","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", swift); !strings.Contains(refusal, "deslocamento") {
		t.Errorf("a Forma Veloz pede o deslocamento escolhido: %q", refusal)
	}
	fire := `{"wild_form":"feroz","wild_tier":"normal","wild_damage":"fogo","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", fire); !strings.Contains(refusal, "fogo") {
		t.Errorf("o dano da arma da forma é corte, impacto ou perfuração: %q", refusal)
	}
}

// "Cada transformação dura pelo tempo que você quiser, mas você reverte à forma normal se ficar
// inconsciente ou morrer" (p63). Morrer de pé também liga a condição Inconsciente (p236), então é ela
// que desfaz a forma, venha de onde vier: do dano que zera os PV ou da mão de quem a marca. A forma
// não volta quando a condição sai: o livro diz que ela REVERTE, e o PM pago não volta.
func TestTheWildShapeEndsWhenTheDruidFallsUnconscious(t *testing.T) {
	for _, tc := range []struct{ name, fall, rise string }{
		{"pelo dano que zera os PV", "/vitais/pv/-500", "/vitais/pv/600"},
		{"pela condição marcada à mão", "/efeitos/condicao/inconsciente", "/efeitos/condicao/inconsciente"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, id := wildDruid(t)
			shape := `{"wild_form":"feroz","wild_tier":"normal","wild_damage":"perfuracao","wild_move":""}`
			if refusal := powerCommand(t, f, id, "forma-selvagem", shape); refusal != "" {
				t.Fatalf("a transformação foi recusada: %q", refusal)
			}
			if !inTheWildShape(t, f, id) {
				t.Fatal("o druida devia estar na forma antes de cair")
			}

			if refusal := sheetPost(t, f, id, tc.fall, ""); refusal != "" {
				t.Fatalf("cair foi recusado: %q", refusal)
			}
			if !strings.Contains(activeConditionsOf(t, f, id), "inconsciente") {
				t.Fatalf("o caso precisa da condição Inconsciente na ficha, e ela tem %s", activeConditionsOf(t, f, id))
			}
			if inTheWildShape(t, f, id) {
				t.Error("inconsciente, o druida reverte à forma normal (p63), e a forma continua gravada")
			}
			if strings.Contains(effectScreen(t, f, id), "Voltar ao normal") {
				t.Error("a aba Efeitos ainda mostra a forma em vigor")
			}

			if refusal := sheetPost(t, f, id, tc.rise, ""); refusal != "" {
				t.Fatalf("levantar foi recusado: %q", refusal)
			}
			if strings.Contains(activeConditionsOf(t, f, id), "inconsciente") {
				t.Fatalf("o caso precisa tirar a condição Inconsciente, e ela continua: %s", activeConditionsOf(t, f, id))
			}
			if inTheWildShape(t, f, id) {
				t.Error("a forma voltou junto com a consciência: o livro diz que ela reverte, e voltar é transformar de novo")
			}
		})
	}
}

// inTheWildShape lê a tabela da forma, e não a tela: a tela poderia esconder uma forma que continua
// valendo na ficha.
func inTheWildShape(t *testing.T, f sceneFixture, id int64) bool {
	t.Helper()
	rows, err := f.s.queries.GetCharacterWildShape(t.Context(), id)
	if err != nil {
		t.Fatalf("ler a forma de %d: %v", id, err)
	}
	return len(rows) > 0
}
