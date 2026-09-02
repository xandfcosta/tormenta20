package api

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// O ORÁCULO DO MARKDOWN (ALE-269).
//
// Ele nasceu como PARIDADE com o `markdown.ts` da SPA: as duas telas desenhavam
// a mesma nota do banco, e a gramática tem divergências deliberadas do markdown
// padrão que uma biblioteca desfaria — a quebra de linha da ALE-122 é a maior.
// O esperado era MEDIDO rodando o TypeScript de verdade, e não digitado aqui,
// para não ser uma segunda transcrição da gramática.
//
// Com a SPA apagada (ALE-272, fatia 10c) não há segundo lado: o script que
// gerava o arquivo saiu, e o oráculo virou uma LINHA DE BASE congelada — ele
// acusa qualquer árvore que mude sem ter sido pedido, e deixou de provar que
// duas implementações concordam. É a mesma perda que o `genoracle` documenta no
// `engine-go/CLAUDE.md`, e a mitigação é a mesma: o diff de um oráculo se
// revisa contra o que se queria mudar, nunca se aceita porque "ficou verde".
//
// O oráculo é comparado como ÁRVORE e não como texto JSON: chave fora de ordem
// ou campo omitido são detalhe de serialização, e um teste que os prendesse
// falharia por motivo errado.

// oraculoDoMarkdown é a forma do arquivo gerado pelo script.
type oraculoDoMarkdown struct {
	Arvores []struct {
		Nota   string    `json:"nota"`
		Fonte  string    `json:"fonte"`
		Blocos []mdBloco `json:"blocos"`
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
	bruto, err := os.ReadFile("testdata/markdown-do-js.json")
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
			meu := parseNota(caso.Fonte)
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
			if got := alternaTarefa(caso.Fonte, caso.Linha, caso.Marcada); got != caso.Saida {
				t.Errorf("alternaTarefa(%q, %d, %v) = %q, o JS dá %q",
					caso.Fonte, caso.Linha, caso.Marcada, got, caso.Saida)
			}
		})
	}
}

// O oráculo cobre a gramática; o que ele NÃO cobre é o que nenhuma das duas
// telas escreveu ainda — e é aqui que este port pode quebrar sozinho.
//
// `alternaTarefa` recebe uma LINHA vinda de um clique do navegador, e o cliente
// pode estar um remendo atrás do servidor. Um índice negativo, ou além do fim,
// é caminho NORMAL e não erro: a resposta certa é devolver a nota intacta, e a
// errada é entrar em pânico e derrubar o handler que estava salvando o texto de
// alguém. O JS devolve `undefined` do array e cai no mesmo lugar; em Go isso é
// um `index out of range`, e por isso a guarda existe e é testada.
func TestTogglingATaskDoesNotPanicOnAnOutOfRangeLine(t *testing.T) {
	nota := "- [ ] dar XP"
	for _, linha := range []int{-1, 1, 99} {
		if got := alternaTarefa(nota, linha, true); got != nota {
			t.Errorf("linha %d mexeu na nota: %q", linha, got)
		}
	}
}
