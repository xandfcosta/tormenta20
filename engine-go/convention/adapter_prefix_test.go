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
// e o arquivo dos membros tinha dois — ele foi repartido, um por dono, e o nome
// de cada metade passou a dizer qual. As duas acabaram descendo para o `app/` e
// deixando de existir aqui: a da mesa na ALE-344, a de campanha na ALE-348.
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
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ler %s: %v", dir, err)
	}

	measured := 0
	seen := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("ler %s: %v", name, err)
		}
		owners := map[string]bool{}
		for _, row := range strings.Split(string(content), "\n") {
			m := methodReceiver.FindStringSubmatch(row)
			if m == nil {
				continue
			}
			if _, isAdapter := adapterPrefixes[m[1]]; isAdapter {
				owners[m[1]] = true
			}
		}
		if len(owners) == 0 {
			continue
		}
		measured++
		for d := range owners {
			seen[d]++
		}

		if len(owners) > 1 {
			names := make([]string, 0, len(owners))
			for d := range owners {
				names = append(names, d)
			}
			sort.Strings(names)
			t.Errorf("%s declara métodos de %s. Um arquivo, um dono: parta-o, "+
				"uma metade por adaptador. Foi o defeito que a ALE-330 consertou em "+
				"`character.go`, que tinha três donos, e no arquivo dos membros, que "+
				"tinha dois.",
				name, strings.Join(names, " e "))
			continue
		}

		var owner string
		for d := range owners {
			owner = d
		}
		if reason, ok := adapterPrefixExceptions[name]; ok {
			t.Logf("%s é exceção declarada (%s): %s", name, owner, reason)
			continue
		}
		if prefix := adapterPrefixes[owner]; !strings.HasPrefix(name, prefix) {
			t.Errorf("%s declara métodos de %s e devia começar com %q. "+
				"O nome do arquivo diz o ADAPTADOR que o possui; o assunto vem depois "+
				"do prefixo (CLAUDE.md do engine-go, \"Onde procurar\").",
				name, owner, prefix+"_")
		}
	}

	// CONTROLE: uma lista de reprovados vazia e um regex que parou de casar se
	// parecem no terminal. O receptor é a única coisa que este guarda lê, e se o
	// padrão dele quebrar não sobra nada para reprovar.
	//
	// O controle pergunta se CADA adaptador da tabela acima foi encontrado, e
	// não se o total passa de um piso. Aqui morava `medidos < 20`, e ele reprovou
	// na ALE-347 sem que nada estivesse errado: conjurar, beber e subir de nível
	// desceram para o `app/`, três arquivos de `sheetRules` deixaram de existir,
	// e o número caiu para 19. **Piso escrito à mão sobre família que muda
	// envelhece** — e o erro dele é do pior tipo, porque acusa quem fez a coisa
	// certa.
	//
	// Esta forma não apodrece e mede o mesmo: se o padrão do receptor quebrar, os
	// QUATRO somem de uma vez. E se um adaptador de fato acabar — o `sheetRules`
	// está a caminho disso —, apagar a linha dele da tabela é um ato deliberado,
	// que é o que se quer exigir.
	for _, owner := range []string{"tableRules", "sheetRules", "campaignRules", "accountRules"} {
		if seen[owner] == 0 {
			t.Errorf("nenhum arquivo de produção declara método de %s.\n"+
				"Ou o adaptador deixou de existir — e então a linha dele sai do "+
				"`adapterPrefixes`, de propósito e à vista —, ou o padrão do receptor "+
				"parou de casar e o verde não significa nada.", owner)
		}
	}
	t.Logf("%d arquivos de produção com adaptador dono: %v", measured, seen)
}
