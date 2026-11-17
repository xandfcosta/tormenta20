package engine

import "testing"

// O ATRIBUTO DE CONJURAÇÃO DE CADA CLASSE É O DO LIVRO, e o do Arcanista é o do
// caminho. Ele mora só no motor: o catálogo de classes também o transcrevia, e a
// linha de magia da ficha lia de lá a do Arcanista sem o caminho. Esta é a
// conferência contra o livro que o teste da progressão de círculos fazia no
// catálogo.
func TestEveryCasterCastsWithTheBooksAttribute(t *testing.T) {
	cases := []struct {
		class, path, attribute string
		page                   int
	}{
		// "Seu atributo-chave para lançar magias é definido pelo seu Caminho" (p37).
		{"Arcanista", "bruxo", "intelligence", 37},
		{"Arcanista", "feiticeiro", "charisma", 37},
		{"Arcanista", "mago", "intelligence", 37},
		// A ficha antes da escolha obrigatória do 1º nível: o de dois dos três caminhos.
		{"Arcanista", "", "intelligence", 37},
		{"Bardo", "", "charisma", 44},
		{"Clérigo", "", "wisdom", 57},
		{"Druida", "", "wisdom", 61},
		// O Orar: "Seu atributo-chave para esta magia é Sabedoria" (p83).
		{"Paladino", "", "wisdom", 83},
		{"Guerreiro", "", "", 0},
	}
	for _, c := range cases {
		ch := Character{Classes: []CharacterClass{{ClassName: c.class, Level: 5}}, ClassChoices: "{}"}
		if c.path != "" {
			ch.ClassChoices = `{"` + c.class + `":{"caminho":"` + c.path + `"}}`
		}
		got := spellAttributeByClass(ch)[c.class]
		if got != c.attribute {
			t.Errorf("%s (caminho %q) lança com %q, e o livro diz %q (p%d)", c.class, c.path, got, c.attribute, c.page)
		}
	}
}
