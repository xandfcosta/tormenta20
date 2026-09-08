package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// O CRACHÁ SE ESCREVE PELA RECEITA, NUNCA À MÃO (ALE-177).
//
// # O defeito que ele prende, e por que ele é de VARREDURA
//
// A ALE-177 dizia que 56% dos alvos da ficha reprovavam o WCAG 2.5.8 no
// telefone. Medido na ficha em Datastar — 390px, as sete abas, um herói que
// conjura e um que não —, **reprovavam quatro**, e os quatro eram a mesma peça
// escrita à mão em lugares diferentes: a pílula pequena que liga e desliga, com
// 21px de altura.
//
// O número não era 4 por sorte. Um sexto sítio, o crachá de aprimoramento das
// Magias, JÁ tinha ganhado `min-h-7` — alguém consertou um e não varreu os
// irmãos, que é exatamente como esta família anda. Os outros cinco continuaram
// a 21px por mais tempo do que qualquer um teria adivinhado, e três deles só
// reprovam quando o conteúdo ao redor aperta: a exceção de ESPAÇAMENTO da norma
// os perdoava enquanto a mochila estivesse vazia.
//
// É por isso que o conserto não podia ser "aumentar os quatro". Alvo que passa
// por folga volta a reprovar quando a folga some, e nada avisa.
//
// # O que este guarda cobra
//
// Elemento INTERATIVO com a forma da pílula — `rounded-full`, uma borda e um
// tamanho de letra da escala miúda — tem de vir do `ui.BadgeClasses`, que carrega
// o piso de 24px. O `<span>` decorativo de mesma aparência não é cobrado: ele
// não é alvo de nada, e a norma fala de alvos.
//
// Ele é o irmão do `TestNoHandwrittenLabelRecipe`, e pela mesma razão: visitar
// mais telas é ENUMERAÇÃO, e o que devolve a AMOSTRAGEM é perguntar "alguém
// escreveu a receita à mão?" em vez de "esta tela está na lista?".
func TestNoHandwrittenBadgeRecipe(t *testing.T) {
	classe := regexp.MustCompile(`class=(?:"([^"]*)"|\{ "([^"]*)")`)
	abertura := regexp.MustCompile(`^<([a-zA-Z][a-zA-Z0-9]*)`)
	receita := regexp.MustCompile(`ui\.BadgeClasses\(`)

	// O ALVO é o que o dedo aperta. `div` e `span` de mesma aparência ficam de
	// fora porque a norma mede ALVO, e um enfeite não é um.
	alvo := map[string]bool{"button": true, "a": true, "select": true, "summary": true, "input": true, "textarea": true}

	var aMao, pelaReceita, arquivosLidos, atributosLidos int
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
		// O KIT é onde a receita MORA: cobrá-la lá seria cobrar a definição.
		if !strings.HasSuffix(nome, ".templ") || strings.Contains(nome, "/web/ui/") {
			return nil
		}
		arquivosLidos++
		bruto, err := os.ReadFile(nome)
		if err != nil {
			return err
		}
		// COMENTÁRIO FORA ANTES DE MEDIR, pela razão que o guarda irmão registra:
		// o comentário de uma receita cita a grafia que ela substitui, e um guarda
		// que lê a fonte crua acusa a explicação do defeito como se fosse o
		// defeito.
		texto := strings.Join(semComentario(strings.Split(string(bruto), "\n")), "\n")
		pelaReceita += len(receita.FindAllString(texto, -1))
		for _, achado := range classe.FindAllStringSubmatchIndex(texto, -1) {
			atributosLidos++
			// O `class` do templ tem duas formas — `class="…"` e `class={ "…",
			// templ.KV(…) }` —, e o segundo grupo é o da segunda.
			cru := ""
			if achado[2] >= 0 {
				cru = texto[achado[2]:achado[3]]
			} else if achado[4] >= 0 {
				cru = texto[achado[4]:achado[5]]
			}
			tokens := strings.Fields(cru)
			if !contem(tokens, "rounded-full") || !comPrefixo(tokens, "border") {
				continue
			}
			// A ESCALA MIÚDA é o que separa o crachá do resto do redondo: a barra
			// de progresso da mochila e o monograma do Hub também são
			// `rounded-full` e não são pílula de texto.
			if !contem(tokens, "text-3xs") && !contem(tokens, "text-2xs") {
				continue
			}
			// O `<` mais próximo ANTES do atributo é o que abre o elemento que o
			// carrega — a posição do caractere é exata onde a linha não é: o
			// atributo mora numa linha própria em quase todos os sítios.
			corte := strings.LastIndex(texto[:achado[0]], "<")
			if corte < 0 {
				continue
			}
			m := abertura.FindStringSubmatch(texto[corte:min(corte+24, len(texto))])
			if m == nil {
				continue
			}
			if !alvo[m[1]] {
				aMao++ // enfeite de mesma aparência: fora da família, ver a docstring.
				continue
			}
			linha := strings.Count(texto[:achado[0]], "\n") + 1
			t.Errorf("%s:%d — <%s> escreve a receita do crachá à mão (%s). Use ui.BadgeClasses(extra).\n"+
				"    Sem o piso de 24px ele reprova o WCAG 2.5.8 assim que o conteúdo ao redor apertar — e a\n"+
				"    exceção de espaçamento o perdoa até lá, então o defeito nasce mudo.",
				nome, linha, m[1], cru)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, em três metades, e nenhuma delas é enfeite.
	//
	// A primeira: sem arquivo lido, tudo acima é verde sobre nada.
	if arquivosLidos < 40 {
		t.Fatalf("o guarda leu só %d arquivos `.templ`: a caminhada parou de achar as cenas", arquivosLidos)
	}
	// A segunda: se o casamento do `class=` parar de funcionar, o laço nunca
	// entra e nada é cobrado — verde idêntico ao de "está tudo certo".
	if atributosLidos < 300 {
		t.Fatalf("o guarda leu só %d atributos `class`: o padrão parou de casar e ele deixou de cobrar qualquer coisa", atributosLidos)
	}
	// A terceira: se a RECEITA for desfeita ou renomeada, ninguém a estaria
	// usando — e "ninguém usa" e "não sei procurar" se parecem no terminal.
	if pelaReceita < 6 {
		t.Fatalf("só %d sítios chamam `ui.BadgeClasses`: ou ela foi desfeita, ou o padrão parou de casar", pelaReceita)
	}
	// E a prova de que o ramo do enfeite é alcançado: os `<span>` redondos da
	// escala miúda (o círculo da magia, o crachá do item guardado) continuam
	// existindo e TÊM de ser achados.
	if aMao < 2 {
		t.Fatalf("o guarda achou só %d enfeites redondos fora da família: o ramo que os separa do alvo não está sendo alcançado", aMao)
	}
}
