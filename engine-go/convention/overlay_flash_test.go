package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// SOBREPOSIÇÃO ESCONDIDA SÓ POR `data-show` PISCA NA TELA (ALE-296).
//
// O `data-show` é avaliado pelo Datastar DEPOIS que o runtime carrega e
// processa o DOM. Até lá o nó está no documento com `fixed inset-0 z-50 …
// bg-black/60` e PINTA — um pano preto cobrindo a janela inteira. Não é um
// quadro: medido com uma sonda de `requestAnimationFrame` instalada antes de
// qualquer script da página, foram 30 quadros nas Perícias, 13 na Mochila, 10 no
// Combate e 9 nas Magias. A 60fps, meio segundo.
//
// O dono relatou como "um dialog de 1 frame que some" em duas abas. Eram seis
// nós e todas as abas.
//
// # A correção, e por que ela é um `style` ESTÁTICO
//
// `style="display:none"` ao lado do `data-show`: o nó nasce escondido e o
// Datastar troca o `display` para `”` quando o sinal fica verdadeiro. O
// `web/table/notes.templ` já fazia assim, e é o controle positivo de que a
// forma funciona.
//
// **Isto não colide com o `TestNoNodeHasDataShowAndDataAttrStyleTogether`**, e a
// diferença é a issue daquele guarda: o proibido é o `data-attr:style`, que
// REESCREVE o atributo inteiro a cada avaliação e apaga o `display` que o
// `data-show` acabou de pôr. Um `style` estático é lido uma vez, na análise do
// HTML, e é justamente o valor inicial que falta aqui.
//
// # Por que um guarda e não seis correções
//
// Porque a família tem seis irmãos hoje e vai ter mais: um deles é o `templ
// overlay` da casa, com dez call sites. A próxima sobreposição nasce piscando e
// isso não aparece em revisão de diff nenhuma — o defeito é meio segundo numa
// tela que ninguém está olhando com cronômetro.
func TestNoDataShowNodeIsBornVisible(t *testing.T) {
	// O nó ABERTO inteiro, que é onde os atributos moram. `[^<>"]|"[^"]*"`
	// atravessa o valor de um atributo que contenha `>` — e eles contêm: todo
	// `data-on:` deste repositório carrega expressão com `>` e `&&`.
	no := regexp.MustCompile(`<(\w+)((?:[^<>"]|"[^"]*")*?)>`)
	classe := regexp.MustCompile(`class="([^"]*)"`)

	// `data-show="$x"` PURO — sem negação e sem operador — quer dizer "escondido
	// até o sinal ficar verdadeiro", e todos os seis sinais assim deste
	// repositório nascem `false`. Um nó desses que não nasce escondido pinta o
	// próprio conteúdo e some.
	sinalPuro := regexp.MustCompile(`^\$[a-z0-9]+$`)

	var candidatos, arquivosLidos int
	err := filepath.WalkDir("..", func(nome string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			if entrada.Name() == "node_modules" || entrada.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(nome, ".templ") {
			return nil
		}
		arquivosLidos++
		bruto, err := os.ReadFile(nome)
		if err != nil {
			return err
		}
		texto := strings.Join(semComentario(strings.Split(string(bruto), "\n")), "\n")
		for _, m := range no.FindAllStringSubmatchIndex(texto, -1) {
			atributos := texto[m[4]:m[5]]
			if !strings.Contains(atributos, "data-show") {
				continue
			}
			// A CLASSE é opcional, e isto foi um furo: a primeira versão exigia
			// `class=` para julgar qualquer nó, então um `<span data-show="$sound">`
			// sem classe nenhuma — que pisca igual — passava por baixo do guarda.
			// Só a regra da SOBREPOSIÇÃO precisa da classe; a do sinal puro não.
			var tokens []string
			if c := classe.FindStringSubmatch(atributos); c != nil {
				tokens = strings.Fields(c[1])
			}
			// SOBREPOSIÇÃO é o que sai do fluxo E cobre: fora do fluxo sozinho
			// não basta (um crachá `absolute` no canto de um cartão não cobre
			// nada), e é por isso que o `inset-0` ou uma camada `z-` entram na
			// conta.
			cobre := (contem(tokens, "fixed") || contem(tokens, "absolute")) &&
				(contem(tokens, "inset-0") || comPrefixo(tokens, "z-"))
			mostra := regexp.MustCompile(`data-show="([^"]*)"`).FindStringSubmatch(atributos)
			puro := mostra != nil && sinalPuro.MatchString(strings.TrimSpace(mostra[1]))
			if !cobre && !puro {
				continue
			}
			candidatos++
			if strings.Contains(atributos, "display:none") || contem(tokens, "hidden") {
				continue
			}
			linha := strings.Count(texto[:m[0]], "\n") + 1
			t.Errorf("%s:%d — <%s> se esconde SÓ pelo `data-show`: ele nasce visível e pinta até o "+
				"Datastar chegar (medido: até 30 quadros). "+
				"Ponha `style=\"display:none\"` ao lado, como o `web/table/notes.templ` faz.",
				nome, linha, texto[m[2]:m[3]])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR. Sem ele, um regex que parou de casar e um repositório sem
	// sobreposição nenhuma dizem a mesma coisa — e este guarda inteiro é uma
	// afirmação de AUSÊNCIA.
	if arquivosLidos < 40 {
		t.Fatalf("o guarda leu só %d arquivos `.templ`: a caminhada parou de achar as cenas", arquivosLidos)
	}
	if candidatos < 15 {
		t.Fatalf("o guarda achou só %d nós que ele sabe julgar: o padrão parou de casar e o verde não significa nada", candidatos)
	}
}
