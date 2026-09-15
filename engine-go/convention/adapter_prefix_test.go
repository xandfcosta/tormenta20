package convention

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// O NOME DO ARQUIVO DIZ O ADAPTADOR QUE O POSSUI (ALE-330).
//
// O `serve/api` nomeava por ASSUNTO e era possuído por TIPO, e as duas coisas
// não se encontravam: o maior grupo, `character_*` com treze arquivos, estava
// repartido entre a MESA e a FICHA, e não havia como saber pelo nome a qual dos
// dois um arquivo pertencia. Os 115 testes do diretório já seguiam a convenção
// — `table_*`, `sheetui_*`, `campaigns_*` —; a produção é que não.
//
// # Por que um guarda e não um parágrafo
//
// Porque uma convenção escrita e não varrida vale exatamente para os arquivos
// que alguém apontou. A varredura aconteceu uma vez, à mão, em dezenove
// arquivos; o que impede o vigésimo de nascer torto é este teste, que FALHA COM
// O NOME do arquivo e do dono dele — a diferença entre "conserte isto" e
// "procure".
//
// # O que ele mede, e o que não
//
// Ele lê o RECEPTOR de cada método declarado. O `*Server` não conta: ele é a
// fiação, está em metade do diretório e não distingue nada. Os HOSTS de cena
// (`tableHost`, `hubHost`, `doorHost`) também não — eles já têm a convenção
// deles, `*_deps.go`, e um arquivo pode legitimamente ter os dois.
//
// Um arquivo com métodos de DOIS adaptadores reprova sem exceção possível, e
// esse é o defeito que a issue veio consertar: `character.go` tinha três donos,
// e o arquivo dos membros tinha dois — ele virou `table_combatants.go` e
// `campaign_members.go` —, e o nome não dizia nenhum.
var adapterPrefixes = map[string]string{
	"tableRules":    "table",
	"sheetRules":    "sheet",
	"campaignRules": "campaign",
	"accountRules":  "account",
}

// O `server.go` é a ÚNICA exceção, e ela é argumentada e não tolerada.
//
// Ele declara o `sheetRules.characterChanged`, que é metade do mecanismo de
// serialização de escrita por personagem — a trava, o middleware e o id são do
// `Server` e moram nas linhas seguintes. Partir por RECEPTOR partiria o
// mecanismo em dois arquivos que ninguém lê junto, e o mecanismo é a coisa que
// tem uma razão para mudar. Renomear o arquivo para `sheet_*` seria pior ainda:
// ele é o `Server`.
//
// A lista é curta de propósito. Ela crescer é o sinal de que a convenção parou
// de servir — não de que faltam exceções.
var adapterPrefixExceptions = map[string]string{
	"server.go": "o `characterChanged` é metade de um mecanismo cuja outra metade é do `Server`",
}

var methodReceiver = regexp.MustCompile(`^func \(\w+ \*?(\w+)\)`)

func TestEveryAdapterFileCarriesItsPrefix(t *testing.T) {
	dir := filepath.Join("..", "serve", "api")
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ler %s: %v", dir, err)
	}

	medidos := 0
	for _, e := range entradas {
		nome := e.Name()
		if e.IsDir() || !strings.HasSuffix(nome, ".go") || strings.HasSuffix(nome, "_test.go") {
			continue
		}
		conteudo, err := os.ReadFile(filepath.Join(dir, nome))
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		donos := map[string]bool{}
		for _, linha := range strings.Split(string(conteudo), "\n") {
			m := methodReceiver.FindStringSubmatch(linha)
			if m == nil {
				continue
			}
			if _, ehAdaptador := adapterPrefixes[m[1]]; ehAdaptador {
				donos[m[1]] = true
			}
		}
		if len(donos) == 0 {
			continue
		}
		medidos++

		if len(donos) > 1 {
			nomes := make([]string, 0, len(donos))
			for d := range donos {
				nomes = append(nomes, d)
			}
			sort.Strings(nomes)
			t.Errorf("%s declara métodos de %s. Um arquivo, um dono: parta-o, "+
				"uma metade por adaptador. Foi o defeito que a ALE-330 consertou em "+
				"`character.go`, que tinha três donos, e no arquivo dos membros, que "+
				"tinha dois.",
				nome, strings.Join(nomes, " e "))
			continue
		}

		var dono string
		for d := range donos {
			dono = d
		}
		if motivo, ok := adapterPrefixExceptions[nome]; ok {
			t.Logf("%s é exceção declarada (%s): %s", nome, dono, motivo)
			continue
		}
		if prefixo := adapterPrefixes[dono]; !strings.HasPrefix(nome, prefixo) {
			t.Errorf("%s declara métodos de %s e devia começar com %q. "+
				"O nome do arquivo diz o ADAPTADOR que o possui; o assunto vem depois "+
				"do prefixo (CLAUDE.md do engine-go, \"Onde procurar\").",
				nome, dono, prefixo+"_")
		}
	}

	// CONTROLE: uma lista de reprovados vazia e um regex que parou de casar se
	// parecem no terminal. O receptor é a única coisa que este guarda lê, e se o
	// padrão dele quebrar não sobra nada para reprovar.
	if medidos < 20 {
		t.Fatalf("só %d arquivos com dono medidos — o padrão do receptor parou de "+
			"casar e o verde não significa nada", medidos)
	}
	t.Logf("%d arquivos de produção com adaptador dono", medidos)
}
