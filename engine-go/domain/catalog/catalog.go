// Package catalog serve o dado de referência transcrito do livro: magias,
// bestiário, itens, raças, origens, poderes.
//
// Ele é EMBUTIDO no binário (`go:embed`), então o servidor carrega o livro
// consigo e um deploy é um arquivo só.
package catalog

import (
	"embed"
	"encoding/json"
	"log"
	"sort"
	"sync"
)

//go:embed data/*.json
var files embed.FS

// Spell is the subset of a SPELL_CATALOG entry the API's cast/apply paths read.
type Spell struct {
	// Name é o nome do livro, e ele é TEXTO DE TELA: o extrato da manutenção
	// diz qual sustentada caiu, e "velocidade" não é o que a mesa chama de
	// Velocidade.
	Name   string `json:"name"`
	Circle int    `json:"circle"`
	School string `json:"school"`
	// Class lists the spell appears on — the per-spell PM limit is the level in
	// the class that PROVIDES the ability (p224), so the gate needs to know.
	Classes  []string  `json:"classes"`
	Augments []Augment `json:"augments"`
	Buff     *Buff     `json:"buff"`
	// Duration é a duração do livro, uma das seis da p227. Ela estava
	// transcrita nas 198 magias e NENHUM código a lia — o efeito aplicado
	// seguia um segundo campo, escrito à mão e em inglês, que divergia em oito
	// (ALE-365). Quem a traduz em duração de efeito é o `engine.EffectScope`.
	Duration string `json:"duration"`
	// DurationNote é a MEDIDA da definida, que o livro imprime em prosa no
	// verbete: "Definida. A duração pode ser medida em rodadas, horas, dias ou
	// outra unidade de tempo" (p227) nomeia a espécie e deixa o número para a
	// magia. São 26 definidas e dezesseis notas diferentes, quase todas
	// condicionais; quem separa medida de prosa é o `engine.SpellDuration`.
	DurationNote string `json:"durationNote"`
	// Execution é o TIPO DE AÇÃO que conjurar custa, uma das cinco da p233:
	// `padrao` 160, `completa` 31, `reacao` 3, `livre` 2, `movimento` 2. A
	// palavra é a do catálogo e a mesma do `engine.ActionCost` — traduzi-la na
	// leitura daria duas grafias para um conceito.
	Execution string `json:"execution"`
}

type Augment struct {
	PmCost int    `json:"pmCost"`
	Kind   string `json:"kind"`
	// RequiresCircle é o círculo MÍNIMO que o personagem precisa alcançar para
	// escolher este aprimoramento. Cento e vinte e seis dos 486 aprimoramentos o
	// têm, e um `validateAugments` que não o leia aceita qualquer um.
	//
	// Ponteiro e não zero: círculo 0 é o TRUQUE, um valor legítimo, e um `int`
	// zerado não distinguiria "exige truque" de "não exige nada".
	RequiresCircle *int `json:"requiresCircle"`
	// Exclusive é o aprimoramento que não aceita companhia na mesma conjuração.
	//
	// A marca é do APRIMORAMENTO e não do truque porque o livro diz a mesma frase
	// em três lugares: como regra geral dos truques (p171) e à mão em dois
	// aprimoramentos comuns — a esfera da Invisibilidade (p195) e a Luz
	// permanente de pó de rubi (p197). Um ramo que perguntasse "é truque?"
	// deixaria os outros dois de fora. Ver GLOSSARY, **aprimoramento exclusivo**.
	Exclusive bool `json:"exclusive"`
	// Cantrip zera o custo da magia INTEIRA, e não só o do aprimoramento: "reduz
	// seu custo em PM para zero" (p171).
	//
	// São dois campos e não um porque são duas perguntas: `Exclusive` responde se
	// ele aceita companhia (os catorze truques e mais dois), `Truque` responde
	// quanto a conjuração custa (só os catorze). Quem impede os dois de divergirem
	// é o `TestEveryTruqueIsFreeAndAlone`.
	//
	// O nome fica em português pela regra do glossário: é termo do livro sem
	// tradução assentada, como `tormenta` — e os valores deste catálogo já são
	// assim (`kind: "muda"`, `classOnly: "arcanos"`).
	Cantrip bool `json:"truque"`
}

// Buff carries the modifiers an applied spell effect stores (raw JSON so it
// re-serializes byte-identical to the catalog).
type Buff struct {
	// DefaultScope só existe onde a magia NÃO PODE dizer quanto o efeito dura:
	// a instantânea, cuja consequência não é a magia, e a definida sem quantia.
	// Nos outros 29 casos ele era uma cópia da duração e foi apagado — quem
	// recusa o retorno dele é o `TestEveryBuffLastsAsLongAsItsSpell`.
	DefaultScope string          `json:"defaultScope"`
	Modifiers    json.RawMessage `json:"modifiers"`
}

var (
	spellsOnce sync.Once
	spellsByID map[string]Spell
)

// LookupSpell returns the parsed spell entry, or (zero, false) if unknown.
func LookupSpell(id string) (Spell, bool) {
	spellsOnce.Do(func() {
		spellsByID = map[string]Spell{}
		if b, err := files.ReadFile("data/spells.json"); err == nil {
			_ = json.Unmarshal(b, &spellsByID)
		}
	})
	sp, ok := spellsByID[id]
	return sp, ok
}

// Item is the subset of a CATALOG_ITEMS entry the consume + seed paths read.
type Item struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Slots      float64     `json:"slots"`
	Consumable *Consumable `json:"consumable"`
}

// Consumable mirrors the item consumable spec (scope + instant dice + effect mods).
type Consumable struct {
	Scope      string          `json:"scope"` // 'instant' | 'scene' | 'day'
	OncePerDay bool            `json:"oncePerDay"`
	Instant    *Instant        `json:"instant"`
	Modifiers  json.RawMessage `json:"modifiers"`
}

type Instant struct {
	Hp *DiceGain `json:"hp"`
	Mp *DiceGain `json:"mp"`
}

type DiceGain struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

var (
	itemsOnce sync.Once
	itemsByID map[string]Item
)

// LookupItem returns the parsed catalog item, or (zero, false) if unknown.
func LookupItem(id string) (Item, bool) {
	itemsOnce.Do(func() {
		itemsByID = map[string]Item{}
		if b, err := files.ReadFile("data/items.json"); err == nil {
			var items []Item
			if json.Unmarshal(b, &items) == nil {
				for _, it := range items {
					itemsByID[it.ID] = it
				}
			}
		}
	})
	it, ok := itemsByID[id]
	return it, ok
}

// Activation is the subset of an ACTIVATION_SPECS entry the power-grant path reads.
type Activation struct {
	ID    string           `json:"id"`
	Grant *ActivationGrant `json:"grant"`
}

// ActivationGrant is the effect a power activation grants: a temp-HP pool scaled by an
// attribute (kind "temp-hp"), or a fixed set of active-effect modifiers (kind "active-effect").
type ActivationGrant struct {
	Kind      string          `json:"kind"`
	Scope     string          `json:"scope"`
	Attribute string          `json:"attribute"`
	Modifiers json.RawMessage `json:"modifiers"`
}

var (
	activationsOnce sync.Once
	activationsByID map[string]Activation
)

// ActivationsLoaded diz se as concessões de poder foram carregadas. Existe para
// o `/health` poder ANUNCIAR a degradação: sem elas o servidor sobe e funciona,
// mas poder nenhum concede nada, e isso morria numa linha de log.
func ActivationsLoaded() bool {
	_, _ = LookupActivation("") // força o `sync.Once`, senão isto responde antes da carga
	return len(activationsByID) > 0
}

// LookupActivation returns the parsed activation spec by id, or (zero, false) if unknown.
func LookupActivation(id string) (Activation, bool) {
	activationsOnce.Do(func() {
		activationsByID = map[string]Activation{}
		b, err := files.ReadFile("data/activations.json")
		if err != nil {
			log.Printf("catalog: read activations.json failed: %v", err)
			return
		}
		var specs []Activation
		if err := json.Unmarshal(b, &specs); err != nil {
			log.Printf("catalog: parse activations.json failed — power grants disabled: %v", err)
			return
		}
		for _, a := range specs {
			activationsByID[a.ID] = a
		}
	})
	a, ok := activationsByID[id]
	return a, ok
}

// resources é o registro ORDENADO de recursos — o índice de `GET /catalog`.
//
// Alguns são ESCRITOS AQUI em vez de despejados: eram as últimas tabelas do
// livro que um cliente importava em tempo de BUILD, e servi-las é o que encerrou
// essa dependência — o ponto era cortá-la, não os 8 KB. A forma delas é presa
// pelo `rules_tables_test.go`, a validação de schema que substitui teste de
// transcrição campo a campo.
var resources = []string{
	"spells", "bestiary", "items", "conditions", "gods", "races", "origins",
	"race-defs", "class-powers", "general-powers", "granted-powers", "origins-source",
	"tormenta-powers", "divine-powers", "activations",
	"class-expertises", "devotee-terms", "gm-tables", "dungeon-design",
	// `classes` tem três campos — id, nome e página do livro. As classes existiam
	// só como uma lista de NOMES dentro de `options.json`, e sem lugar para a
	// página não havia botão para o livro. Ver `scripts/book-pages.py`.
	"classes",
	// `effect-types` existe pelo mesmo motivo de `classes`: a condição CITA o tipo
	// ("Abalado … Medo.") e não havia para onde o elo apontar. As definições saem
	// do texto da p228, extraídas pelo `scripts/book-pages.py`.
	"effect-types",
	// `spell-schools`, idem: a magia CITA a escola e não havia para onde o elo
	// apontar — o nome dela nem aparecia no cartão. As oito definições saem do
	// texto da p172.
	"spell-schools",
	// `expertises` existiam como lista de nome e atributo dentro do `options.json`,
	// sem página e sem as duas regras que o livro imprime ao lado de cada uma — só
	// treinada e penalidade de armadura, da Tabela 2-1.
	"expertises",
}

var valid = func() map[string]bool {
	m := make(map[string]bool, len(resources))
	for _, r := range resources {
		m[r] = true
	}
	return m
}()

// Resources returns the accepted resource names (the /catalog index payload).
func Resources() []string { return resources }

// Resource returns the raw JSON for a resource, or (nil, false) if unknown.
func Resource(name string) ([]byte, bool) {
	if !valid[name] {
		return nil, false
	}
	b, err := files.ReadFile("data/" + name + ".json")
	if err != nil {
		return nil, false
	}
	return b, true
}

// Options returns the character-creation option lists JSON (/personagens/options).
func Options() ([]byte, error) {
	return files.ReadFile("data/options.json")
}

// SpellIDs e ItemIDs existem para a SUGESTÃO de vizinho: recusar um id
// desconhecido sem dizer qual é o parecido deixa quem leu a mensagem procurando
// na mão, e errar id é erro de digitação — digitação erra por pouco.
func SpellIDs() []string {
	LookupSpell("") // força o `sync.Once`, senão isto responde antes da carga
	ids := make([]string, 0, len(spellsByID))
	for id := range spellsByID {
		ids = append(ids, id)
	}
	return ids
}

func ItemIDs() []string {
	LookupItem("") // idem
	ids := make([]string, 0, len(itemsByID))
	for id := range itemsByID {
		ids = append(ids, id)
	}
	return ids
}

var (
	optionsOnce sync.Once
	optionLists map[string][]string
)

// OptionList devolve os valores aceitos de uma lista de criação — `races`,
// `classes`, `origins`, `gods`, `sizes`, `expertises` —, ou `nil` para uma lista
// que não existe.
//
// Ela existe porque o catálogo sabia procurar ITEM, MAGIA e ATIVAÇÃO e mais
// nada: um deus inventado, uma raça fora do livro ou uma origem com erro de
// digitação não tinham contra o que ser conferidos, e o seed os escrevia no
// banco em silêncio.
//
// São NOMES e não ids, e isso é do dado: o `options.json` lista "Humano",
// "Arcanista", "Acólito" — é o que a ficha grava e é o que se confere.
//
// **Lista VAZIA e lista DESCONHECIDA são coisas diferentes**, e quem chama
// precisa distinguir: com o embed quebrado toda lista vem vazia, e um chamador
// que leia isso como "nenhum valor é válido" acusa todo id do arquivo em vez de
// dizer que o catálogo não carregou. Ver o `catalogOfSeed.ausente`.
func OptionList(kind string) []string {
	optionsOnce.Do(func() {
		optionLists = map[string][]string{}
		if b, err := files.ReadFile("data/options.json"); err == nil {
			_ = json.Unmarshal(b, &optionLists)
		}
	})
	return optionLists[kind]
}

var (
	grantedOnce  sync.Once
	grantedNames []string
)

// GrantedPowerNames devolve o nome de cada poder concedido pelos deuses
// (`granted-powers.json`), que é a forma como a ficha os grava — "Bênção do
// Mana", e não um id.
func GrantedPowerNames() []string {
	grantedOnce.Do(func() {
		var list []struct {
			Name string `json:"name"`
		}
		if b, err := files.ReadFile("data/granted-powers.json"); err == nil {
			if json.Unmarshal(b, &list) == nil {
				for _, p := range list {
					grantedNames = append(grantedNames, p.Name)
				}
			}
		}
	})
	return grantedNames
}

var (
	conditionsOnce sync.Once
	conditionIDSet map[string]bool
)

// IsCondition diz se o id é uma condição do livro (p394-395).
//
// Lê do CATÁLOGO, que é onde as condições são autoradas. Uma lista escrita à mão
// ao lado da do catálogo desvia: a que faltava — `enfeitiçado` — dava 400 ao ser
// aplicada, tanto para o jogador quanto para o mestre.
func IsCondition(id string) bool {
	conditionsOnce.Do(func() {
		conditionIDSet = map[string]bool{}
		b, err := files.ReadFile("data/conditions.json")
		if err != nil {
			return
		}
		var byID map[string]json.RawMessage
		if json.Unmarshal(b, &byID) != nil {
			return
		}
		for key := range byID {
			conditionIDSet[key] = true
		}
	})
	return conditionIDSet[id]
}

// ConditionIDs lista todas as condições do livro, para o teste que precisa
// PERCORRER a tabela em vez de repeti-la.
func ConditionIDs() []string {
	ids := make([]string, 0, len(conditionIDSet))
	IsCondition("") // garante o parse antes de ler o mapa
	for id := range conditionIDSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
