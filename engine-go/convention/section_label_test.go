package convention

import (
	"io/fs"
	"os"
	"path/filepath"
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
// # Por que ele mora AQUI e não no pacote da cena
//
// Ele nasceu num `piloto_html_valido_test` do `api` e varria `os.ReadDir(".")` — o
// PRÓPRIO diretório. Isso estava certo enquanto todas as cenas eram um pacote
// só; na ALE-278 as campanhas viraram `web/campaigns` e sobraram **ZERO** usos
// de `@ui.SectionLabel` no `api`. Os oito arquivos que usam o componente hoje
// estão todos em `web/*`.
//
// É o SEGUNDO guarda desta família na mesma issue — o outro é o
// `TestNoFocusAsksTheServerWithoutAKeyboardGuard` —, e a diferença entre os dois
// é a lição: **aquele não tinha piso de arquivos visitados e teria passado verde
// medindo metade; este tinha `visitados == 0` e falhou alto no instante em que
// deixou de medir.** O controle não é enfeite; é o que transforma "não mediu"
// em vermelho.
//
// O conserto é o mesmo do guarda de tinta: caminhada, não lista — nem lista de
// cenas, nem o diretório em que o arquivo por acaso mora.
//
// Medido: o `blockSection` do bestiário punha um `<h4>` dentro do
// `@ui.SectionLabel`. Resultado na página — o `h4` virou filho da `<section>`,
// `text-transform: none`, e VINTE E QUATRO parágrafos vazios na árvore de
// acessibilidade.
//
// O guarda é grep, como o dos ícones, e cobre a mesma classe de defeito: uma
// peça que nasce errada sem ninguém reclamar. Quem precisa da receita num
// cabeçalho usa `classesDoRotulo` no próprio elemento.
func TestNoFlowContentInsideASectionLabel(t *testing.T) {
	// O bloco do componente e as duas linhas seguintes: é onde o filho entra.
	abre := regexp.MustCompile(`@ui.SectionLabel\([^)]*\)\s*\{`)
	fluxo := regexp.MustCompile(`<(h[1-6]|div|p|ul|ol|dl|section|article|table|form|fieldset)[\s>]`)

	var visitados, arquivosLidos int
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
		conteudo, err := os.ReadFile(nome)
		if err != nil {
			return err
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
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, em duas metades.
	//
	// A primeira é a que já existia e é a que FALHOU quando as campanhas
	// mudaram de pacote: sem nenhum uso encontrado, um regex que parou de casar
	// e um diretório que ficou vazio dizem a mesma coisa.
	if visitados == 0 {
		t.Fatal("o guarda não achou nenhum uso de @ui.SectionLabel: o padrão parou de casar " +
			"e o verde não significa nada")
	}
	// A segunda é nova, e é a lição do guarda irmão: a caminhada pode ENCOLHER
	// sem zerar. Um piso de arquivos lidos denuncia a raiz trocada.
	if arquivosLidos < 40 {
		t.Fatalf("a caminhada leu só %d arquivos `.templ`, e o repositório tem dezenas: "+
			"a raiz da varredura é o primeiro suspeito", arquivosLidos)
	}
	t.Logf("%d usos de @ui.SectionLabel em %d arquivos", visitados, arquivosLidos)
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
