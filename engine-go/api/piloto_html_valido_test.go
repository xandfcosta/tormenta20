package api

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// CONTEÚDO DE FLUXO DENTRO DE `<p>` É HTML INVÁLIDO, E O NAVEGADOR NÃO RECLAMA
// (ALE-262).
//
// Ele CONSERTA: expulsa o elemento do parágrafo, deixa um `<p>` vazio para trás,
// e a classe que estava no parágrafo não alcança mais o conteúdo. O sintoma é
// tipografia errada numa cena que compila, passa no typecheck e passa nos
// guardas de contraste — porque parágrafo vazio não tem texto para medir.
//
// Medido: o `secaoDoBloco` do bestiário punha um `<h4>` dentro do
// `@rotuloDeSecao`. Resultado na página — o `h4` virou filho da `<section>`,
// `text-transform: none`, e VINTE E QUATRO parágrafos vazios na árvore de
// acessibilidade.
//
// O guarda é grep, como o dos ícones, e cobre a mesma classe de defeito: uma
// peça que nasce errada sem ninguém reclamar. Quem precisa da receita num
// cabeçalho usa `classesDoRotulo` no próprio elemento.
func TestNadaDeConteudoDeFluxoDentroDeRotuloDeSecao(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	// O bloco do componente e as duas linhas seguintes: é onde o filho entra.
	abre := regexp.MustCompile(`@rotuloDeSecao\([^)]*\)\s*\{`)
	fluxo := regexp.MustCompile(`<(h[1-6]|div|p|ul|ol|dl|section|article|table|form|fieldset)[\s>]`)

	var visitados int
	for _, f := range arquivos {
		nome := f.Name()
		if !strings.HasSuffix(nome, ".templ") {
			continue
		}
		conteudo, err := os.ReadFile(nome)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		linhas := strings.Split(string(conteudo), "\n")
		for i, linha := range linhas {
			if !abre.MatchString(linha) {
				continue
			}
			visitados++
			// Até o fecho do bloco, ou 6 linhas — o rótulo é curto por natureza.
			for j := i + 1; j < len(linhas) && j <= i+6; j++ {
				if strings.TrimSpace(linhas[j]) == "}" {
					break
				}
				if m := fluxo.FindString(linhas[j]); m != "" {
					t.Errorf("%s:%d — `%s` dentro de @rotuloDeSecao. O `<p>` dele não aceita "+
						"conteúdo de fluxo: o navegador expulsa o elemento e a classe fica "+
						"num parágrafo vazio. Use `classesDoRotulo` no próprio elemento.",
						nome, j+1, strings.TrimSpace(m))
				}
			}
		}
	}
	// CONTROLE: sem isto, um regex que deixou de casar passaria como "nenhuma
	// violação" — o guarda diria verde por não ter visitado nada.
	if visitados == 0 {
		t.Fatal("o guarda não achou nenhum uso de @rotuloDeSecao: o padrão parou de casar " +
			"e o verde não significa nada")
	}
	t.Logf("%d usos de @rotuloDeSecao visitados", visitados)
}
