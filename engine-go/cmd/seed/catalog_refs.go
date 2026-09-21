package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"t20engine/domain/catalog"
)

// A CONFERÊNCIA DE TODA REFERÊNCIA AO CATÁLOGO, num lugar só.
//
// Um deus inexistente, um poder concedido escrito errado, uma magia com id
// inventado ou uma raça fora do catálogo produzem um personagem que nasce SEM
// aquilo e abre normal — nenhum erro em lugar nenhum.
//
// SEED QUE MENTE É CARO PORQUE NÃO QUEBRA: ele entrega um personagem quase
// certo, e o e2e roda contra a seed. Um combatente sem a arma dele vira um teste
// que mede o ambiente em vez do app.
//
// Ela roda depois do `Unmarshal` e ANTES de o servidor subir: nenhuma linha vai
// para o banco com referência quebrada. Sete checagens espalhadas pelos pontos
// de uso deixariam a próxima referência nascer descoberta; aqui a lista de
// campos se lê de cima a baixo.
//
// E ela junta TUDO antes de falhar: falhar no primeiro erro faria quem escreveu
// cinco ids errados rodar o gerador cinco vezes. O custo de juntar é uma
// passada; o de não juntar é do humano.

// referenciaQuebrada é um apontamento que não acha o que aponta.
type referenciaQuebrada struct {
	where    string
	value    string
	catalog  string
	neighbor string
}

func (r referenciaQuebrada) String() string {
	msg := fmt.Sprintf("%s: %q não existe no catálogo de %s", r.where, r.value, r.catalog)
	if r.neighbor != "" {
		msg += fmt.Sprintf(" — você quis dizer %q?", r.neighbor)
	}
	return msg
}

// validateCatalogRefs recusa o seed inteiro se qualquer referência ao catálogo
// não achar o que aponta.
func validateCatalogRefs(sf seedFile) error {
	cat, err := carregaOCatalogo()
	if err != nil {
		return err
	}
	var broken []referenciaQuebrada
	for iu, u := range sf.Users {
		for ic, ch := range u.Characters {
			where := fmt.Sprintf("usuário %d (%s), personagem %d", iu+1, u.Email, ic+1)
			broken = append(broken, cat.confereUmPersonagem(where, ch)...)
		}
	}
	if len(broken) == 0 {
		return nil
	}
	rows := make([]string, 0, len(broken))
	for _, q := range broken {
		rows = append(rows, "  "+q.String())
	}
	return fmt.Errorf("o seed aponta para %d coisas que o catálogo não tem:\n%s",
		len(broken), strings.Join(rows, "\n"))
}

// catalogoDaSeed são as listas de nomes contra as quais o seed é conferido.
// Magia e item não entram aqui: eles são procurados por id, e o `catalog` já
// tem o `LookupSpell` e o `LookupItem`.
type catalogoDaSeed struct {
	lists map[string][]string
}

// carregaOCatalogo monta as listas e AFIRMA O DENOMINADOR antes de devolvê-las.
//
// O `catalog` carrega os embeds com `sync.Once` e ENGOLE erro de leitura e de
// parse: com o embed quebrado a lista volta vazia, e um validador que leia isso
// como "nenhum valor é válido" acusaria TODOS os nomes do arquivo — culpando
// quem escreveu o seed por um defeito do build. Lista vazia aqui é falha de
// carga, e a mensagem diz isso em vez de listar dezesseis personagens.
func carregaOCatalogo() (catalogoDaSeed, error) {
	lists := map[string][]string{
		"raças":              catalog.OptionList("races"),
		"classes":            catalog.OptionList("classes"),
		"origens":            catalog.OptionList("origins"),
		"deuses":             catalog.OptionList("gods"),
		"tamanhos":           catalog.OptionList("sizes"),
		"poderes concedidos": catalog.GrantedPowerNames(),
	}
	var empty []string
	for name, list := range lists {
		if len(list) == 0 {
			empty = append(empty, name)
		}
	}
	if len(empty) > 0 {
		sort.Strings(empty)
		return catalogoDaSeed{}, fmt.Errorf(
			"o catálogo embutido não carregou (%s vieram vazios): o defeito é do build e não do seed — "+
				"conferir se `catalog/data/*.json` foi para o binário", strings.Join(empty, ", "))
	}
	return catalogoDaSeed{lists: lists}, nil
}

func (c catalogoDaSeed) confereUmPersonagem(where string, ch seedCharacter) []referenciaQuebrada {
	var broken []referenciaQuebrada
	for _, spell := range ch.Spells {
		if _, ok := catalog.LookupSpell(spell.ID); !ok {
			broken = append(broken, referenciaQuebrada{
				where: where + ", spells[].id", value: spell.ID, catalog: "magias",
				neighbor: oVizinho(spell.ID, catalog.SpellIDs()),
			})
		}
	}
	var create map[string]json.RawMessage
	if err := json.Unmarshal(ch.Create, &create); err != nil {
		return append(broken, referenciaQuebrada{
			where: where + ", create", value: err.Error(), catalog: "JSON válido",
		})
	}
	for field, list := range map[string]string{
		"origin": "origens", "god": "deuses", "godPower": "poderes concedidos", "size": "tamanhos",
	} {
		if name, ok := textoDe(create, field); ok {
			broken = append(broken, c.confere(where+", create."+field, name, list)...)
		}
	}
	for _, race := range listaDe(create, "races") {
		broken = append(broken, c.confere(where+", create.races", race, "raças")...)
	}
	for _, name := range osNomesDasClasses(create) {
		broken = append(broken, c.confere(where+", create.classes[].className", name, "classes")...)
	}
	// A CHAVE do `classChoices` é um nome de CLASSE — `{"Arcanista": {…}}` —, e
	// é referência de catálogo como qualquer outra. Referência escondida em
	// CHAVE de objeto não se parece com referência, e foi assim que ela escapou
	// da primeira versão.
	for _, name := range asChavesDe(create, "classChoices") {
		broken = append(broken, c.confere(where+", create.classChoices{}", name, "classes")...)
	}
	for _, id := range osIdsDosItens(create) {
		if _, ok := catalog.LookupItem(id); !ok {
			broken = append(broken, referenciaQuebrada{
				where: where + ", create.items[].catalogId", value: id, catalog: "itens",
				neighbor: oVizinho(id, catalog.ItemIDs()),
			})
		}
	}
	return broken
}

func (c catalogoDaSeed) confere(where, value, catalog string) []referenciaQuebrada {
	list := c.lists[catalog]
	for _, accepted := range list {
		if accepted == value {
			return nil
		}
	}
	return []referenciaQuebrada{{where: where, value: value, catalog: catalog, neighbor: oVizinho(value, list)}}
}

// oVizinho devolve o valor aceito mais parecido, ou "" quando nenhum é parecido
// o bastante.
//
// **O teto de um terço do comprimento é o que separa sugestão de chute.** Sem
// ele, `Anãoo` sugeriria a primeira raça da lista com a menor distância — e uma
// sugestão errada é pior que sugestão nenhuma, porque quem lê a segue. Errar id
// é erro de DIGITAÇÃO, e digitação erra por pouco: `machado-de-batalha` está a
// 3 de `machado-batalha` num catálogo que também tem `machado-guerra`.
func oVizinho(value string, accepted []string) string {
	best, min := "", len(value)/3+1
	for _, ok := range accepted {
		if d := distancia(value, ok); d < min {
			best, min = ok, d
		}
	}
	return best
}

// distancia é a de Levenshtein, em duas linhas de matriz.
func distancia(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	anterior := make([]int, len(rb)+1)
	current := make([]int, len(rb)+1)
	for j := range anterior {
		anterior[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		current[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			current[j] = min(min(current[j-1]+1, anterior[j]+1), anterior[j-1]+cost)
		}
		anterior, current = current, anterior
	}
	return anterior[len(rb)]
}

func textoDe(create map[string]json.RawMessage, field string) (string, bool) {
	raw, found := create[field]
	if !found {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil || text == "" {
		return "", false
	}
	return text, true
}

func listaDe(create map[string]json.RawMessage, field string) []string {
	raw, found := create[field]
	if !found {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

func osNomesDasClasses(create map[string]json.RawMessage) []string {
	raw, found := create["classes"]
	if !found {
		return nil
	}
	var classes []struct {
		ClassName string `json:"className"`
	}
	if err := json.Unmarshal(raw, &classes); err != nil {
		return nil
	}
	names := make([]string, 0, len(classes))
	for _, c := range classes {
		if c.ClassName != "" {
			names = append(names, c.ClassName)
		}
	}
	return names
}

func osIdsDosItens(create map[string]json.RawMessage) []string {
	raw, found := create["items"]
	if !found {
		return nil
	}
	var items []struct {
		CatalogID string `json:"catalogId"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, it := range items {
		if it.CatalogID != "" {
			ids = append(ids, it.CatalogID)
		}
	}
	return ids
}

func asChavesDe(create map[string]json.RawMessage, field string) []string {
	raw, found := create[field]
	if !found {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
