package sheet

import (
	"slices"
	"testing"

	"t20engine/domain/book"
)

// O CONJURADOR responde, sobre cada magia, o que aprender, conjurar e a aba Magias perguntam (p170-171).

// AS CLASSES DA FICHA QUE LANÇAM A MAGIA: o Bardo 5/Arcanista 1 de evocação lança a Explosão de Chamas pelas
// duas classes, embora o verbete liste só o Arcanista (p44); sem evocação, só pelo Arcanista.
func TestTheCastingClassesFollowTheBardsSchools(t *testing.T) {
	multiclass := func(schools string) CharacterDTO {
		return CharacterDTO{
			Classes:      []ClassDTO{{ClassName: "Bardo", Level: 5}, {ClassName: "Arcanista", Level: 1}},
			PowerChoices: `{"class.bardo.magias-1-circulo":` + schools + `}`,
		}
	}
	for _, c := range []struct {
		name, schools string
		want          []string
	}{
		{"com evocação", `["evocacao"]`, []string{"Bardo", "Arcanista"}},
		{"sem evocação", `["ilusao"]`, []string{"Arcanista"}},
	} {
		spell := CasterOf(multiclass(c.schools)).Spell("explosao-de-chamas")
		if !slices.Equal(spell.Casting, c.want) {
			t.Errorf("%s: lançam a Explosão de Chamas %v, e o livro diz %v", c.name, spell.Casting, c.want)
		}
		if spell.Tradition != "arcana" {
			t.Errorf("%s: a tradição saiu %q", c.name, spell.Tradition)
		}
	}
}

// O CÍRCULO QUE A FICHA ALCANÇA É POR MAGIA (p170-171): o das classes que a lançam, e não o maior entre todas;
// sem nenhuma delas, o das classes que conjuram (um poder ensinou a magia de fora da lista, como o Aspecto da
// Primavera, p61); e, sem classe que conjure, o da própria magia, que veio de fora das classes.
func TestTheCircleReachIsTheOneOfTheClassesThatCastTheSpell(t *testing.T) {
	arcaneAndDivine := CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 9}, {ClassName: "Clérigo", Level: 1}}}
	for _, c := range []struct {
		why   string
		dto   CharacterDTO
		spell string
		want  int
	}{
		// O arcanista de 9º nível alcança o 3º círculo; o clérigo de 1º, só o 1º.
		{"a magia arcana pelo arcanista de 9º", arcaneAndDivine, "bola-de-fogo", 3},
		{"a magia divina pelo clérigo de 1º, e não pelo arcanista", arcaneAndDivine, "curar-ferimentos", 1},
		// O arcanista de 4º nível alcança só o 1º: o 2º círculo vem no 5º nível.
		{"o arcanista de 4º", CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 4}}}, "bola-de-fogo", 1},
		// A Metamorfose é só arcana: o druida não a lança pela lista, e vale o círculo que ele alcança.
		{"a magia de fora da lista, pelo druida", CharacterDTO{Classes: []ClassDTO{{ClassName: "Druida", Level: 1}}}, "metamorfose", 1},
		{"a magia de quem não conjura vale o círculo dela", CharacterDTO{Classes: []ClassDTO{{ClassName: "Bárbaro", Level: 2}}}, "bola-de-fogo", 2},
	} {
		if got := CasterOf(c.dto).Spell(c.spell).CircleReach; got != c.want {
			t.Errorf("%s: alcança o %dº círculo para %s; want %dº", c.why, got, c.spell, c.want)
		}
	}
}

// A RECUSA E O AVISO: a magia acima do círculo tem a frase da recusa; a de fora da lista, o aviso, e nenhuma
// recusa. Magia desconhecida não tem nenhum dos dois: quem a recusa é quem procura no livro.
func TestTheSpellAboveTheCircleIsRefusedAndTheOneOffTheListIsWarned(t *testing.T) {
	caster := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 4}}})
	if got, want := caster.Spell("bola-de-fogo").AboveTheCircle, "Bola de Fogo é do 2º círculo, e esta ficha alcança até o 1º (p170)"; got != want {
		t.Errorf("a recusa da Bola de Fogo no Arcanista 4 saiu %q; want %q", got, want)
	}
	if got := caster.Spell("explosao-de-chamas"); got.AboveTheCircle != "" || got.OffTheList != "" {
		t.Errorf("a Explosão de Chamas, da lista e do 1º círculo, saiu com recusa %q e aviso %q", got.AboveTheCircle, got.OffTheList)
	}
	healing := caster.Spell("curar-ferimentos")
	if healing.OffTheList == "" || healing.AboveTheCircle != "" {
		t.Errorf("o Curar Ferimentos, divino e do 1º círculo, saiu com aviso %q e recusa %q", healing.OffTheList, healing.AboveTheCircle)
	}
	if unknown := caster.Spell("nao-existe"); unknown.Known || unknown.AboveTheCircle != "" || unknown.OffTheList != "" {
		t.Errorf("a magia desconhecida saiu %+v", unknown)
	}
}

// A TRADIÇÃO PRINCIPAL é a da classe conjuradora de maior nível: a que as caixas de CD e de limite de PM do
// Combate resumem. Empate de magia nas duas listas resolve pelo maior nível, como o limite de PM (p224).
func TestTheMainTraditionIsTheHighestLevelCastersList(t *testing.T) {
	for _, c := range []struct {
		classes []ClassDTO
		want    string
	}{
		{[]ClassDTO{{ClassName: "Arcanista", Level: 3}, {ClassName: "Clérigo", Level: 5}}, "divina"},
		{[]ClassDTO{{ClassName: "Arcanista", Level: 5}, {ClassName: "Clérigo", Level: 3}}, "arcana"},
		{[]ClassDTO{{ClassName: "Guerreiro", Level: 5}}, ""},
	} {
		if got := CasterOf(CharacterDTO{Classes: c.classes}).MainTradition(); got != c.want {
			t.Errorf("%v: a tradição principal saiu %q; want %q", c.classes, got, c.want)
		}
	}
	both := CasterOf(CharacterDTO{Classes: []ClassDTO{{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 10}}})
	if got := both.Spell("luz").Tradition; got != "arcana" {
		t.Errorf("a Luz do Arcanista 20/Clérigo 10 saiu %q; want arcana, pela classe de maior nível", got)
	}
}

// O APRIMORAMENTO "APENAS ARCANOS / DIVINOS / DRUIDAS" (a Luz, p197; o Controlar o Clima, p187) existe só para
// quem lança a magia por aquela tradição ou classe. A Luz é arcana na mão do arcanista e divina na do clérigo; o
// multiclasse a lança pela classe de maior nível, a mesma regra da tradição.
func TestTheClassOnlyAugmentFollowsHowTheSheetCastsTheSpell(t *testing.T) {
	sheetOf := func(classes ...ClassDTO) Caster { return CasterOf(CharacterDTO{Classes: classes}) }
	// O druida lança as divinas das escolas que escolheu (p61): a de transmutação traz o Controlar o Clima.
	druid := CasterOf(CharacterDTO{
		Classes:      []ClassDTO{{ClassName: "Druida", Level: 14}},
		PowerChoices: `{"class.druida.magias-1-circulo":["transmutacao"]}`,
	})
	for _, c := range []struct {
		why       string
		caster    Caster
		spell     string
		classOnly string
		want      bool
	}{
		{"a Luz do arcanista é arcana", sheetOf(ClassDTO{ClassName: "Arcanista", Level: 5}), "luz", "arcanos", true},
		{"a Luz do arcanista não é divina", sheetOf(ClassDTO{ClassName: "Arcanista", Level: 5}), "luz", "divinos", false},
		{"a Luz do clérigo é divina", sheetOf(ClassDTO{ClassName: "Clérigo", Level: 5}), "luz", "divinos", true},
		{"a Luz do Clérigo 5 / Arcanista 3 é divina", sheetOf(ClassDTO{ClassName: "Clérigo", Level: 5}, ClassDTO{ClassName: "Arcanista", Level: 3}), "luz", "arcanos", false},
		{"o Controlar o Clima do druida", druid, "controlar-o-clima", "druidas", true},
		{"o Controlar o Clima do clérigo não é de druida", sheetOf(ClassDTO{ClassName: "Clérigo", Level: 14}), "controlar-o-clima", "druidas", false},
		{"sem marca, todos", sheetOf(ClassDTO{ClassName: "Clérigo", Level: 5}), "luz", "", true},
		{"a magia que nenhuma classe lança não é de tradição nenhuma", sheetOf(ClassDTO{ClassName: "Bárbaro", Level: 5}), "luz", "arcanos", false},
	} {
		if got := c.caster.Spell(c.spell).CastsAs(c.classOnly); got != c.want {
			t.Errorf("%s: CastsAs(%q) = %v; want %v", c.why, c.classOnly, got, c.want)
		}
	}
}

// TODA MARCA "Apenas …" DO CATÁLOGO tem quem a responda: uma marca nova sem resposta trancaria o aprimoramento
// para todo mundo, calada.
func TestEveryClassOnlyMarkIsUnderstood(t *testing.T) {
	seen := 0
	for _, spell := range book.Catalogs().Spells {
		for i, augment := range spell.Augments {
			if augment.ClassOnly == "" {
				continue
			}
			seen++
			if _, known := classOnlyCasts[augment.ClassOnly]; !known {
				t.Errorf("%s, aprimoramento %d: a marca %q não tem resposta em classOnlyCasts; escreva o que ela pergunta", spell.ID, i, augment.ClassOnly)
			}
		}
	}
	if seen == 0 {
		t.Error("nenhum aprimoramento com marca: o catálogo perdeu o campo, e o guarda não mede nada")
	}
}
