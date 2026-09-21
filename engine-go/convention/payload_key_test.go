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
func balancedObject(text string, i int) (string, bool) {
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	if i >= len(text) || text[i] != '{' {
		return "", false
	}
	depth := 0
	for j := i; j < len(text); j++ {
		switch text[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[i : j+1], true
			}
		}
	}
	return "", false
}

// serverReaders são os nomes que o servidor sabe ler de um corpo JSON, na
// grafia EXATA em que ele os declara.
//
// A tag GANHA do nome do campo, e essa é a regra do `encoding/json`: um campo
// tagueado só se lê pelo nome da tag. `X int ` + "`" + `json:"x"` + "`" + ` é lido por `x`, e não
// por `X` — a única razão de `{"X":2}` chegar lá é a TOLERÂNCIA de caixa da
// biblioteca, que só entra quando não há correspondência exata.
//
// # Por que o guarda cobra a grafia exata (ALE-313)
//
// Porque essa tolerância é a mesma que já produziu defeito nesta casa. O
// `CLAUDE.md` registra o camelCase do construtor de encontros: ele *funcionava
// por acidente, porque o encoding/json casa campo sem diferenciar caixa quando
// não há correspondência exata, e a chave ligada vinha por último* — invertida a
// ordem das chaves, o mesmo código lê `""` e a busca deixa de filtrar, sem erro
// em lugar nenhum.
//
// Duas grafias para um conceito, seguradas por uma tolerância de biblioteca, é
// exatamente a raiz que a ALE-301 existiu para apagar. O fio do tabuleiro tinha
// a mesma forma: o cliente escrevia `{X: cx, Y: cy}` e o `engine.Square` declara
// `json:"x"`/`json:"y"`, que é também a grafia GRAVADA em `campaign_places` e
// `open_boards` — foi o dado no banco que decidiu qual lado muda.
func serverReaders(t *testing.T, root string, files []string) map[string]bool {
	t.Helper()
	tag := regexp.MustCompile(`json:"([a-zA-Z_][\w]*)"`)
	read := map[string]bool{}
	for _, relative := range files {
		if relative == "" || !strings.HasSuffix(relative, ".go") || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		for _, row := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(row), "//") {
				continue
			}
			// A TAG GANHA, e o `else` é a regra inteira: um campo tagueado não
			// se lê pelo nome dele. Somar os dois faria o guarda aceitar `X`
			// num campo que só responde por `x`, que é o defeito.
			if findings := tag.FindAllStringSubmatch(row, -1); len(findings) > 0 {
				for _, m := range findings {
					read[m[1]] = true
				}
			} else if m := goFieldName.FindStringSubmatch(row); m != nil {
				read[m[1]] = true
			}
		}
	}
	return read
}

func TestEveryPayloadKeyMatchesTheSignalItReads(t *testing.T) {
	root := filepath.Join("..", "..")
	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	files := strings.Split(strings.TrimRight(string(output), "\x00"), "\x00")

	// OS SINAIS DA ÁRVORE, colhidos como o `TestNoNewSignalBreaksTheNamingStandard`
	// os colhe: é essa lista que separa "chave que também é sinal" de "campo de
	// corpo com nome próprio".
	signals := map[string]bool{}
	for _, relative := range files {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		// COMENTÁRIO fora, aqui também. Sem isto o `data-show="$x"` que o
		// cabeçalho do `overlay_flash_test.go` usa como EXEMPLO entra na lista
		// como se fosse um sinal da árvore — e então `x: $rect_to_x`, que é uma
		// chave de corpo perfeitamente correta, reprova como "chave que é sinal
		// e carrega outro" (ALE-313). O instrumento contaminando a si mesmo com
		// a prosa que o explica.
		noProse := strings.Join(semComentario(strings.Split(string(body), "\n")), "\n")
		for _, m := range signalInExpression.FindAllStringSubmatch(noProse, -1) {
			signals[m[1]] = true
		}
	}
	readByServer := serverReaders(t, root, files)

	sites, pares, filesRead := 0, 0, 0
	var crooked, noReader []string
	for _, relative := range files {
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
		text := strings.Join(semComentario(strings.Split(string(body), "\n")), "\n")
		// OS OBJETOS declarados por `const x = {…}` na mesma expressão, porque
		// dois sítios mandam `{payload: traco}` com o objeto montado na linha de
		// cima. Sem resolvê-los, esses sítios contribuem ZERO pares em silêncio.
		objects := map[string]string{}
		for _, cut := range jsConstObject.FindAllStringSubmatchIndex(text, -1) {
			if body, ok := balancedObject(text, cut[1]-1); ok {
				objects[text[cut[2]:cut[3]]] = body
			}
		}

		for _, cut := range payloadAnchor.FindAllStringIndex(text, -1) {
			sites++
			window, ok := balancedObject(text, cut[1])
			if !ok {
				// `{payload: traco}`: o valor é a variável inteira, e as chaves
				// dela já foram lidas onde ela foi declarada.
				continue
			}
			// O SPREAD traz as chaves do objeto de origem junto.
			for _, m := range jsSpread.FindAllStringSubmatch(window, -1) {
				window += objects[m[1]]
			}
			row := strings.Count(text[:cut[0]], "\n") + 1
			where := relative + ":" + strconv.Itoa(row)

			for _, m := range payloadPair.FindAllStringSubmatch(window, -1) {
				if !signals[m[1]] {
					// A chave é campo de corpo com nome próprio (`kind`,
					// `shape`), e não há o que casar.
					continue
				}
				pares++
				if m[1] != m[2] {
					crooked = append(crooked, where+" — a chave `"+m[1]+"` carrega o sinal `$"+m[2]+"`")
				}
			}
			// O TERCEIRO CANAL: a chave que o cliente escreve tem de ser lida
			// pelo servidor. Nada prendia isso — o guarda só olhava o lado do
			// cliente, e uma chave renomeada de um lado só pousa em `undefined`
			// sem erro em lugar nenhum.
			for _, m := range payloadKey.FindAllStringSubmatch(window, -1) {
				key := m[1]
				if key == "payload" || readByServer[key] {
					continue
				}
				noReader = append(noReader, where+" — a chave `"+key+"`")
			}
		}
	}

	// O DENOMINADOR é sobre SÍTIOS, e não sobre pares. O piso era `pares < 1`
	// sobre nove, e trocar `payload:` por `payload :` em três lugares derrubava a
	// conta para UM com o guarda passando: ele só afirmava "o regex ainda casa em
	// algum lugar do repositório". Hoje são quinze sítios.
	if filesRead < 300 || sites < 12 {
		t.Fatalf("a varredura leu %d arquivos e achou %d sítios de `payload:` — a raiz é o primeiro suspeito",
			filesRead, sites)
	}

	sort.Strings(crooked)
	if len(crooked) > 0 {
		t.Errorf("chave de payload que não tem o nome do sinal que lê — %d de %d pares:\n  %s\n"+
			"O servidor aceita o valor errado no campo certo e nada estoura. Se a chave e o sinal "+
			"são MESMO diferentes, o lugar de dizer isso é um campo de corpo com nome próprio, "+
			"não um par que parece igual e não é.",
			len(crooked), pares, strings.Join(crooked, "\n  "))
	}
	sort.Strings(noReader)
	if len(noReader) > 0 {
		t.Errorf("chave de payload que nenhum campo do servidor lê NESTA GRAFIA — %d:\n  %s\n"+
			"Leitor é uma tag `json:\"chave\"`, ou o nome do campo exportado quando ele não tem tag. "+
			"A CAIXA conta: o `encoding/json` casa sem diferenciar caixa só quando não há "+
			"correspondência exata, e depender disso é duas grafias para um conceito — o mesmo "+
			"acidente que segurava o camelCase do construtor de encontros até a ALE-301. Sem "+
			"leitor nenhum, o campo chega ao servidor e cai no chão: o gesto responde 200 com o "+
			"valor-zero, sem erro em lugar nenhum.",
			len(noReader), strings.Join(noReader, "\n  "))
	}
	if !t.Failed() {
		t.Logf("payload: %d sítios, %d pares `chave: $sinal`, todos casados e lidos, de %d arquivos",
			sites, pares, filesRead)
	}
}
