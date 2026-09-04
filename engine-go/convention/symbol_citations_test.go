package convention

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// NENHUMA CITAÇÃO NOMEIA UM SÍMBOLO QUE NÃO EXISTE (ALE-286).
//
// Terceiro da família. O `TestNoCitationNamesAMissingTest` prende nome de TESTE
// (mediu 136) e o `TestNoCitationNamesAMissingFile` prende CAMINHO DE ARQUIVO
// (mediu 41). Falta a forma mais numerosa: o comentário que nomeia uma FUNÇÃO,
// um TIPO ou uma VARIÁVEL que sumiu. Nome de função não é nome de teste nem
// caminho de arquivo, então os dois passam por cima — não havia rede nenhuma, e
// a varredura desta issue mediu 466 casos.
//
// # A origem não é descuido: é o renomeador funcionando como projetado
//
// Ele PROTEGE comentário — tem de proteger, senão a troca entra na prosa e na
// tela. Isso já aconteceu três vezes nesta casa: "Invite inválido ou expirado",
// "Difficulty" onde estava "Dificuldade", e "Encerrar scene" onde estava
// "Encerrar cena". O preço é a docstring falando do nome velho.
//
// # O DESAFIO deste guarda é o RUÍDO, e ele se resolve por REGRA
//
// Uma sonda ingênua acusa 1.371 e a maioria é legítima. As naturezas do ruído
// são cinco, e as CINCO são colhidas mecanicamente em vez de enumeradas —
// enumerar é remendo, e a lista envelheceria a cada arquivo novo:
//
//  1. **chave de fio**: a tag `json:` e a coluna do `.sql`;
//  2. **chave de catálogo**: o que os `catalog/data/*.json` declaram;
//  3. **literal de string**: nome de sinal do Datastar, id de poder, chave de
//     modificador. Se o código escreve `"X"`, então `X` existe — é contrato;
//  4. **API chamada**: qualquer `.X(` no código do repositório. Isso cobre a
//     biblioteca padrão, o `chi`, o `templ` e o navegador de uma vez;
//  5. **símbolo do JS/TS** das ilhas do piloto.
//
// Mais duas regras que não são coleta e sim JULGAMENTO, e as duas foram medidas
// antes de virar código:
//
//   - **A CAIXA**: `stateForRole` casando com `StateForRole` é PROCEDÊNCIA, e
//     não citação morta. Esta casa valoriza dizer de onde a regra veio.
//   - **A PROCEDÊNCIA DECLARADA no bloco**: quando o comentário nomeia o arquivo
//     `.ts` de origem, ou diz "ports"/"mirrors", ou escreve com todas as letras
//     que a coisa "não veio junto" / "não existe mais" / "morreu com a SPA", ele
//     está fazendo o que a lápide faz. Cobrar um símbolo vivo aí seria cobrar a
//     documentação de ser falsa sobre si mesma.
//
// **É por isso que a lista de lápides é curta.** Ela sobrou para o que nenhuma
// regra alcança, e cada entrada diz por quê.
var simbolosAusentesDePROPOSITO = map[string]bool{
	// ── API DE FORA DO REPOSITÓRIO ───────────────────────────────────────────
	//
	// A coleta é do repositório: ela não conhece a biblioteca padrão do Go nem o
	// navegador. Um comentário que explica POR QUE uma opção não foi usada tem de
	// nomeá-la, e o caso mais claro é o `WriteTimeout` — o servidor não o define
	// DE PROPÓSITO, porque ele mataria o fluxo SSE, e há teste afirmando a
	// ausência. Um guarda que proibisse nomeá-lo apagaria a razão.
	"WriteTimeout":        true,
	"ReadHeaderTimeout":   true,
	"ErrNotSupported":     true,
	"DisableCompression":  true,
	"SameSite":            true,
	"rc.SetWriteDeadline": true,
	"EventSource":         true,
	"NotFoundError":       true,
	"InvalidStateError":   true,
	"localeCompare":       true,
	"scrollTo":            true,
	"translateX":          true,
	"ipNet":               true,

	// ── PROCEDÊNCIA que a regra do bloco não alcança ─────────────────────────
	//
	// A regra da procedência lê o bloco procurando o `.ts` de origem ou a frase
	// que declara a morte. Estas ficam de fora porque a frase que as explica é
	// mais antiga que a convenção — são comentários em inglês herdados do porte,
	// no formato "X ports Y" com o Y sem extensão. Reescrevê-los para caber no
	// guarda seria mexer na procedência para agradar o instrumento.
	"rankItem":                true,
	"sinopseDeWynlla":         true,
	"parseImprovementIds":     true,
	"resolveAccess":           true,
	"addItem":                 true,
	"ensureCatalogs":          true,
	"enginePools":             true,
	"frontVitalResolver":      true,
	"syncVitalsForProjection": true,
	"healVitalsFromEngine":    true,
	"applyCharacterVitals":    true,
	"buildVitalContext":       true,
	"collectVitalGrants":      true,
	"assertCharacterRules":    true,
	"assertEquipAxisAllowed":  true,
	"assertSpellExists":       true,
	"assertVitalsEditable":    true,
	"assertGm":                true,
	"classProficiencies":      true,
	"describeModifierTarget":  true,
	"familyFor":               true,
	"validateCreature":        true,
	"CreatureBlock":           true,
	"CreatureAttack":          true,
	"CreatureSkill":           true,
	"characterOwnedBy":        true,
	"beforeLoad":              true,
	"refetchOnWindowFocus":    true,
	"samiraFeiticeira":        true,

	// ── O QUE MORREU COM O SOCKET.IO E COM A SPA ─────────────────────────────
	//
	// Mesma natureza da lista do guarda irmão: o corte foi pelo receptor, e dizer
	// o que SOBROU exige nomear o que foi cortado.
	"guardSocketOrigin":      true,
	"emitSessionState":       true,
	"mutateAndBroadcast":     true,
	"mutateBoard":            true,
	"msgCtx":                 true,
	"onSceneEnd":             true,
	"onGetState":             true,
	"notifyCharacterChanged": true,
	"CharacterWatch":         true,
	"persistAndWarn":         true,
	"writeHp":                true,
	"trimmedNull":            true,
	"nextStepOrigin":         true,
	"leitorView.EmDialogo":   true,
	"handleImproviso":        true,

	// ── O NOME HIPOTÉTICO, que é o caso mais sutil ───────────────────────────
	//
	// `detalheAberto` nunca existiu: ele é o exemplo do que ACONTECERIA se
	// alguém escrevesse um sinal em camelCase — o HTML minuscula a chave do
	// atributo e o Datastar liga um sinal NOVO. A prosa precisa escrever o nome
	// errado para mostrar o defeito, e um guarda que a proibisse tiraria do
	// repositório justamente a explicação que impede o defeito.
	//
	// O mesmo vale para o `ForTable`: ele é o SUFIXO de três métodos da porta da
	// Mesa (`StartSessionForTable` e irmãos), citado como sufixo e não como
	// símbolo.
	"detalheAberto": true,
	"ForTable":      true,

	// E o `ifNoneMatch` é a terceira forma do mesmo caso: ele é a VARIÁVEL de um
	// trecho de código que o comentário cita para mostrar o defeito que já foi
	// consertado (`strings.Contains(ifNoneMatch, "")` respondia verdadeiro para
	// dígito vazio). O código sumiu; a explicação de por que a ordem das linhas é
	// aquela precisa dele para fazer sentido.
	"ifNoneMatch": true,

	// ── A PORTA QUE A ALE-205 APOSENTOU (ALE-289) ────────────────────────────
	//
	// O `ShowPlace` punha uma cena guardada na mesa ARQUIVANDO antes a que estava
	// lá — a saída que a ALE-191 inventou para o mestre pular da taverna para a
	// cripta sem perder a taverna. Com abas, nada é substituído e o problema
	// deixou de existir; ele ficou três fatias no ar com zero chamadores de
	// produção e quatro testes em cima, um deles afirmando em VERDE o
	// comportamento que a ALE-205 tinha removido.
	//
	// Ele é citado em TRÊS blocos que não são lápide e estão certos: a docstring
	// do `OpenPlace`, que se define pelo contraste com ele; o `collection_test`,
	// que nomeia o papercut que a fatia 3 consertou; e o `reopenPlace` da cena.
	// Os três explicam POR QUE a coisa de hoje é como é, e apagar o nome deles
	// apagaria a razão junto — que é exatamente o que esta lista existe para
	// impedir.
	"ShowPlace": true,
}

// oSimboloCitado é o que parece identificador dentro de um comentário: entre
// crases, ou camelCase de verdade.
//
// O filtro de camelCase é o que a fatia da Mesa deixou escrito e não se
// negocia: `filtra`, `aperta`, `empilha` e `sorteio` viraram nomes ingleses no
// código e continuam sendo PALAVRAS portuguesas no comentário ao lado. Uma
// palavra só, sem maiúscula no meio, é prosa até prova em contrário.
var oSimboloCitado = regexp.MustCompile("`([A-Za-z][\\w.]*)`|\\b([a-z]+[A-Z]\\w*)\\b")

// DUAS minúsculas antes da maiúscula, e não uma: `vCPUs` e `mAh` são PROSA
// (a máquina da CI tem "2 vCPUs"), e uma letra só antes da maiúscula é o que os
// separa de um identificador de verdade.
var ehCamelDeVerdade = regexp.MustCompile(`^[a-z]{2,}[A-Z]\w*$|^[A-Z][a-z]+[A-Z]\w*$`)

// aProcedenciaDeclarada é o bloco dizendo, ele mesmo, que o nome não vive aqui.
var aProcedenciaDeclarada = regexp.MustCompile(
	`\.tsx?\b|\bports\b|\bmirrors\b|\bSPA\b|TypeScript|` +
		`não veio junto|deixou de existir|não existe mais|morreu com|some junto|` +
		`Aqui morava|apagad|morta`)

func TestNoCitationNamesAMissingSymbol(t *testing.T) {
	existe, semCaixa := oQueORepositorioDeclara(t)
	arquivos := arquivosParaCitacao(t)

	medidas := 0
	for _, caminho := range arquivos {
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		linhas := strings.Split(string(conteudo), "\n")
		for numero, linha := range linhas {
			k := strings.Index(linha, "//")
			if k < 0 {
				continue
			}
			for _, achado := range oSimboloCitado.FindAllStringSubmatch(linha[k:], -1) {
				citado := achado[1]
				if citado == "" {
					citado = achado[2]
				}
				nome := citado[strings.LastIndex(citado, ".")+1:]
				raiz := citado
				if i := strings.Index(citado, "."); i >= 0 {
					raiz = citado[:i]
				}
				if !ehCamelDeVerdade.MatchString(nome) {
					continue
				}
				medidas++
				if existe[nome] || existe[raiz] || semCaixa[strings.ToLower(nome)] {
					continue
				}
				if simbolosAusentesDePROPOSITO[nome] {
					continue
				}
				if aProcedenciaDeclarada.MatchString(oBlocoDoComentario(linhas, numero)) {
					continue
				}
				t.Errorf("%s:%d cita `%s`, que não existe na árvore.\n"+
					"Se o símbolo foi RENOMEADO, a citação acompanha — o nome novo costuma ser\n"+
					"a declaração logo abaixo do bloco.\n"+
					"Se ele MORREU de propósito, o bloco pode dizer isso em prosa (\"não veio\n"+
					"junto\", \"não existe mais\", o `.ts` de onde veio) ou o nome entra em\n"+
					"`simbolosAusentesDePROPOSITO` — dizer por que uma coisa saiu é bom, e o\n"+
					"que falta é o ato ser explícito.",
					caminho, numero+1, citado)
			}
		}
	}

	// O DENOMINADOR, pela mesma razão dos irmãos: uma lista de reprovados vazia e
	// um regex que parou de casar são a mesma linha verde. O piso fica longe do
	// número real de propósito — o que ele pega é a varredura QUEBRAR, não a
	// prosa encolher.
	if medidas < 1500 {
		t.Fatalf("só %d citações de símbolo lidas — o guarda ficou cego", medidas)
	}
}

// oBlocoDoComentario devolve o bloco de comentário CONTÍGUO em volta da linha.
//
// A procedência quase nunca está na mesma linha do nome: ela está na frase, que
// ocupa três ou quatro linhas. Ler linha a linha faria a regra da procedência
// não pegar quase nada.
func oBlocoDoComentario(linhas []string, i int) string {
	ini, fim := i, i
	for ini > 0 && strings.Contains(linhas[ini-1], "//") {
		ini--
	}
	for fim+1 < len(linhas) && strings.Contains(linhas[fim+1], "//") {
		fim++
	}
	return strings.Join(linhas[ini:fim+1], " ")
}

// oQueORepositorioDeclara colhe as cinco naturezas mecânicas.
//
// Ele devolve DOIS conjuntos: o dos nomes como estão, e o dos nomes em
// minúscula — o segundo é o que faz a regra da CAIXA funcionar.
func oQueORepositorioDeclara(t *testing.T) (map[string]bool, map[string]bool) {
	t.Helper()
	existe := map[string]bool{}
	// A última alternância é a CONSTANTE DE `iota` SEM TIPO, e ela entrou na
	// ALE-287 depois de o guarda acusar FALSO.
	//
	// Num bloco `const (…)` só a PRIMEIRA linha carrega o tipo:
	//
	//	const (
	//	    JoinOK JoinRefusal = iota   // esta o `^\t(\w+)\s+[\w…]` pega
	//	    JoinNoSuchCampaign          // esta não tinha quem a pegasse
	//	    JoinNeedsInvite
	//	)
	//
	// São 17 nomes na árvore, e o silêncio deles é do tipo caro: quem os citasse
	// levava um "não existe na árvore" apontando para um símbolo que EXISTE, e o
	// conserto óbvio — pôr o nome em `simbolosAusentesDePROPOSITO` — teria feito
	// o guarda parar de conferir um nome vivo. **Guarda que acusa falso ensina a
	// desligá-lo**, e essa é a única forma de cegueira que uma lista de exceções
	// não consegue distinguir de um acerto.
	//
	// Um identificador sozinho numa linha indentada não é ambíguo em Go: dentro
	// de função ele não é instrução válida (rótulo tem dois-pontos), e em bloco
	// `var` ou `type` ele não compila sem tipo. Só `const` produz esta forma.
	decl := regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?(\w+)|^(?:type|var|const)\s+(\w+)|^templ\s+(\w+)|^\t(\w+)\s+[\w\[\]\*\.]|^\t(\w+)\s*=|^\t(\w+)\s*$`)
	local := regexp.MustCompile(`\b(\w+)\s*:=`)
	// `var a, b bool` dentro de uma função: o `:=` não a pega, e ela é
	// declaração igual.
	varLocal := regexp.MustCompile(`\bvar\s+([\w,\s]+?)\s+[\w\[\]\*]`)
	tagJSON := regexp.MustCompile(`json:"([^",]+)`)
	chamada := regexp.MustCompile(`\.(\w+)\(`)
	literal := regexp.MustCompile(`"([^"\n]{2,60})"`)
	palavra := regexp.MustCompile(`[A-Za-z_]\w*`)
	param := regexp.MustCompile(`(?m)^func[^{]*\(([^)]*)\)`)
	// CHAVE DE LITERAL COMPOSTO: `ReadHeaderTimeout: 5 * time.Second` nomeia um
	// campo de um tipo de OUTRO pacote, que nenhuma das outras naturezas colhe.
	campoLiteral := regexp.MustCompile(`(?m)^\s*(\w+):\s`)

	visitados := 0
	err := filepath.WalkDir("../..", func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entrada.IsDir() {
			if entrada.Name() == "node_modules" || entrada.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(caminho)
		bruto, lerErr := os.ReadFile(caminho)
		if lerErr != nil {
			return nil
		}
		s := string(bruto)
		switch ext {
		case ".go", ".templ":
			visitados++
			for _, m := range decl.FindAllStringSubmatch(s, -1) {
				for _, g := range m[1:] {
					if g != "" {
						existe[g] = true
					}
				}
			}
			for _, m := range local.FindAllStringSubmatch(s, -1) {
				existe[m[1]] = true
			}
			for _, m := range varLocal.FindAllStringSubmatch(s, -1) {
				for _, w := range palavra.FindAllString(m[1], -1) {
					existe[w] = true
				}
			}
			for _, m := range tagJSON.FindAllStringSubmatch(s, -1) {
				existe[m[1]] = true
			}
			for _, m := range chamada.FindAllStringSubmatch(s, -1) {
				existe[m[1]] = true
			}
			for _, m := range literal.FindAllStringSubmatch(s, -1) {
				for _, w := range palavra.FindAllString(m[1], -1) {
					existe[w] = true
				}
			}
			for _, m := range param.FindAllStringSubmatch(s, -1) {
				for _, w := range palavra.FindAllString(m[1], -1) {
					existe[w] = true
				}
			}
			for _, m := range campoLiteral.FindAllStringSubmatch(s, -1) {
				existe[m[1]] = true
			}
		case ".sql", ".ts", ".tsx", ".js":
			for _, w := range palavra.FindAllString(s, -1) {
				existe[w] = true
			}
		case ".json":
			var qualquer any
			if json.Unmarshal(bruto, &qualquer) == nil {
				colheChaves(qualquer, existe)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O SEGUNDO DENOMINADOR: um conjunto vazio faria TODA citação reprovar, e um
	// conjunto que engoliu o `node_modules` faria toda citação passar. Os dois
	// são silêncio, e este piso separa os dois do caso normal.
	if visitados < 300 || len(existe) < 10000 {
		t.Fatalf("a coleta leu %d arquivos e achou %d símbolos — ela está medindo "+
			"a árvore errada", visitados, len(existe))
	}

	semCaixa := make(map[string]bool, len(existe))
	for d := range existe {
		semCaixa[strings.ToLower(d)] = true
	}
	return existe, semCaixa
}

func colheChaves(o any, existe map[string]bool) {
	switch v := o.(type) {
	case map[string]any:
		for k, filho := range v {
			existe[k] = true
			colheChaves(filho, existe)
		}
	case []any:
		for _, filho := range v {
			colheChaves(filho, existe)
		}
	}
}
