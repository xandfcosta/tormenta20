package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// A CHAVE DO PAYLOAD TEM O NOME DO SINAL QUE ELA LÊ (ALE-306).
//
// O `payload` do `@post` SUBSTITUI os sinais em vez de acrescentá-los. Um gesto
// que precisa de um valor calculado no clique E do formulário da página tem de
// LISTAR o formulário à mão:
//
//	{payload: {from: {X: cx, Y: cy},
//	           new_token_name: $new_token_name, …}}
//
// Isso é uma grafia A MAIS de cada nome de sinal, num lugar que o `grep` de
// `$nome` não distingue de um leitor qualquer. E o modo de falhar é o da
// família: `new_token_name: $new_token_look` manda o valor errado no campo
// certo, o servidor aceita, e a peça nasce com a aparência no nome — sem erro em
// lugar nenhum.
//
// # O que ele cobra, e por que a primeira versão era larga demais
//
// Ele cobra a igualdade SÓ quando a chave é ela mesma um nome de sinal. Escrito
// como "toda chave tem de casar com o sinal que lê", o guarda reprovou três
// pares CERTOS na primeira execução: `kind: $tool`, `shape: $template` — ali o
// campo do corpo e o sinal são conceitos diferentes de propósito (a ferramenta
// que o dedo segura contra a espécie que o corpo carrega).
//
// A regra estreita pega o que interessa e só isso: se a chave `new_token_name`
// existe COMO SINAL, então ela tem de carregar `$new_token_name` e não
// `$new_token_look`. É o copiar-colar dentro de uma família de sinais irmãos —
// que é exatamente onde o olho escorrega.
//
// Ele NÃO cobra que o payload esteja completo: ligar o sítio do payload ao
// handler que o lê exige seguir a rota, e uma varredura que chutasse isso
// acusaria o certo.
var (
	// A ÂNCORA aceita espaço antes dos dois-pontos. `payload :` é JavaScript
	// válido, e trocar `payload:` por `payload :` em três sítios derrubava o
	// guarda de nove pares para UM — e ele passava, porque o piso era `pares < 1`.
	payloadAnchor = regexp.MustCompile(`payload\s*:`)
	payloadPair   = regexp.MustCompile(`([a-zA-Z_][\w]*)\s*:\s*\$([a-zA-Z_][\w]*)\b`)
	payloadKey    = regexp.MustCompile(`([a-zA-Z_][\w]*)\s*:`)
	jsSpread      = regexp.MustCompile(`\.\.\.([a-zA-Z_][\w]*)`)
	jsConstObject = regexp.MustCompile(`const\s+([a-zA-Z_][\w]*)\s*=\s*\{`)
	goFieldName   = regexp.MustCompile(`^\s*([A-Z][\w]*)\s+[\[\]\*a-zA-Z]`)
)

// balancedObject devolve o objeto `{…}` que começa no primeiro `{` a partir de
// `i`, casando as chaves.
//
// Era uma janela de 400 CARACTERES, e ela sangrava de um `payload:` para o
// seguinte: os nove pares que o guarda reportava eram OITO distintos, com
// `marked_tokens` contado duas vezes. Um denominador que conta o mesmo par duas
// vezes é um denominador que não denuncia nada.
//
// Devolve `"", false` quando o valor do payload não é um objeto literal — é uma
// VARIÁVEL —, e quem chama resolve ou conta como não lido.
func balancedObject(texto string, i int) (string, bool) {
	for i < len(texto) && (texto[i] == ' ' || texto[i] == '\t') {
		i++
	}
	if i >= len(texto) || texto[i] != '{' {
		return "", false
	}
	profundidade := 0
	for j := i; j < len(texto); j++ {
		switch texto[j] {
		case '{':
			profundidade++
		case '}':
			profundidade--
			if profundidade == 0 {
				return texto[i : j+1], true
			}
		}
	}
	return "", false
}

// serverReaders são os nomes que o servidor sabe ler de um corpo JSON.
//
// DOIS canais, e o segundo não é zelo: o `encoding/json` casa pelo NOME DO CAMPO
// quando não há tag, e casa SEM DIFERENCIAR CAIXA quando não há correspondência
// exata. É por isso que `{X: cx, Y: cy}` no cliente pousa num `engine.Square`
// cujas tags são `json:"x"` e `json:"y"` — em minúsculas.
//
// Isso FUNCIONA POR ACIDENTE, e o acidente é o mesmo que o `CLAUDE.md` registra
// na armadilha do camelCase: duas grafias para um conceito, unidas por uma
// tolerância da biblioteca. Ele está anotado aqui em vez de consertado porque
// consertá-lo mexe no fio de oito sítios, e isso é issue própria.
func serverReaders(t *testing.T, root string, arquivos []string) map[string]bool {
	t.Helper()
	tag := regexp.MustCompile(`json:"([a-zA-Z_][\w]*)"`)
	lidos := map[string]bool{}
	for _, relative := range arquivos {
		if relative == "" || !strings.HasSuffix(relative, ".go") || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		for _, linha := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(linha), "//") {
				continue
			}
			for _, m := range tag.FindAllStringSubmatch(linha, -1) {
				lidos[strings.ToLower(m[1])] = true
			}
			if m := goFieldName.FindStringSubmatch(linha); m != nil {
				lidos[strings.ToLower(m[1])] = true
			}
		}
	}
	return lidos
}

func TestEveryPayloadKeyMatchesTheSignalItReads(t *testing.T) {
	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	arquivos := strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00")

	// OS SINAIS DA ÁRVORE, colhidos como o `TestNoNewSignalBreaksTheNamingStandard`
	// os colhe: é essa lista que separa "chave que também é sinal" de "campo de
	// corpo com nome próprio".
	sinais := map[string]bool{}
	for _, relative := range arquivos {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		for _, m := range signalInExpression.FindAllStringSubmatch(string(body), -1) {
			sinais[m[1]] = true
		}
	}
	lidosPeloServidor := serverReaders(t, root, arquivos)

	sitios, pares, filesRead := 0, 0, 0
	var tortos, semLeitor []string
	for _, relative := range arquivos {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") ||
			strings.HasSuffix(relative, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		// COMENTÁRIO fora, e a primeira versão sem isso acusou o próprio
		// cabeçalho deste arquivo: o exemplo errado está escrito ali de
		// propósito, para quem lê entender o que se evita.
		texto := strings.Join(semComentario(strings.Split(string(body), "\n")), "\n")
		// OS OBJETOS declarados por `const x = {…}` na mesma expressão, porque
		// dois sítios mandam `{payload: traco}` com o objeto montado na linha de
		// cima. Sem resolvê-los, esses sítios contribuem ZERO pares em silêncio.
		objetos := map[string]string{}
		for _, corte := range jsConstObject.FindAllStringSubmatchIndex(texto, -1) {
			if corpo, ok := balancedObject(texto, corte[1]-1); ok {
				objetos[texto[corte[2]:corte[3]]] = corpo
			}
		}

		for _, corte := range payloadAnchor.FindAllStringIndex(texto, -1) {
			sitios++
			janela, ok := balancedObject(texto, corte[1])
			if !ok {
				// `{payload: traco}`: o valor é a variável inteira, e as chaves
				// dela já foram lidas onde ela foi declarada.
				continue
			}
			// O SPREAD traz as chaves do objeto de origem junto.
			for _, m := range jsSpread.FindAllStringSubmatch(janela, -1) {
				janela += objetos[m[1]]
			}
			linha := strings.Count(texto[:corte[0]], "\n") + 1
			onde := relative + ":" + strconv.Itoa(linha)

			for _, m := range payloadPair.FindAllStringSubmatch(janela, -1) {
				if !sinais[m[1]] {
					// A chave é campo de corpo com nome próprio (`kind`,
					// `shape`), e não há o que casar.
					continue
				}
				pares++
				if m[1] != m[2] {
					tortos = append(tortos, onde+" — a chave `"+m[1]+"` carrega o sinal `$"+m[2]+"`")
				}
			}
			// O TERCEIRO CANAL: a chave que o cliente escreve tem de ser lida
			// pelo servidor. Nada prendia isso — o guarda só olhava o lado do
			// cliente, e uma chave renomeada de um lado só pousa em `undefined`
			// sem erro em lugar nenhum.
			for _, m := range payloadKey.FindAllStringSubmatch(janela, -1) {
				chave := m[1]
				if chave == "payload" || lidosPeloServidor[strings.ToLower(chave)] {
					continue
				}
				semLeitor = append(semLeitor, onde+" — a chave `"+chave+"`")
			}
		}
	}

	// O DENOMINADOR é sobre SÍTIOS, e não sobre pares. O piso era `pares < 1`
	// sobre nove, e trocar `payload:` por `payload :` em três lugares derrubava a
	// conta para UM com o guarda passando: ele só afirmava "o regex ainda casa em
	// algum lugar do repositório". Hoje são quinze sítios.
	if filesRead < 300 || sitios < 12 {
		t.Fatalf("a varredura leu %d arquivos e achou %d sítios de `payload:` — a raiz é o primeiro suspeito",
			filesRead, sitios)
	}

	sort.Strings(tortos)
	if len(tortos) > 0 {
		t.Errorf("chave de payload que não tem o nome do sinal que lê — %d de %d pares:\n  %s\n"+
			"O servidor aceita o valor errado no campo certo e nada estoura. Se a chave e o sinal "+
			"são MESMO diferentes, o lugar de dizer isso é um campo de corpo com nome próprio, "+
			"não um par que parece igual e não é.",
			len(tortos), pares, strings.Join(tortos, "\n  "))
	}
	sort.Strings(semLeitor)
	if len(semLeitor) > 0 {
		t.Errorf("chave de payload que NENHUM campo do servidor lê — %d:\n  %s\n"+
			"Leitor é uma tag `json:\"chave\"` ou um campo exportado de mesmo nome (o `encoding/json` "+
			"casa por nome quando não há tag). Sem nenhum dos dois, o campo chega ao servidor e cai "+
			"no chão: o gesto responde 200 com o valor-zero, sem erro em lugar nenhum.",
			len(semLeitor), strings.Join(semLeitor, "\n  "))
	}
	if !t.Failed() {
		t.Logf("payload: %d sítios, %d pares `chave: $sinal`, todos casados e lidos, de %d arquivos",
			sitios, pares, filesRead)
	}
}
