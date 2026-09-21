package markdown

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// O ORÁCULO DO MARKDOWN.
//
// A gramática tem divergências deliberadas do markdown padrão que uma biblioteca
// desfaria, e o oráculo é a LINHA DE BASE congelada que as protege: ele acusa
// qualquer árvore que mude sem ter sido pedido. Ele NÃO prova que duas
// implementações concordam — não há segunda —, então o diff dele se revisa
// contra o que se queria mudar, nunca se aceita porque "ficou verde".
//
// O oráculo é comparado como ÁRVORE e não como texto JSON: chave fora de ordem
// ou campo omitido são detalhe de serialização, e um teste que os prendesse
// falharia por motivo errado.

// oraculoDoMarkdown é a forma do arquivo gerado pelo script.
type oraculoDoMarkdown struct {
	Trees []struct {
		Note   string  `json:"nota"`
		Source string  `json:"fonte"`
		Blocks []Block `json:"blocos"`
	} `json:"arvores"`
	Toggle []struct {
		Note   string `json:"nota"`
		Source string `json:"fonte"`
		Row    int    `json:"linha"`
		Marked bool   `json:"marcada"`
		Output string `json:"saida"`
	} `json:"alterna"`
}

func leOOraculoDoMarkdown(t *testing.T) oraculoDoMarkdown {
	t.Helper()
	raw, err := os.ReadFile("testdata/markdown-from-the-js.json")
	if err != nil {
		t.Fatalf("oráculo ausente — ele é versionado e não se regenera mais: %v", err)
	}
	var o oraculoDoMarkdown
	if err := json.Unmarshal(raw, &o); err != nil {
		t.Fatalf("oráculo ilegível: %v", err)
	}
	// O CONTROLE: um oráculo vazio faria todos os laços abaixo passarem verde
	// sem comparar nada, que é o formato exato do teste que não mede — o
	// arquivo pode existir e estar vazio se o script mudar de forma.
	if len(o.Trees) == 0 || len(o.Toggle) == 0 {
		t.Fatal("o oráculo está vazio — os laços abaixo passariam verde sem comparar nada")
	}
	return o
}

func TestTheNoteMarkdownMatchesTheJs(t *testing.T) {
	oracle := leOOraculoDoMarkdown(t)
	for _, tc := range oracle.Trees {
		t.Run(tc.Note, func(t *testing.T) {
			mine := Parse(tc.Source)
			if reflect.DeepEqual(mine, tc.Blocks) {
				return
			}
			// O diff sai em JSON porque a árvore aninhada é ilegível no `%+v`
			// do Go — e um erro que ninguém lê é um erro que vira `-run` de
			// outro teste.
			doJS, _ := json.Marshal(tc.Blocks)
			doGo, _ := json.Marshal(mine)
			t.Errorf("fonte %q\n  o JS dá: %s\n  o Go dá: %s", tc.Source, doJS, doGo)
		})
	}
}

func TestTogglingATaskMatchesTheJs(t *testing.T) {
	oracle := leOOraculoDoMarkdown(t)
	for _, tc := range oracle.Toggle {
		t.Run(tc.Note, func(t *testing.T) {
			if got := ToggleTask(tc.Source, tc.Row, tc.Marked); got != tc.Output {
				t.Errorf("ToggleTask(%q, %d, %v) = %q, o JS dá %q",
					tc.Source, tc.Row, tc.Marked, got, tc.Output)
			}
		})
	}
}

// O oráculo cobre a gramática; o que ele NÃO cobre é o que nenhuma das duas
// telas escreveu ainda — e é aqui que este port pode quebrar sozinho.
//
// `ToggleTask` recebe uma LINHA vinda de um clique do navegador, e o cliente
// pode estar um remendo atrás do servidor. Um índice negativo, ou além do fim, é
// caminho NORMAL e não erro: a resposta certa é devolver a nota intacta, e a
// errada é um `index out of range` derrubando o handler que estava salvando o
// texto de alguém.
func TestTogglingATaskDoesNotPanicOnAnOutOfRangeLine(t *testing.T) {
	note := "- [ ] dar XP"
	for _, row := range []int{-1, 1, 99} {
		if got := ToggleTask(note, row, true); got != note {
			t.Errorf("linha %d mexeu na nota: %q", row, got)
		}
	}
}
