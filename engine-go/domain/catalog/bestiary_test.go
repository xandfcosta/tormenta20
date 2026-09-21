package catalog

import (
	"encoding/json"
	"testing"
)

// O bestiário — 80 verbetes transcritos do livro.
//
// A regra da casa para catálogo é SCHEMA no dump, não um `expect` por campo
// repetindo o mesmo número: prender a tabela inteira só transcreve o erro para
// um segundo lugar. O que se prende é a EXCEÇÃO — a armadilha da tabela.
//
// Aqui a armadilha é o atributo AUSENTE. O livro escreve `Int —` no Zumbi, no
// Esqueleto, no Golem de Ferro e em mais seis, e `For —` na Aparição: a
// criatura não TEM aquele atributo. A importação inventava um número no lugar
// — o Glop ficou com `Int -5` —, e um número diz outra coisa: que ele tem
// Inteligência, muito baixa. São dez casos, e é exatamente o tipo de dado que
// alguém "conserta" de volta para zero achando que é um buraco.

type monsterRow struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ND           any    `json:"nd"`
	Kind         string `json:"tipo"`
	Size         string `json:"size"`
	HP           *int   `json:"hp"`
	Defense      *int   `json:"defesa"`
	Strength     *int   `json:"forca"`
	Dexterity    *int   `json:"destreza"`
	Constitution *int   `json:"constituicao"`
	Intelligence *int   `json:"inteligencia"`
	Wisdom       *int   `json:"sabedoria"`
	Charisma     *int   `json:"carisma"`
	Fortitude    *int   `json:"fortitude"`
	Reflex       *int   `json:"reflexos"`
	Will         *int   `json:"vontade"`
	Speed        string `json:"deslocamento"`
	Initiative   *int   `json:"iniciativa"`
	Perception   *int   `json:"percepcao"`
	PM           *int   `json:"pm"`
	Skills       *[]struct {
		Name  string `json:"name"`
		Bonus int    `json:"bonus"`
		Note  string `json:"nota"`
	} `json:"skills"`
	Equipment *string `json:"equipamento"`
	Treasure  *string `json:"tesouro"`
	// TreasureXp não existe mais: era `nd * 1000` nos OITENTA verbetes, a mesma
	// conta do `xpForNd`, e o nome mentia. O ponteiro fica para o teste poder
	// afirmar a AUSÊNCIA — sem ele, um campo ressuscitado passaria despercebido.
	TreasureXp *int `json:"treasureXp"`
	BookPage   int  `json:"bookPage"`
}

func lerBestiario(t *testing.T) []monsterRow {
	t.Helper()
	raw, err := files.ReadFile("data/bestiary.json")
	if err != nil {
		t.Fatalf("ler bestiary.json: %v", err)
	}
	var rows []monsterRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("bestiary.json não casa com o schema: %v", err)
	}
	return rows
}

// O schema: o que NUNCA pode faltar, e a faixa de página do capítulo.
func TestBestiarySchema(t *testing.T) {
	rows := lerBestiario(t)
	if len(rows) != 80 {
		t.Fatalf("verbetes=%d, queria 80", len(rows))
	}
	seen := map[string]bool{}
	for _, m := range rows {
		if m.ID == "" || m.Name == "" || m.Kind == "" || m.Size == "" {
			t.Errorf("%q: identidade incompleta", m.Name)
		}
		if seen[m.ID] {
			t.Errorf("id repetido: %q", m.ID)
		}
		seen[m.ID] = true
		// Vitais e resistências são do bloco de TODA criatura: o livro não tem
		// verbete sem Defesa nem sem Pontos de Vida.
		for name, v := range map[string]*int{
			"hp": m.HP, "defesa": m.Defense,
			"fortitude": m.Fortitude, "reflexos": m.Reflex, "vontade": m.Will,
		} {
			if v == nil {
				t.Errorf("%q: %s ausente — nenhum verbete do livro omite isso", m.Name, name)
			}
		}
		// Iniciativa, Percepção e Tesouro existem em TODO verbete do livro;
		// perícias e equipamento existem como CAMPO em todos, vazios em quem não
		// tem.
		for name, v := range map[string]any{
			"iniciativa": m.Initiative, "percepcao": m.Perception,
		} {
			if v == (*int)(nil) {
				t.Errorf("%q: %s ausente — abre o bloco de toda criatura", m.Name, name)
			}
		}
		if m.Skills == nil {
			t.Errorf("%q: skills ausente — vazio é uma lista vazia, não um buraco", m.Name)
		}
		if m.Equipment == nil {
			t.Errorf("%q: equipamento ausente — vazio é string vazia, não um buraco", m.Name)
		}
		if m.Treasure == nil || *m.Treasure == "" {
			t.Errorf("%q: tesouro vazio — o livro dá um a todo verbete", m.Name)
		}
		if m.TreasureXp != nil {
			t.Errorf("%q: treasureXp voltou — era nd*1000 duplicado, e o nome mentia", m.Name)
		}

		// O capítulo do bestiário. Uma página fora da faixa é transcrição
		// errada, e foi assim que verbetes ficaram com números do vizinho.
		if m.BookPage < 286 || m.BookPage > 316 {
			t.Errorf("%q: p%d fora do capítulo (286–316)", m.Name, m.BookPage)
		}
	}
}

// A exceção presa pelo nome: quem NÃO tem um atributo, e qual.
//
// Prender o conjunto inteiro (e não só "existe algum nulo") é de propósito: o
// risco real é alguém preencher um deles de volta com zero, e uma asserção
// frouxa passaria verde com nove dos dez consertados.
func TestBestiaryMissingAttribute(t *testing.T) {
	noAttribute := map[string]string{
		"zumbi":                 "inteligencia",
		"esqueleto":             "inteligencia",
		"esqueleto-elite":       "inteligencia",
		"turba-zumbi":           "inteligencia",
		"golem-de-ferro":        "inteligencia",
		"glop":                  "inteligencia",
		"colosso-supremo":       "inteligencia",
		"falange":               "inteligencia",
		"engenho-guerra-goblin": "inteligencia",
		"aparicao":              "forca",
	}
	attributes := func(m monsterRow) map[string]*int {
		return map[string]*int{
			"forca": m.Strength, "destreza": m.Dexterity, "constituicao": m.Constitution,
			"inteligencia": m.Intelligence, "sabedoria": m.Wisdom, "carisma": m.Charisma,
		}
	}

	findings := map[string]string{}
	for _, m := range lerBestiario(t) {
		for name, v := range attributes(m) {
			if v == nil {
				findings[m.ID] = name
			}
		}
	}
	for id, field := range noAttribute {
		if findings[id] != field {
			t.Errorf("%s: o livro marca %s com travessão, e o catálogo tem %v",
				id, field, findings[id])
		}
	}
	for id, field := range findings {
		if noAttribute[id] != field {
			t.Errorf("%s: %s virou ausente e não está na lista do livro", id, field)
		}
	}
}
