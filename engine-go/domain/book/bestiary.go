package book

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"t20engine/domain/catalog"
	"t20engine/domain/creature"
	"t20engine/domain/search"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// O BESTIÁRIO: a entrada do livro e o que se pergunta sobre ela — o filtro por
// tipo e por ND, os rótulos que traduzem o que o catálogo guarda, e a conta de
// XP. O que mora no `api` é a view da CENA: o cursor, os sinais e os gestos.

// Entry é o verbete: uma entrada do bestiário do livro.
//
// O nome vem do GLOSSARY.md, seção D: `verbete` é a entrada IMUTÁVEL do livro,
// `bloco de criatura` é o que o mestre escreve, e `criatura` é o guarda-chuva.
//
// Estrutura PRÓPRIA e não o `CreatureBlock` do homebrew, e o `encoding/json`
// aceitaria a troca em SILÊNCIO — são três perdas de uma vez:
//
//   - `bookPage` não existe no `CreatureBlock`, e é o que a linha mostra
//     ("p289"). Sumiria.
//   - o bloco do mestre chama os dois campos de `equipment` e `treasure`; o
//     livro grava `equipamento` e `tesouro`. Nomes diferentes não casam, e os
//     dois viriam VAZIOS.
//   - e a pior: os atributos são `int` lá e ANULÁVEIS aqui. Nove criaturas têm
//     `inteligencia: null` e uma tem `forca: null`, porque o livro escreve
//     TRAVESSÃO — o Zumbi não tem Inteligência (p297). Num `int` isso vira 0, e
//     "+0" afirma que ele tem a média de um humano.
//
// Por isso os seis atributos são ponteiros. O `creature.Attack` e o
// `creature.Skill` são reusados porque esses SIM têm o mesmo formato nos dois
// lados — conferido campo a campo contra o JSON.
type Entry struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ND         float64 `json:"nd"`
	Kind       string  `json:"tipo"`
	Size       string  `json:"size"`
	HP         int     `json:"hp"`
	Defense    int     `json:"defesa"`
	Initiative int     `json:"iniciativa"`
	Perception int     `json:"percepcao"`
	Fortitude  int     `json:"fortitude"`
	Reflex     int     `json:"reflexos"`
	Will       int     `json:"vontade"`
	Speed      string  `json:"deslocamento"`
	// Os seis que podem ser TRAVESSÃO. Ver o comentário do tipo.
	Strength     *int `json:"forca"`
	Dexterity    *int `json:"destreza"`
	Constitution *int `json:"constituicao"`
	Intelligence *int `json:"inteligencia"`
	Wisdom       *int `json:"sabedoria"`
	Charisma     *int `json:"carisma"`
	// PM só existe em conjurador: um zero diria "tem mana e está sem".
	PM               *int              `json:"pm,omitempty"`
	Attacks          []creature.Attack `json:"attacks"`
	Skills           []creature.Skill  `json:"skills"`
	SpecialAbilities []string          `json:"specialAbilities"`
	Equipment        string            `json:"equipamento"`
	Treasure         string            `json:"tesouro"`
	BookPage         int               `json:"bookPage"`
}

// WithSignPtr escreve o modificador como o livro, e o TRAVESSÃO quando ele não
// existe: ausência não é zero.
//
// O caso presente delega ao `WithSign` em vez de repetir as três linhas dele —
// a mesma função já existiu TRÊS vezes neste repositório, e a ficha chamava duas
// delas, às vezes no mesmo arquivo.
func WithSignPtr(n *int) string {
	if n == nil {
		return "—"
	}
	return WithSign(*n)
}

var (
	bestiarioUmaVez sync.Once
	bestiario       []Entry
)

func Creatures() []Entry {
	bestiarioUmaVez.Do(func() {
		raw, ok := catalog.Resource("bestiary")
		if !ok {
			return
		}
		// Catálogo ausente é degradação NORMAL: a ferramenta abre vazia em vez
		// de derrubar a Mesa inteira. É a mesma decisão do `RaceTraitsByKey`.
		_ = json.Unmarshal(raw, &bestiario)
	})
	return bestiario
}

// CreatureFilter são os quatro critérios da tela.
type CreatureFilter struct {
	Search string
	// Kinds VAZIO significa TODOS, e não nenhum. É a convenção da tela: sem
	// crachá aceso, o filtro não filtra por tipo — tratar vazio como "nenhum"
	// mostraria bestiário vazio a quem não escolheu nada.
	Kinds []string
	NDMin float64
	NDMax float64
}

const (
	CRMin = 0.0
	CRMax = 20.0
)

// FilterCreatures aplica os critérios e ORDENA por ND e depois por nome.
//
// A ordem é regra e não apresentação: o mestre procura por desafio, e uma lista
// alfabética o faria ler 80 linhas para achar as de ND 3.
//
// O desempate por nome usa COLLATION pt-BR e não `strings.Compare`: em bytes,
// "Á" (0xC3 0x81) vem depois de "Z", e "Águia" cairia no fim da faixa em vez do
// começo. Nenhum nome do livro começa com acento AINDA, e é justamente por isso
// que a linha errada passaria despercebida.
//
// O collator nasce por chamada porque não é seguro para concorrência.
func FilterCreatures(all []Entry, f CreatureFilter) []Entry {
	outside := make([]Entry, 0, len(all))
	for _, m := range all {
		if !search.Matches([]string{m.Name}, f.Search) {
			continue
		}
		if len(f.Kinds) > 0 && !slices.Contains(f.Kinds, m.Kind) {
			continue
		}
		if m.ND < f.NDMin || m.ND > f.NDMax {
			continue
		}
		outside = append(outside, m)
	}
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(outside, func(a, b Entry) int {
		if c := cmp.Compare(a.ND, b.ND); c != 0 {
			return c
		}
		return col.CompareString(a.Name, b.Name)
	})
	return outside
}

// CRRange aperta o que veio da URL para dentro dos limites do livro.
//
// Um 999 digitado ou um texto que não é número esconderia TODAS as criaturas, e
// a tela leria como "bestiário vazio" em vez de "filtro absurdo". A entrada aqui
// é a URL, que qualquer um edita à mão.
//
// A faixa INVERTIDA (min 10, max 2) devolve lista vazia de propósito: a tela já
// diz "Nenhuma criatura casa com os filtros", que é resposta honesta. Consertar
// para "faixa inteira" faria o filtro MENTIR — pedir 10..2 e receber tudo é pior
// que receber nada.
func CRRange(rawMin, rawMax string) (float64, float64) {
	return numberOrDefault(rawMin, CRMin), numberOrDefault(rawMax, CRMax)
}

func numberOrDefault(raw string, standard float64) float64 {
	if raw == "" {
		return standard
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || n < CRMin || n > CRMax {
		return standard
	}
	return n
}

// ── como o livro escreve ─────────────────────────────────────────────────────

// CRWritten: abaixo de 1 o livro usa FRAÇÃO, não decimal. "ND 0.25" não existe
// em lugar nenhum de Tormenta 20 — a mesa diz "ND 1/4".
func CRWritten(nd float64) string {
	switch {
	case roughly(nd, 0.25):
		return "1/4"
	case roughly(nd, 0.5):
		return "1/2"
	case nd == float64(int(nd)):
		return strconv.Itoa(int(nd))
	default:
		return strconv.FormatFloat(nd, 'g', -1, 64)
	}
}

// A comparação é por PROXIMIDADE porque 0.25 e 0.5 vêm de JSON como float, e
// igualdade exata de ponto flutuante é a armadilha clássica desse caminho.
func roughly(a, b float64) bool {
	d := a - b
	return d < 0.001 && d > -0.001
}

// Os rótulos dos tipos. O dado vem do catálogo sem acento e em caixa baixa; a
// tela mostra como se escreve.
var TypeLabels = map[string]string{
	"humanoide":  "Humanoide",
	"animal":     "Animal",
	"monstro":    "Monstro",
	"morto-vivo": "Morto-vivo",
	"construto":  "Construto",
	"espirito":   "Espírito",
	"planar":     "Planar",
}

// A ordem do trilho de tipos não é alfabética: ela vai do mais comum na mesa
// para o mais raro.
var CreatureTypes = []string{"humanoide", "animal", "monstro", "morto-vivo", "construto", "espirito", "planar"}

func TypeName(kind string) string {
	if r, ok := TypeLabels[kind]; ok {
		return r
	}
	return kind
}

var sizeLabels = map[string]string{
	"minusculo": "Minúsculo",
	"pequeno":   "Pequeno",
	"medio":     "Médio",
	"grande":    "Grande",
	"enorme":    "Enorme",
	"colossal":  "Colossal",
}

func SizeName(t string) string {
	if r, ok := sizeLabels[t]; ok {
		return r
	}
	return t
}

// XPForCR é o XP de tesouro derivado do ND.
//
// A citação de Cap 8 p326 é HERDADA e NÃO foi reconferida contra o livro — fica
// dito porque uma página repetida sem conferir parece uma conferida.
func XPForCR(nd float64) int {
	return int(math.Round(nd * 1000))
}

// ── a cena ───────────────────────────────────────────────────────────────────

// EntryByID acha a criatura do livro, ou nil.
//
// Nil e não erro: quem chama decide o que dizer. O painel da Mesa recusa a
// entrada com o id na frase, porque ali um id desconhecido só chega por adulteração.
func EntryByID(id string) *Entry {
	if id == "" {
		return nil
	}
	all := Creatures()
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// ── ABRIR a ficha na hora certa ──────────────────────────────────────────────
//
// O clique NÃO abre a ficha; quem abre é o SERVIDOR, depois de o conteúdo estar
// remendado. Abrindo no clique, clicar numa linha NÃO selecionada mostra a
// criatura ANTERIOR e troca um quadro depois — a 0ms a ficha dizia "Bandido", a
// 16ms dizia "Lobo". Clicar na linha JÁ selecionada não piscava, e é essa
// diferença que separa conteúdo obsoleto exibido antes do novo de renderização
// lenta: a lentidão apareceria nas duas linhas.
//
// Custa uma ida ao servidor antes de a ficha aparecer, e é o preço certo:
// mostrar o errado rápido é pior que mostrar o certo um quadro depois.
