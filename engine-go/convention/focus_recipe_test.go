package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// O REALCE DE FOCO SE ESCREVE UMA VEZ, NO `index.css`, E NUNCA NUMA `class=`
// (ALE-317).
//
// # A receita já era global, e a cópia já era inerte
//
// A regra da casa mora no `index.css` desde a ALE-173 (P4) e vale para tudo que
// recebe foco dentro de uma cena. Ela não é layerada, então ganha de qualquer
// utilitário do Tailwind — e isso quer dizer que
// `focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring`
// escrito num `class=` **não muda um pixel**. Medido no navegador antes da
// varredura: um botão que pede `outline-offset-2` computa `outline-offset: 1px`.
//
// Eram 242 sítios — 240 em 43 `.templ` e 2 em `.go` de produção —, e a issue os
// contava como 147 porque a primeira medição olhou só `<button>`: a mesma tripla
// estava em `<a>`, `<input>`, `<summary>` e `<label>`. *Uma medição parcial não é
// um número menor, é um número de outra pergunta.*
//
// # Por que um guarda, e não só a varredura
//
// Este é o irmão do `TestNoHandwrittenLabelRecipe`, e existe pela mesma razão
// que o `CLAUDE.md` da raiz chama de "o molde de como esta família se conserta":
// visitar mais cenas é ENUMERAÇÃO, e a cena que nascer amanhã nasce sem medição.
// Um guarda que pergunta "alguém escreveu a receita à mão?" vale para ela.
//
// A cópia é pior que inerte, e vale saber por quê antes de reescrevê-la um dia:
// a mesma receita traz `outline-none`, que escreve `--tw-outline-style: none`, e
// o `focus-visible:outline-2` do Tailwind v4 é `outline-style:
// var(--tw-outline-style); outline-width: 2px`. Os dois na mesma tag produzem
// contorno de estilo `none` — largura 2px que não desenha nada. Onde a regra
// global alcança, isso não aparece; fora dela, o botão fica SEM realce.
//
// # O que este guarda NÃO recusa, e não é fresta
//
// `has-[:focus-visible]:outline-*` fica, e é a única receita de foco legítima
// fora do `index.css`: as três plaquetas de rádio (as duas da forja e a de
// entrar na mesa) escondem o `<input>` com `sr-only`, e quem tem de acender é o
// `<label>`. A regra global casa `:is(a, button, input, …)` e não o alcança;
// uma regra `label:has(:focus-visible)` no lugar dela desenharia dois anéis
// concêntricos nos dez rótulos do app que envolvem um campo VISÍVEL.
//
// Elas são o DENOMINADOR deste guarda, e por isso não estão numa lista de
// permitidos: o padrão que as acha é o mesmo que acha a violação, então achá-las
// prova que o scanner enxerga o token. Uma lista vazia de falhas e um regex que
// parou de casar se parecem no terminal.
//
// As outras variantes de `focus-visible:` — `opacity-80`, `bg-accent`,
// `bg-grimorio-gold` — não são deste guarda: elas mudam a PEÇA no foco, não o
// anel, e a regra global não tem opinião sobre isso.
func TestNoHandwrittenFocusRing(t *testing.T) {
	// O token só é receita quando começa a classe. `has-[:focus-visible]:…` é
	// precedido por `]` e cai fora, que é exatamente a intenção.
	receita := regexp.MustCompile(`(^|[\s"'` + "`" + `])(focus-visible:outline-[a-z0-9-]+)`)
	excecao := regexp.MustCompile(`has-\[:focus-visible\]:outline-[a-z0-9-]+`)

	var arquivosLidos, excecoes int
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
		// O `_templ.go` é derivado do `.templ` e acusaria o mesmo sítio duas
		// vezes; o `_test.go` cita a grafia proibida ao explicá-la.
		gerado := strings.HasSuffix(nome, "_templ.go") || strings.HasSuffix(nome, "_test.go")
		if gerado || (!strings.HasSuffix(nome, ".templ") && !strings.HasSuffix(nome, ".go")) {
			return nil
		}
		arquivosLidos++
		bruto, err := os.ReadFile(nome)
		if err != nil {
			return err
		}
		// COMENTÁRIO FORA ANTES DE MEDIR, pela razão que os guardas irmãos
		// registram: o comentário que explica a receita cita a grafia que ela
		// substitui, e um guarda que lê a fonte crua acusa a explicação.
		for i, linha := range semComentario(strings.Split(string(bruto), "\n")) {
			excecoes += len(excecao.FindAllString(linha, -1))
			for _, achado := range receita.FindAllStringSubmatch(linha, -1) {
				t.Errorf("%s:%d — `%s` escrito à mão, e ele não faz efeito nenhum.\n"+
					"    O realce de foco é GLOBAL e mora no `api/assets/src/index.css`; a regra de lá não é layerada,\n"+
					"    então ela ganha do utilitário e o que está na `class=` é decoração. APAGUE a classe.\n"+
					"    (Realce próprio de verdade, quando existir, é regra no `index.css` — e o\n"+
					"    `e2e/tests/support/focus.ts` cobra que a casa continue tendo uma cara só.)",
					nome, i+1, achado[2])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, em três metades, e nenhuma é enfeite.
	//
	// A primeira: sem arquivo lido, tudo acima é verde sobre nada — foi assim que
	// um guarda irmão quase passou medindo um diretório que tinha esvaziado.
	if arquivosLidos < 200 {
		t.Fatalf("o guarda leu só %d arquivos `.templ`/`.go`: a caminhada parou de achar as cenas", arquivosLidos)
	}
	// A segunda: as três plaquetas de rádio são o controle POSITIVO do padrão. Se
	// o regex parar de casar `focus-visible:outline-…`, elas somem primeiro — e a
	// ausência delas denuncia antes que a ausência de falhas engane.
	// São nove classes hoje — três plaquetas escrevendo `outline-2`,
	// `outline-offset-1` e `outline-ring` cada uma. O piso é três e não nove
	// porque o número é de DESENHO e pode mudar; o que não pode é ir a zero.
	if excecoes < 3 {
		t.Fatalf("o guarda achou só %d classes `has-[:focus-visible]:outline-…`, e as três plaquetas de rádio "+
			"escrevem nove: o padrão parou de casar e a lista de falhas vazia não é evidência de nada", excecoes)
	}
	// A terceira, e a mais importante: este guarda proíbe a receita à mão porque
	// existe uma GLOBAL. No dia em que alguém apagar a global, proibir a cópia
	// deixa o app inteiro sem realce — e o guarda precisa falhar junto, em vez de
	// seguir cobrando uma regra cuja razão sumiu.
	folha, err := os.ReadFile("../serve/api/assets/src/index.css")
	if err != nil {
		t.Fatalf("ler a folha-fonte: %v", err)
	}
	// O padrão é a DECLARAÇÃO e não "algum `:focus-visible` seguido de outline":
	// a regra do trilho (`[data-nav-region] … { outline: none }`) casaria com essa
	// forma frouxa e o controle passaria com a receita apagada.
	global := regexp.MustCompile(`:focus-visible \{\n\s*outline: 2px solid var\(--grimorio-gold\);\n\s*outline-offset: 1px;`)
	if !global.Match(folha) {
		t.Fatalf("a receita GLOBAL de foco sumiu do `index.css`: sem ela, proibir a receita à mão " +
			"deixa o app sem realce nenhum, e este guarda estaria cobrando uma regra sem razão")
	}
	t.Logf("%d arquivos lidos, %d classes de plaqueta de rádio com receita própria e declarada", arquivosLidos, excecoes)
}
