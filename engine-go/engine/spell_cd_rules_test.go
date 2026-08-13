package engine

import "testing"

// CD de magia — o livro enuncia a mesma fórmula duas vezes, no capítulo de
// magia (p173, quadro "Resistência") e no de habilidades (p227):
//
//	"Dificuldade. A CD do teste de resistência contra uma magia é 10 + metade
//	 do nível do personagem + atributo-chave da magia."
//
//	"A CD do teste de resistência para qualquer efeito gerado por um personagem
//	 é 10 + metade do nível do personagem + seu valor num atributo."
//
// E o atributo-chave por classe, nas entradas de classe do Capítulo 2:
//   - Arcanista (p37): "Seu atributo-chave para lançar magias é DEFINIDO PELO
//     SEU CAMINHO" — Bruxo e Mago → Inteligência, Feiticeiro → Carisma.
//   - Bardo (p44) → Carisma; Clérigo (p57) e Druida (p61) → Sabedoria.
//
// A cobertura vinha só dos oráculos de paridade, que provam que os dois motores
// CONCORDAM e nunca que algum acerta o livro (ALE-105).

func effectsNone() ItemEffects { return ItemEffects{Flags: map[string]bool{}} }

// A fórmula crua. "Metade do nível" arredonda para baixo — a convenção do livro,
// escrita com todas as letras em outro lugar ("metade do seu nível, arredondado
// para baixo", p10) — e o nível ÍMPAR é o único lugar onde isso aparece.
func TestSpellSaveDcFormula(t *testing.T) {
	tests := []struct {
		level, mod, want int
		nota             string
	}{
		{1, 0, 10, "no 1º nível a metade é 0, não 1"},
		{2, 0, 11, ""},
		{3, 0, 11, "ímpar arredonda para BAIXO"},
		{8, 5, 19, "Samira (p173): 10 + 4 + 5"},
		{20, 0, 20, "o teto de nível"},
		{4, -2, 10, "atributo negativo desce a CD, sem piso"},
	}
	for _, tt := range tests {
		if got := spellSaveDc(tt.level, tt.mod); got != tt.want {
			t.Errorf("nível %d, atributo %+d: CD = %d, want %d %s", tt.level, tt.mod, got, tt.want, tt.nota)
		}
	}
}

// O exemplo trabalhado do próprio livro, p173:
//
//	"Samira é uma qareen FEITICEIRA de 8º nível com Carisma 5. A CD para resistir
//	 a suas magias é 19 (10 +4 +5 = 19)."
//
// Feiticeiro é um CAMINHO do Arcanista, e o motor cravava Inteligência para todo
// Arcanista — então a Samira do livro saía com a CD do atributo errado. O
// catálogo já modelava o caminho do lado do PM (o `maxPm` do Feiticeiro escala
// por Carisma); era só a metade da CD que ficou para trás (ALE-113).
func TestSamiraTheBookWorkedExample(t *testing.T) {
	samira := Character{
		Level:        8,
		Intelligence: 0,
		Charisma:     5,
		Classes:      []CharacterClass{{ClassName: "Arcanista", Level: 8}},
		ClassChoices: `{"Arcanista":{"caminho":"feiticeiro"}}`,
	}
	got := bestBaseSpellCd(samira, effectsNone())
	if got == nil {
		t.Fatal("CD = nil — a Samira é conjuradora")
	}
	if *got != 19 {
		t.Errorf("CD da Samira = %d, want 19 (10 + 4 + Carisma 5)", *got)
	}
}

// "Seu atributo-chave para lançar magias é definido pelo seu Caminho" (p37).
// Mesmo personagem, mesmos atributos, muda só a escolha — se o motor ignorar o
// caminho, os três dão o mesmo número.
func TestArcanistaKeyAttributeFollowsTheCaminho(t *testing.T) {
	arcanista := func(caminho string) Character {
		return Character{
			Level:        10,
			Intelligence: 2,
			Charisma:     5,
			Classes:      []CharacterClass{{ClassName: "Arcanista", Level: 10}},
			ClassChoices: `{"Arcanista":{"caminho":"` + caminho + `"}}`,
		}
	}
	// Nível 10 → 10 + 5 = 15 de base. Inteligência 2 → 17; Carisma 5 → 20.
	tests := []struct {
		caminho string
		want    int
		atrib   string
	}{
		{"bruxo", 17, "Inteligência"},
		{"mago", 17, "Inteligência"},
		{"feiticeiro", 20, "Carisma"},
	}
	for _, tt := range tests {
		got := bestBaseSpellCd(arcanista(tt.caminho), effectsNone())
		if got == nil || *got != tt.want {
			t.Errorf("caminho %s: CD = %v, want %d (%s)", tt.caminho, deref(got), tt.want, tt.atrib)
		}
	}

	// Ficha meio preenchida: a escolha é obrigatória no 1º nível, mas o
	// personagem pode existir antes dela. Inteligência é o padrão porque é o de
	// dois dos três caminhos — e é o que a ficha já mostrava.
	semCaminho := Character{
		Level:        10,
		Intelligence: 2,
		Charisma:     5,
		Classes:      []CharacterClass{{ClassName: "Arcanista", Level: 10}},
		ClassChoices: `{}`,
	}
	if got := bestBaseSpellCd(semCaminho, effectsNone()); got == nil || *got != 17 {
		t.Errorf("sem caminho escolhido: CD = %v, want 17 (cai em Inteligência)", deref(got))
	}
}

// As duas regras do PM e da CD são OPOSTAS e moram em páginas diferentes, o que
// as torna fáceis de trocar: a CD usa "metade do nível DO PERSONAGEM" (p173),
// enquanto o teto de PM usa "seu nível NA CLASSE que fornece a habilidade"
// (p224). Este teste prende a metade da CD; `TestSpellPmLimit` prende a outra.
func TestSpellCdUsesCharacterLevelNotClassLevel(t *testing.T) {
	guerreiroArcanista := Character{
		Level:        10,
		Intelligence: 3,
		Classes: []CharacterClass{
			{ClassName: "Guerreiro", Level: 9},
			{ClassName: "Arcanista", Level: 1},
		},
		ClassChoices: `{"Arcanista":{"caminho":"mago"}}`,
	}
	// 10 + ½ de 10 + 3 = 18. Pelo nível de Arcanista daria 10 + 0 + 3 = 13.
	got := bestBaseSpellCd(guerreiroArcanista, effectsNone())
	if got == nil || *got != 18 {
		t.Errorf("CD = %v, want 18 — usou o nível da CLASSE em vez do de personagem?", deref(got))
	}

	// E o teto de PM da mesma ficha faz o contrário: 1, o nível de Arcanista.
	if limit := SpellPmLimit(guerreiroArcanista, 0, []string{"Arcanista"}); limit != 1 {
		t.Errorf("limite de PM = %d, want 1 — as duas regras não são a mesma", limit)
	}
}

// Multiclasse de duas conjuradoras: cada classe lança com o SEU atributo-chave,
// então a ficha mostra a melhor das duas.
func TestBestSpellCdTakesTheBestCasterClass(t *testing.T) {
	bardoClerigo := Character{
		Level:    8,
		Charisma: 5,
		Wisdom:   1,
		Classes: []CharacterClass{
			{ClassName: "Bardo", Level: 4},
			{ClassName: "Clérigo", Level: 4},
		},
	}
	// Bardo/Carisma → 19; Clérigo/Sabedoria → 15.
	if got := bestBaseSpellCd(bardoClerigo, effectsNone()); got == nil || *got != 19 {
		t.Errorf("CD = %v, want 19 (a melhor das duas)", deref(got))
	}
}

// A CD usa o atributo FINAL, com bônus de raça e item dentro — não o valor cru
// da ficha. Um Osteon Necromante saía com 21 na tela quando o correto era 22.
func TestSpellCdUsesTheFinalAttribute(t *testing.T) {
	arcanista := Character{
		Level:        10,
		Intelligence: 2,
		Classes:      []CharacterClass{{ClassName: "Arcanista", Level: 10}},
		ClassChoices: `{"Arcanista":{"caminho":"mago"}}`,
	}
	comItem := ItemEffects{
		Flags: map[string]bool{},
		ByTarget: map[string]AggregatedStat{
			targetKey(ModifierTarget{K: "attribute", Name: "intelligence"}): {
				Total:         2,
				Contributions: []Contribution{{Source: "Cajado do Sábio", BonusType: "item", Amount: 2}},
			},
		},
	}
	if got := bestBaseSpellCd(arcanista, comItem); got == nil || *got != 19 {
		t.Errorf("CD = %v, want 19 (15 + Inteligência 2 + item 2)", deref(got))
	}
}

// Quem não lança magia não tem CD de magia — nil, e não 10.
func TestNonCasterHasNoSpellCd(t *testing.T) {
	barbaro := Character{
		Level:   10,
		Classes: []CharacterClass{{ClassName: "Bárbaro", Level: 10}},
	}
	if got := bestBaseSpellCd(barbaro, effectsNone()); got != nil {
		t.Errorf("CD = %d, want nil — o Bárbaro não lança magias", *got)
	}
}

func deref(p *int) any {
	if p == nil {
		return "nil"
	}
	return *p
}
