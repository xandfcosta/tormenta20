package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A REGRA DE IDIOMA NUNCA TEVE VARREDURA, e por isso ela valia exatamente nos
// arquivos que alguém apontou (ALE-300).
//
// O `CLAUDE.md` da raiz diz, com todas as letras, que identificador é em INGLÊS
// — variável, função, tipo, método, campo, constante, pacote, arquivo, nome de
// teste, e componente `templ`. Só uma metade tinha guarda: o
// `TestEveryTestNameIsEnglish`. O resultado foi medido quando o dono perguntou
// se eu estava escrevendo em português: **39 identificadores novos em sete
// fatias seguidas**, todos em português, ao lado de nomes de teste 100% em
// inglês no MESMO commit. A diferença entre as duas metades não foi cuidado,
// foi varredura.
//
// ESTE GUARDA NÃO É AMOSTRAGEM, É CATRACA, e a escolha é de olhos abertos. Havia
// 393 identificadores em português no dia em que ele nasceu, e varrer 393 nomes
// COM CHAMADOR num commit é justamente a varredura em massa que o guia diz que
// não cabe — foi por isso que a ALE-282, só nomes de teste, teve issue própria:
// um `TestX` não tem chamador e cabia num commit; `escrevePagina` não cabe.
//
// Então a linha de base registra a dívida e SÓ PODE ENCOLHER: nome novo em
// português nasce vermelho com o nome dele na mensagem, e nome baselinado que
// deixou de existir também reprova — senão o arquivo vira mentira sozinho, que é
// o defeito descrito na seção "Documentação" do guia.
const portugueseBaseline = "testdata/portuguese_identifiers.txt"

// portugueseWords são marcadores INEQUÍVOCOS, e a lista é curta de propósito.
//
// FORA dela ficam `a`, `o`, `do`, `no`, `as`, `os`, `e` e `se`: todos são
// palavra inglesa também, e a primeira versão do medidor que os contava acusou
// cerca de trezentos nomes de teste em inglês CORRETÍSSIMO — o
// `TestWithoutASceneTheTurnDoesNotAdvance` reprovava pelo artigo "a". Um guarda
// que grita sobre o que está certo é desligado na segunda semana, e aí ele
// deixa de medir o que estava errado.
//
// Nome PRÓPRIO também fica fora, pela regra do glossário: `tormenta`, `tibar` e
// `piloto` não são tradução pendente, são o nome da coisa.
var portugueseWords = wordSet(`
da das dos na nas uma umas uns que pela pelo pelos pelas com sem para ao aos
liga ligar poe poem abre abrir fecha fechar guarda guardar alterna alternar desliza deslizar
pega pegar escreve escrever conta contar mede medir monta montar marca marcar desenha desenhar
aplica aplicar acha achar cria criar apaga apagar limpa limpar mostra mostrar volta voltar
entra entrar sai sair cabe caber vale valer diz dizer faz fazer vai leva levar traz trazer
solta soltar arrasta arrastar arrasto arrastos anda andar pinta pintar corta cortar junta juntar
peca pecas mesa mesas fila filas largura altura alturas tela telas cena cenas ficha fichas
tabuleiro casa casas quadrado quadrados palco palcos vez vezes toque toques alvo alvos
piscada piscadas pulso pulsos surgir condicao condicoes fracao fracoes atual atuais
movimento movimentos reduzido reduzida parado parada descartavel descartaveis combatente combatentes
mapa mapas notas nota cor cores lugar lugares livro livros folha folhas trilho trilhos
rascunho rascunhos elenco fantasma fantasmas gabarito regua cortina cracha
vista vistas janela janelas dedo dedos passo passos teto tetos piso pisos divisa divisas
coluna colunas linha linhas gaveta gavetas faixa faixas barra barras seta setas
proxima proximo primeiro primeira ultimo ultima cada todos todas nenhum nenhuma
sessao sessoes campanha campanhas jogador jogadores mestre mestres personagem personagens
vivo viva morto morta cheio cheia vazio vazia aberto aberta fechado fechada
permitidos deformidade carisma bloco teste botao classe rotulo contem campo pede
devoto barbaro caminho banco escolhe medicao minimo primeiros posta texto existe arcanista ler
escrito chaves especializacao armadura aceita partida herois caixa conjurador carta contraste
nada limite corpo troca cartaz variante estatico fontes nao linearizado resumo quadro amostra
javali taverna guerreiro ladino clerigo druida bardo paladino cacador inventor nobre lutador
pericia pericias magia magias efeito efeitos tesouro tesouros ataque ataques defesa dano
vida morte nivel niveis raca racas origem origens poder poderes divindade divindades
sentido sentidos idioma idiomas tamanho tamanhos deslocamento pontos ponto valor valores
nome nomes numero numeros ordem ordens lista listas grupo grupos`)

func wordSet(bruto string) map[string]bool {
	m := map[string]bool{}
	for _, p := range strings.Fields(bruto) {
		m[p] = true
	}
	return m
}

// declarations são as formas que DECLARAM um nome. `.templ` recebe as do Go
// também: um `type RailMarker struct` mora dentro de um `.templ` ao lado do
// componente, e a primeira versão deste varredor só procurava `templ Nome(` ali
// — o controle acusou o `RailMarker` como AUSENTE, que é o balde que ninguém lê.
var declarations = map[string][]*regexp.Regexp{
	".go": {
		regexp.MustCompile(`^func (?:\([^)]*\) )?([A-Za-z_][A-Za-z0-9_]*)\(`),
		regexp.MustCompile(`^type ([A-Za-z_][A-Za-z0-9_]*) `),
		regexp.MustCompile(`^(?:var|const) ([A-Za-z_][A-Za-z0-9_]*)[ =]`),
	},
	".templ": {regexp.MustCompile(`^templ ([A-Za-z_][A-Za-z0-9_]*)\(`)},
	".ts": {
		regexp.MustCompile(`^(?:export )?(?:async )?function ([A-Za-z_$][\w$]*)\(`),
		regexp.MustCompile(`^(?:export )?(?:const|let) ([A-Za-z_$][\w$]*)[ :=]`),
		regexp.MustCompile(`^(?:export )?(?:class|interface|type) ([A-Za-z_$][\w$]*)[ <={]`),
	},
}

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// portugueseIn devolve os segmentos portugueses de um nome, ou nil.
func portugueseIn(nome string) []string {
	espacado := camelBoundary.ReplaceAllString(strings.ReplaceAll(nome, "_", " "), "$1 $2")
	var achados []string
	for _, seg := range strings.Fields(espacado) {
		if s := strings.ToLower(seg); portugueseWords[s] {
			achados = append(achados, s)
		}
	}
	return achados
}

// TestNoNewIdentifierIsWrittenInPortuguese varre `.go`, `.templ` e `.ts` dos DOIS
// pacotes e cobra a regra de idioma contra a linha de base.
func TestNoNewIdentifierIsWrittenInPortuguese(t *testing.T) {
	linhaDeBase := map[string]bool{}
	bruto, err := os.ReadFile(portugueseBaseline)
	if err != nil {
		t.Fatalf("ler a linha de base %s: %v", portugueseBaseline, err)
	}
	for _, l := range strings.Split(string(bruto), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			linhaDeBase[l] = true
		}
	}

	achados := map[string]string{}
	medidos, arquivos := 0, 0
	// A RAIZ é o repositório e não o `engine-go`: metade dos identificadores que
	// originaram este guarda mora no `e2e/`, e um varredor que parasse no pacote
	// de cima passaria verde sobre eles.
	raiz := filepath.Join("..", "..")
	err = filepath.WalkDir(raiz, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			switch entrada.Name() {
			case ".git", "node_modules", "test-results", "playwright-report", "dist", "backups", "data":
				return fs.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(caminho)
		regras, tem := declarations[ext]
		if !tem || strings.HasSuffix(caminho, "_templ.go") {
			return nil
		}
		if ext == ".templ" {
			regras = append(append([]*regexp.Regexp{}, regras...), declarations[".go"]...)
		}
		corpo, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		arquivos++
		relativo := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(caminho, raiz)), "/")
		for _, linha := range strings.Split(string(corpo), "\n") {
			for _, rx := range regras {
				m := rx.FindStringSubmatch(linha)
				if m == nil {
					continue
				}
				medidos++
				if segs := portugueseIn(strings.TrimPrefix(m[1], "Test")); segs != nil {
					achados[m[1]] = relativo + " (" + strings.Join(segs, ", ") + ")"
				}
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer: %v", err)
	}

	// O DENOMINADOR, porque "nada reprovou" e "não mediu nada" são a mesma cor no
	// terminal. Os pisos são folgados de propósito: eles denunciam a raiz trocada
	// e o `WalkDir` que parou no primeiro diretório, não uma fatia que apagou dez
	// funções.
	if arquivos < 300 || medidos < 3000 {
		t.Fatalf("a varredura leu %d arquivos e %d declarações — a raiz é o primeiro suspeito", arquivos, medidos)
	}

	var novos, sumidos []string
	for nome, onde := range achados {
		if !linhaDeBase[nome] {
			novos = append(novos, nome+" — "+onde)
		}
	}
	for nome := range linhaDeBase {
		if _, ainda := achados[nome]; !ainda {
			sumidos = append(sumidos, nome)
		}
	}
	sort.Strings(novos)
	sort.Strings(sumidos)
	if len(novos) > 0 {
		t.Errorf("identificador em PORTUGUÊS, e a regra pede inglês (CLAUDE.md, \"Idioma\") — %d:\n  %s\n"+
			"Renomeie. A linha de base em %s registra a dívida ANTIGA e não aceita nome novo.",
			len(novos), strings.Join(novos, "\n  "), portugueseBaseline)
	}
	if len(sumidos) > 0 {
		t.Errorf("a linha de base cita %d nome(s) que não existem mais:\n  %s\n"+
			"Tire-os de %s: uma catraca que não encolhe deixa de ser catraca, e um arquivo "+
			"que descreve o que já não existe é defeito entregue igual a qualquer outro.",
			len(sumidos), strings.Join(sumidos, "\n  "), portugueseBaseline)
	}
	t.Logf("dívida de idioma: %d identificadores em português, de %d medidos em %d arquivos",
		len(achados), medidos, arquivos)
}
