package api

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

// A PARIDADE DO MARKDOWN COM O JS (ALE-269).
//
// Este é o guarda que justifica o port ter sido escrito à mão em vez de o
// projeto puxar um goldmark: enquanto a migração durar, as DUAS telas desenham a
// MESMA nota do banco, e a gramática da SPA tem divergências deliberadas do
// markdown padrão que uma biblioteca desfaria — a quebra de linha da ALE-122 é
// a maior delas.
//
// O ESPERADO NÃO É DIGITADO AQUI, e essa é a diferença entre paridade e uma
// segunda transcrição. Ele é MEDIDO rodando o `markdown.ts` de verdade:
//
//	node scripts/dump-markdown-oracle.ts
//
// Escrever as árvores à mão em Go seria reescrever a gramática pela segunda vez,
// e a segunda transcrição é exatamente onde as duas telas passam a discordar em
// silêncio — que é a família de defeito que este repositório mais paga caro.
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
		t.Fatalf("oráculo ausente (rode `node scripts/dump-markdown-oracle.ts`): %v", err)
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

func TestOMarkdownDaNotaCasaComOJS(t *testing.T) {
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

func TestAlternarTarefaCasaComOJS(t *testing.T) {
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
func TestAlternarTarefaNaoEntraEmPanicoComLinhaDeFora(t *testing.T) {
	nota := "- [ ] dar XP"
	for _, linha := range []int{-1, 1, 99} {
		if got := alternaTarefa(nota, linha, true); got != nota {
			t.Errorf("linha %d mexeu na nota: %q", linha, got)
		}
	}
}
