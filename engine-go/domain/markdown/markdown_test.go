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
	Arvores []struct {
		Nota   string  `json:"nota"`
		Fonte  string  `json:"fonte"`
		Blocos []Block `json:"blocos"`
	} `json:"arvores"`
	Alterna []struct {
		Nota    string `json:"nota"`
		Fonte   string `json:"fonte"`
		Linha   int    `json:"linha"`
		Marcada bool   `json:"marcada"`
		Saida   string `json:"saida"`
	} `json:"alterna"`
}

func leOOraculoDoMarkdown(t *testing.T) oraculoDoMarkdown {
	t.Helper()
	bruto, err := os.ReadFile("testdata/markdown-from-the-js.json")
	if err != nil {
		t.Fatalf("oráculo ausente — ele é versionado e não se regenera mais: %v", err)
	}
	var o oraculoDoMarkdown
	if err := json.Unmarshal(bruto, &o); err != nil {
		t.Fatalf("oráculo ilegível: %v", err)
	}
	// O CONTROLE: um oráculo vazio faria todos os laços abaixo passarem verde
	// sem comparar nada, que é o formato exato do teste que não mede — o
	// arquivo pode existir e estar vazio se o script mudar de forma.
	if len(o.Arvores) == 0 || len(o.Alterna) == 0 {
		t.Fatal("o oráculo está vazio — os laços abaixo passariam verde sem comparar nada")
	}
	return o
}

func TestTheNoteMarkdownMatchesTheJs(t *testing.T) {
	oraculo := leOOraculoDoMarkdown(t)
	for _, caso := range oraculo.Arvores {
		t.Run(caso.Nota, func(t *testing.T) {
			meu := Parse(caso.Fonte)
			if reflect.DeepEqual(meu, caso.Blocos) {
				return
			}
			// O diff sai em JSON porque a árvore aninhada é ilegível no `%+v`
			// do Go — e um erro que ninguém lê é um erro que vira `-run` de
			// outro teste.
			doJS, _ := json.Marshal(caso.Blocos)
			doGo, _ := json.Marshal(meu)
			t.Errorf("fonte %q\n  o JS dá: %s\n  o Go dá: %s", caso.Fonte, doJS, doGo)
		})
	}
}

func TestTogglingATaskMatchesTheJs(t *testing.T) {
	oraculo := leOOraculoDoMarkdown(t)
	for _, caso := range oraculo.Alterna {
		t.Run(caso.Nota, func(t *testing.T) {
			if got := ToggleTask(caso.Fonte, caso.Linha, caso.Marcada); got != caso.Saida {
				t.Errorf("ToggleTask(%q, %d, %v) = %q, o JS dá %q",
					caso.Fonte, caso.Linha, caso.Marcada, got, caso.Saida)
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
	nota := "- [ ] dar XP"
	for _, linha := range []int{-1, 1, 99} {
		if got := ToggleTask(nota, linha, true); got != nota {
			t.Errorf("linha %d mexeu na nota: %q", linha, got)
		}
	}
}
