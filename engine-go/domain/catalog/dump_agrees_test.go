package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// O catálogo existe em DUAS cópias e ninguém comparava as duas.
//
// O motor resolve regras a partir de `parity/_catalogs.json` (o default de
// `CATALOG_PATH`, lido do disco); o navegador e as validações de schema leem os
// arquivos embutidos em `catalog/data`. Hoje elas batem byte a byte, e é por
// isso que nada nunca doeu — mas a invariante anti-auto-acúmulo
// (`modifier_stacking_test.go`) varre a cópia EMBUTIDA enquanto o motor calcula
// com a cópia em DISCO: um modificador acrescentado só ao dump escaparia da
// invariante inteira, verde.
//
// Este teste é a costura que faltava. Ele não valida conteúdo — quem faz isso é
// `rules_tables_test.go`; ele garante que as duas cópias são a MESMA coisa, que
// é a premissa daquele teste valer para o que a produção executa.
func TestDumpAgreesWithEmbeddedCatalog(t *testing.T) {
	dumpPath := filepath.Join("..", "..", "parity", "_catalogs.json")
	raw, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("ler %s: %v", dumpPath, err)
	}
	var dump map[string]json.RawMessage
	if err := json.Unmarshal(raw, &dump); err != nil {
		t.Fatalf("dump não é JSON: %v", err)
	}

	// Chave do dump → recurso embutido. Só o que é a MESMA lista dos dois lados:
	// `races` (derivada) e `tormentaPowerIds` (só os ids) são projeções do dump e
	// não têm gêmeo direto.
	pairs := map[string]string{
		"items":         "items",
		"origins":       "origins",
		"classPowers":   "class-powers",
		"generalPowers": "general-powers",
		"grantedPowers": "granted-powers",
		"racas":         "races",
	}

	for key, resource := range pairs {
		t.Run(key, func(t *testing.T) {
			embedded, err := files.ReadFile("data/" + resource + ".json")
			if err != nil {
				t.Fatalf("recurso embutido %q: %v", resource, err)
			}
			fromDump, ok := dump[key]
			if !ok {
				t.Fatalf("o dump não tem a chave %q", key)
			}
			var a, b any
			if err := json.Unmarshal(fromDump, &a); err != nil {
				t.Fatalf("dump[%s]: %v", key, err)
			}
			if err := json.Unmarshal(embedded, &b); err != nil {
				t.Fatalf("data/%s.json: %v", resource, err)
			}
			if !reflect.DeepEqual(a, b) {
				// A mensagem NOMEIA a entrada que diverge, e diz o que de fato
				// conserta. Ela mandava "regenere com `go run ./cmd/genoracle`",
				// e isso é FALSO: o gerador do oráculo LÊ o despejo, não o
				// escreve — nada no repositório o escreve, ele é mantido à mão
				// junto com o arquivo do navegador. Seguir a mensagem custou uma
				// medição inteira: eu corrigi o catálogo, rodei o gerador, e o
				// oráculo saiu "sem mudança" porque o motor continuava lendo o
				// despejo velho (ALE-392).
				t.Errorf("o motor (parity/_catalogs.json → %s) e o navegador "+
					"(catalog/data/%s.json) discordam em: %s\n"+
					"Os DOIS se editam juntos — o despejo não é gerado por ninguém.",
					key, resource, ondeDiverge(a, b))
			}
		})
	}
}

// ondeDiverge nomeia as entradas em que as duas listas diferem, pelo id.
//
// Sem isto a falha diz só "discordam", e achar QUAL verbete de 462 mudou é uma
// caçada — que é o oposto do que uma mensagem de falha existe para fazer.
func ondeDiverge(a, b any) string {
	listaA, okA := a.([]any)
	listaB, okB := b.([]any)
	if !okA || !okB {
		return "as duas não são listas — compare os arquivos à mão"
	}
	porID := func(l []any) map[string]any {
		out := map[string]any{}
		for i, e := range l {
			id := fmt.Sprintf("#%d", i)
			if m, ok := e.(map[string]any); ok {
				if v, temID := m["id"]; temID {
					id = fmt.Sprintf("%v", v)
				}
			}
			out[id] = e
		}
		return out
	}
	ma, mb := porID(listaA), porID(listaB)
	nomes := []string{}
	for id, ea := range ma {
		eb, existe := mb[id]
		if !existe {
			nomes = append(nomes, id+" (só no despejo)")
		} else if !reflect.DeepEqual(ea, eb) {
			nomes = append(nomes, id)
		}
	}
	for id := range mb {
		if _, existe := ma[id]; !existe {
			nomes = append(nomes, id+" (só no catálogo)")
		}
	}
	sort.Strings(nomes)
	if len(nomes) > 8 {
		return fmt.Sprintf("%d entradas, entre elas %v", len(nomes), nomes[:8])
	}
	return fmt.Sprintf("%v", nomes)
}
