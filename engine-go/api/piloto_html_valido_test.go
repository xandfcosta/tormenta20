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
// `@ui.SectionLabel`. Resultado na página — o `h4` virou filho da `<section>`,
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
	abre := regexp.MustCompile(`@ui.SectionLabel\([^)]*\)\s*\{`)
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
		// COMENTÁRIO FORA ANTES DE MEDIR. O comentário do próprio
		// `rotuloDeSecao` cita `<h4>` para explicar por que ele não pode entrar
		// ali, e um guarda que lê a fonte crua acusaria a explicação do
		// defeito como se fosse o defeito. A sessão irmã pagou exatamente isso
		// hoje num guarda irmão: ele nasceu VERMELHO sobre o próprio texto, e
		// só a prova de vermelho revelou — sem ela, entregaria um teste que
		// falha sobre si mesmo para sempre, e o próximo o desligaria.
		linhas := semComentario(strings.Split(string(conteudo), "\n"))
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
					t.Errorf("%s:%d — `%s` dentro de @ui.SectionLabel. O `<p>` dele não aceita "+
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
		t.Fatal("o guarda não achou nenhum uso de @ui.SectionLabel: o padrão parou de casar " +
			"e o verde não significa nada")
	}
	t.Logf("%d usos de @ui.SectionLabel visitados", visitados)
}

// semComentario corta o que vier depois de `//` em cada linha. Grosseiro de
// propósito: `//` dentro de string literal viraria corte indevido, mas em
// template isso é raro e o custo do erro é um falso NEGATIVO — o guarda deixa
// passar —, não um falso positivo que faz alguém desligá-lo.
func semComentario(linhas []string) []string {
	fora := make([]string, len(linhas))
	for i, l := range linhas {
		if j := strings.Index(l, "//"); j >= 0 {
			l = l[:j]
		}
		fora[i] = l
	}
	return fora
}
