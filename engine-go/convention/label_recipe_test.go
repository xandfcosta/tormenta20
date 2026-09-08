package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A FAMÍLIA DE RÓTULOS EM CAIXA ALTA SE ESCREVE PELA RECEITA, NUNCA À MÃO
// (ALE-295).
//
// # Por que este guarda existe, e por que ele é de VARREDURA
//
// A ALE-173 mediu 208 ocorrências escritas de 59 jeitos e descobriu que não eram
// 59 variações da mesma coisa: eram TRÊS papéis, cada um escrito de vinte jeitos
// porque não havia de onde copiar. A resposta foi o componente, e ele cobria as
// 43 telas da SPA por AMOSTRAGEM — o guarda de tipografia media uma tela e valia
// para todas, porque todas passavam pelas mesmas peças.
//
// A migração para templ destruiu a amostragem sem que ninguém decidisse isso:
// cena em templ escreve a classe à mão, então a sentinela perdeu de vista
// exatamente o código novo. Foi assim que quatro violações do piso da Cinzel
// viveram em três cenas com o guarda no ar (ALE-252), e quando o medidor passou
// a visitar quinze cenas em vez de uma ele achou dez sítios de uma vez.
//
// Visitar mais cenas é ENUMERAÇÃO: uma linha por cena, para sempre, e a que
// alguém esquecer nasce sem medição. **Este guarda é o que devolve a
// amostragem** — ele não pergunta "esta cena está na lista?", pergunta "alguém
// escreveu a receita à mão?", e a resposta vale para a cena que nascer amanhã.
//
// # O que está FORA, e não é esquecimento
//
// Três papéis têm receita (`ui.SectionTitleClasses`, `ui.SectionLabelClasses`,
// `ui.FieldLabelClasses`) e são cobrados aqui. Ficam de fora:
//
//   - o CRACHÁ — `<span>` com caixa (borda, fundo ou canto). Ele não é da
//     família: a caixa é o que ele diz, e a SPA nunca teve receita para ele.
//     **Ele ganhou a própria na ALE-177** (`ui.BadgeClasses`, cobrada pelo
//     `TestNoHandwrittenBadgeRecipe`), e ela é de GEOMETRIA e não de tipografia:
//     o piso de 24px do WCAG 2.5.8. As duas convivem sem sobrepor — um crachá
//     que também escreve rótulo em caixa alta continua fora desta família.
//   - o CONTROLE — `<button>`/`<summary>`, cuja tipografia é do `ui.Button`.
//   - a NAVEGAÇÃO — `<a>`, um sítio só.
//   - o TÍTULO DE PALCO — o que declara tamanho RESPONSIVO (`sm:text-*`). São
//     três: o nome do herói e o "Forjar um herói" no palco de personagens, e o
//     nome da campanha no palco de campanhas. Eles medem 24px e crescem até 36,
//     e a primeira versão desta varredura os tratou como título de seção —
//     forçando 18px de base, que ENCOLHE o nome do herói no telefone. Não é
//     descuido deles: este papel **não existia na SPA**, foi inventado na era
//     templ, e por isso não há o que portar.
//
// São dezenove sítios, e eles seguem à mão de propósito: inventar receita para
// eles aqui seria fazer exatamente o que a ALE-252 puniu — escrever a receita em
// vez de portá-la. Quando algum deles ganhar uma, ele entra nesta lista.
func TestNoHandwrittenLabelRecipe(t *testing.T) {
	classe := regexp.MustCompile(`class="([^"]*)"`)
	abertura := regexp.MustCompile(`^<([a-zA-Z][a-zA-Z0-9]*)`)
	receita := regexp.MustCompile(`ui\.(SectionTitle|SectionLabel|FieldLabel)Classes\(|@ui\.(SectionLabel|SectionCaption)\(`)

	// Os elementos cujo papel TEM receita. O `<span>`/`<p>` entra pelo que ele
	// NÃO tem: sem caixa, ele é rótulo; com caixa, é crachá.
	titulo := map[string]bool{"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "th": true, "caption": true}
	campo := map[string]bool{"label": true, "legend": true, "dt": true}
	solto := map[string]bool{"span": true, "p": true, "li": true, "div": true}

	var aMao, pelaReceita, arquivosLidos int
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
			tokens := strings.Fields(texto[achado[2]:achado[3]])
			if !contem(tokens, "uppercase") || !comPrefixo(tokens, "tracking-") {
				continue
			}
			// O `<` mais próximo ANTES do atributo é o que abre o elemento que o
			// carrega — a posição do caractere é exata onde a linha não é: o
			// atributo mora numa linha própria em metade dos sítios, e procurar a
			// tag "na mesma linha" ou "na linha de cima" acerta em uns e erra em
			// outros sem dizer quais.
			corte := strings.LastIndex(texto[:achado[0]], "<")
			if corte < 0 {
				continue
			}
			m := abertura.FindStringSubmatch(texto[corte:min(corte+24, len(texto))])
			if m == nil {
				continue
			}
			tag := m[1]
			temCaixa := comPrefixo(tokens, "rounded") || comPrefixo(tokens, "border") || comPrefixo(tokens, "bg-")
			// TAMANHO RESPONSIVO é a marca do título de PALCO, e não um detalhe:
			// uma receita escreve UM tamanho, e um título que cresce com a janela
			// não cabe em nenhuma das três. Ver a docstring.
			responsivo := comPrefixo(tokens, "sm:text-") || comPrefixo(tokens, "lg:text-")
			var papel string
			switch {
			case responsivo:
				aMao++
				continue
			case titulo[tag]:
				papel = "ui.SectionTitleClasses(contexto, tom, extra)"
			case campo[tag] || (solto[tag] && !temCaixa):
				papel = "ui.FieldLabelClasses(tom, extra)"
			default:
				aMao++ // crachá, controle e navegação: fora da família, ver a docstring.
				continue
			}
			linha := strings.Count(texto[:achado[0]], "\n") + 1
			t.Errorf("%s:%d — <%s> escreve a receita de rótulo à mão (%s). Use %s.\n"+
				"    Escrever a receita em vez de chamá-la é o que produziu 59 grafias na SPA e 30 nas cenas em templ,\n"+
				"    e é o que tira a cobertura do guarda de tipografia da AMOSTRAGEM para a ENUMERAÇÃO.",
				nome, linha, tag, texto[achado[2]:achado[3]], papel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, em três metades, e nenhuma delas é enfeite.
	//
	// A primeira: sem arquivo lido, tudo abaixo é verde sobre nada — foi assim
	// que o guarda irmão quase passou medindo um diretório que tinha esvaziado.
	if arquivosLidos < 40 {
		t.Fatalf("o guarda leu só %d arquivos `.templ`: a caminhada parou de achar as cenas", arquivosLidos)
	}
	// A segunda: se o casamento da RECEITA parar de funcionar, o guarda continua
	// verde porque ninguém a estaria usando — e "ninguém usa" e "não sei
	// procurar" se parecem no terminal.
	if pelaReceita < 100 {
		t.Fatalf("só %d sítios chamam uma das três receitas: ou elas foram desfeitas, ou o padrão parou de casar", pelaReceita)
	}
	// A terceira: se o casamento do `class="…"` parar, o laço acima nunca entra e
	// nada é cobrado. Os dezesseis de fora da família são a prova de que ele
	// entra — eles TÊM de continuar sendo achados.
	if aMao < 10 {
		t.Fatalf("o guarda achou só %d rótulos escritos à mão fora da família: o padrão de `class=` parou de casar "+
			"e este guarda deixou de cobrar qualquer coisa", aMao)
	}
}

func contem(tokens []string, alvo string) bool {
	for _, t := range tokens {
		if t == alvo {
			return true
		}
	}
	return false
}

func comPrefixo(tokens []string, prefixo string) bool {
	for _, t := range tokens {
		if strings.HasPrefix(t, prefixo) {
			return true
		}
	}
	return false
}
