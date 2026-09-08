package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"t20engine/catalog"
)

// A CONFERÊNCIA DE TODA REFERÊNCIA AO CATÁLOGO, num lugar só (ALE-226).
//
// # O defeito que ela fecha
//
// O seed conferia `create.items[].catalogId` e mais nada. Um deus inexistente,
// um poder concedido escrito errado, uma magia com id inventado ou uma raça fora
// do catálogo produziam um personagem que nasce SEM aquilo e abre normal —
// nenhum erro em lugar nenhum. O relato que abriu a issue foi
// `machado-de-batalha` no lugar de `machado-batalha`.
//
// **Seed que mente é caro porque não quebra**: ele entrega um personagem quase
// certo, e o e2e roda contra a seed. Um combatente sem a arma dele vira um teste
// que mede o ambiente em vez do app — a mesma família do vermelho de CI que a
// ALE-124 e a ALE-184 já cobraram.
//
// # Um ponto só, e antes do banco
//
// Ela roda depois do `Unmarshal` e antes de o servidor subir: nenhuma linha vai
// para o banco com referência quebrada. Sete checagens espalhadas pelos pontos
// de uso deixariam a próxima referência nascer descoberta; aqui a lista de
// campos se lê de cima a baixo.
//
// # Ela junta TUDO antes de falhar
//
// Falhar no primeiro erro faria quem escreveu cinco ids errados rodar o gerador
// cinco vezes. O custo de juntar é uma passada; o de não juntar é do humano.

// referenciaQuebrada é um apontamento que não acha o que aponta.
type referenciaQuebrada struct {
	onde     string
	valor    string
	catalogo string
	vizinho  string
}

func (r referenciaQuebrada) String() string {
	msg := fmt.Sprintf("%s: %q não existe no catálogo de %s", r.onde, r.valor, r.catalogo)
	if r.vizinho != "" {
		msg += fmt.Sprintf(" — você quis dizer %q?", r.vizinho)
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
	var quebradas []referenciaQuebrada
	for iu, u := range sf.Users {
		for ic, ch := range u.Characters {
			onde := fmt.Sprintf("usuário %d (%s), personagem %d", iu+1, u.Email, ic+1)
			quebradas = append(quebradas, cat.confereUmPersonagem(onde, ch)...)
		}
	}
	if len(quebradas) == 0 {
		return nil
	}
	linhas := make([]string, 0, len(quebradas))
	for _, q := range quebradas {
		linhas = append(linhas, "  "+q.String())
	}
	return fmt.Errorf("o seed aponta para %d coisas que o catálogo não tem:\n%s",
		len(quebradas), strings.Join(linhas, "\n"))
}

// catalogoDaSeed são as listas de nomes contra as quais o seed é conferido.
// Magia e item não entram aqui: eles são procurados por id, e o `catalog` já
// tem o `LookupSpell` e o `LookupItem`.
type catalogoDaSeed struct {
	listas map[string][]string
}

// carregaOCatalogo monta as listas e AFIRMA O DENOMINADOR antes de devolvê-las.
//
// O `catalog` carrega os embeds com `sync.Once` e ENGOLE erro de leitura e de
// parse: com o embed quebrado a lista volta vazia, e um validador que leia isso
// como "nenhum valor é válido" acusaria TODOS os nomes do arquivo — culpando
// quem escreveu o seed por um defeito do build. Lista vazia aqui é falha de
// carga, e a mensagem diz isso em vez de listar dezesseis personagens.
func carregaOCatalogo() (catalogoDaSeed, error) {
	listas := map[string][]string{
		"raças":              catalog.OptionList("races"),
		"classes":            catalog.OptionList("classes"),
		"origens":            catalog.OptionList("origins"),
		"deuses":             catalog.OptionList("gods"),
		"tamanhos":           catalog.OptionList("sizes"),
		"poderes concedidos": catalog.GrantedPowerNames(),
	}
	var vazias []string
	for nome, lista := range listas {
		if len(lista) == 0 {
			vazias = append(vazias, nome)
		}
	}
	if len(vazias) > 0 {
		sort.Strings(vazias)
		return catalogoDaSeed{}, fmt.Errorf(
			"o catálogo embutido não carregou (%s vieram vazios): o defeito é do build e não do seed — "+
				"conferir se `catalog/data/*.json` foi para o binário", strings.Join(vazias, ", "))
	}
	return catalogoDaSeed{listas: listas}, nil
}

func (c catalogoDaSeed) confereUmPersonagem(onde string, ch seedCharacter) []referenciaQuebrada {
	var quebradas []referenciaQuebrada
	for _, magia := range ch.Spells {
		if _, ok := catalog.LookupSpell(magia.ID); !ok {
			quebradas = append(quebradas, referenciaQuebrada{
				onde: onde + ", spells[].id", valor: magia.ID, catalogo: "magias",
				vizinho: oVizinho(magia.ID, catalog.SpellIDs()),
			})
		}
	}
	var criar map[string]json.RawMessage
	if err := json.Unmarshal(ch.Create, &criar); err != nil {
		return append(quebradas, referenciaQuebrada{
			onde: onde + ", create", valor: err.Error(), catalogo: "JSON válido",
		})
	}
	for campo, lista := range map[string]string{
		"origin": "origens", "god": "deuses", "godPower": "poderes concedidos", "size": "tamanhos",
	} {
		if nome, ok := textoDe(criar, campo); ok {
			quebradas = append(quebradas, c.confere(onde+", create."+campo, nome, lista)...)
		}
	}
	for _, raca := range listaDe(criar, "races") {
		quebradas = append(quebradas, c.confere(onde+", create.races", raca, "raças")...)
	}
	for _, nome := range osNomesDasClasses(criar) {
		quebradas = append(quebradas, c.confere(onde+", create.classes[].className", nome, "classes")...)
	}
	// A CHAVE do `classChoices` é um nome de CLASSE — `{"Arcanista": {…}}` —, e é
	// referência de catálogo como qualquer outra. Ela escapou da primeira versão
	// porque referência escondida em CHAVE de objeto não se parece com
	// referência; só apareceu quando o guarda abaixo obrigou a classificar todo
	// campo do `create`.
	for _, nome := range asChavesDe(criar, "classChoices") {
		quebradas = append(quebradas, c.confere(onde+", create.classChoices{}", nome, "classes")...)
	}
	for _, id := range osIdsDosItens(criar) {
		if _, ok := catalog.LookupItem(id); !ok {
			quebradas = append(quebradas, referenciaQuebrada{
				onde: onde + ", create.items[].catalogId", valor: id, catalogo: "itens",
				vizinho: oVizinho(id, catalog.ItemIDs()),
			})
		}
	}
	return quebradas
}

func (c catalogoDaSeed) confere(onde, valor, catalogo string) []referenciaQuebrada {
	lista := c.listas[catalogo]
	for _, aceito := range lista {
		if aceito == valor {
			return nil
		}
	}
	return []referenciaQuebrada{{onde: onde, valor: valor, catalogo: catalogo, vizinho: oVizinho(valor, lista)}}
}

// oVizinho devolve o valor aceito mais parecido, ou "" quando nenhum é parecido
// o bastante.
//
// **O teto de um terço do comprimento é o que separa sugestão de chute.** Sem
// ele, `Anãoo` sugeriria a primeira raça da lista com a menor distância — e uma
// sugestão errada é pior que sugestão nenhuma, porque quem lê a segue. Errar id
// é erro de DIGITAÇÃO, e digitação erra por pouco: `machado-de-batalha` está a
// 3 de `machado-batalha` num catálogo que também tem `machado-guerra`.
func oVizinho(valor string, aceitos []string) string {
	melhor, menor := "", len(valor)/3+1
	for _, aceito := range aceitos {
		if d := distancia(valor, aceito); d < menor {
			melhor, menor = aceito, d
		}
	}
	return melhor
}

// distancia é a de Levenshtein, em duas linhas de matriz.
func distancia(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	anterior := make([]int, len(rb)+1)
	atual := make([]int, len(rb)+1)
	for j := range anterior {
		anterior[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		atual[0] = i
		for j := 1; j <= len(rb); j++ {
			custo := 1
			if ra[i-1] == rb[j-1] {
				custo = 0
			}
			atual[j] = min(min(atual[j-1]+1, anterior[j]+1), anterior[j-1]+custo)
		}
		anterior, atual = atual, anterior
	}
	return anterior[len(rb)]
}

func textoDe(criar map[string]json.RawMessage, campo string) (string, bool) {
	bruto, tem := criar[campo]
	if !tem {
		return "", false
	}
	var texto string
	if err := json.Unmarshal(bruto, &texto); err != nil || texto == "" {
		return "", false
	}
	return texto, true
}

func listaDe(criar map[string]json.RawMessage, campo string) []string {
	bruto, tem := criar[campo]
	if !tem {
		return nil
	}
	var lista []string
	if err := json.Unmarshal(bruto, &lista); err != nil {
		return nil
	}
	return lista
}

func osNomesDasClasses(criar map[string]json.RawMessage) []string {
	bruto, tem := criar["classes"]
	if !tem {
		return nil
	}
	var classes []struct {
		ClassName string `json:"className"`
	}
	if err := json.Unmarshal(bruto, &classes); err != nil {
		return nil
	}
	nomes := make([]string, 0, len(classes))
	for _, c := range classes {
		if c.ClassName != "" {
			nomes = append(nomes, c.ClassName)
		}
	}
	return nomes
}

func osIdsDosItens(criar map[string]json.RawMessage) []string {
	bruto, tem := criar["items"]
	if !tem {
		return nil
	}
	var itens []struct {
		CatalogID string `json:"catalogId"`
	}
	if err := json.Unmarshal(bruto, &itens); err != nil {
		return nil
	}
	ids := make([]string, 0, len(itens))
	for _, it := range itens {
		if it.CatalogID != "" {
			ids = append(ids, it.CatalogID)
		}
	}
	return ids
}

func asChavesDe(criar map[string]json.RawMessage, campo string) []string {
	bruto, tem := criar[campo]
	if !tem {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(bruto, &obj); err != nil {
		return nil
	}
	chaves := make([]string, 0, len(obj))
	for k := range obj {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	return chaves
}
