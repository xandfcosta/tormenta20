package book

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"t20engine/catalog"
	"t20engine/creature"
	"t20engine/search"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// O BESTIÁRIO: a entrada do livro e o que se pergunta sobre ela.
//
// Ele ficou para trás quando o catálogo tipado saiu (ALE-278, segunda camada), e
// por um defeito do extrator daquela fatia — um bloco `var ( … )` acima do
// carregador fazia o regex tratar o parêntese como RECEPTOR de método, e a
// declaração sumia da lista. Ninguém percebeu porque o resultado foi um pacote
// que compilava com uma coisa a menos.
//
// O que veio é a ENTRADA e as perguntas sobre ela: o filtro por tipo e por ND, os
// rótulos que traduzem o que o catálogo guarda, e a conta de XP. O que ficou no
// `api` é a view da CENA — o cursor, os sinais e os gestos.

// verbete é uma entrada do bestiário do livro.
//
// O nome vem do GLOSSARIO.md, seção D: `verbete` é a entrada IMUTÁVEL do livro,
// `bloco de criatura` é o que o mestre escreve, e `criatura` é o guarda-chuva.
// A primeira versão disto se chamava `monstro`, que é uma quarta palavra para
// um conceito que já tem a sua.
//
// A primeira versão disto EMBUTIA o `CreatureBlock` do homebrew, com a
// justificativa de que "a ficha de uma criatura do livro e a de uma inventada
// são a mesma coisa". O dado desmente, e o `encoding/json` teria aceitado a
// mentira em SILÊNCIO — três perdas de uma vez:
//
//   - `bookPage` não existe no `CreatureBlock`, e é o que a linha mostra
//     ("p289"). Sumiria.
//   - o bloco do mestre chama os dois campos de `equipment` e `treasure`; o
//     livro grava `equipamento` e `tesouro`. Nomes diferentes não casam, e os
//     dois viriam VAZIOS — exatamente a perda que a ALE-151 consertou, com o
//     equipamento faltando nos 80 verbetes.
//   - e a pior: os atributos são `int` lá e ANULÁVEIS aqui. Nove criaturas têm
//     `inteligencia: null` e uma tem `forca: null`, porque o livro escreve
//     TRAVESSÃO — o Zumbi não tem Inteligência (p297). Num `int` isso vira 0, e
//     "+0" afirma que ele tem a média de um humano. É o defeito que a ALE-151
//     nomeia palavra por palavra, e o porte o reintroduziria.
//
// Por isso a estrutura é própria e os seis atributos são ponteiros. O
// `CreatureAttack` e o `CreatureSkill` são reusados porque esses SIM têm o
// mesmo formato nos dois lados — conferido campo a campo contra o JSON.
type Entry struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ND           float64 `json:"nd"`
	Tipo         string  `json:"tipo"`
	Size         string  `json:"size"`
	HP           int     `json:"hp"`
	Defesa       int     `json:"defesa"`
	Iniciativa   int     `json:"iniciativa"`
	Percepcao    int     `json:"percepcao"`
	Fortitude    int     `json:"fortitude"`
	Reflexos     int     `json:"reflexos"`
	Vontade      int     `json:"vontade"`
	Deslocamento string  `json:"deslocamento"`
	// Os seis que podem ser TRAVESSÃO. Ver o comentário do tipo.
	Forca        *int `json:"forca"`
	Destreza     *int `json:"destreza"`
	Constituicao *int `json:"constituicao"`
	Inteligencia *int `json:"inteligencia"`
	Sabedoria    *int `json:"sabedoria"`
	Carisma      *int `json:"carisma"`
	// PM só existe em conjurador: um zero diria "tem mana e está sem".
	PM               *int              `json:"pm,omitempty"`
	Attacks          []creature.Attack `json:"attacks"`
	Skills           []creature.Skill  `json:"skills"`
	SpecialAbilities []string          `json:"specialAbilities"`
	Equipamento      string            `json:"equipamento"`
	Tesouro          string            `json:"tesouro"`
	BookPage         int               `json:"bookPage"`
}

// WithSignPtr escreve o modificador como o livro, e o TRAVESSÃO quando ele não
// existe. Ver o comentário de `verbete`: ausência não é zero (ALE-151).
//
// O caso presente delega ao `WithSign` do `catalogs.go` em vez de repetir as
// três linhas dele. Não é economia: é que a mesma função já existia TRÊS vezes
// neste repositório — aqui, lá, e como `WithSignPtr` na view do bestiário — e a
// ficha chamava duas delas, às vezes no mesmo arquivo (ALE-278).
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
		bruto, ok := catalog.Resource("bestiary")
		if !ok {
			return
		}
		// Catálogo ausente é degradação NORMAL: a ferramenta abre vazia em vez
		// de derrubar a Mesa inteira. É a mesma decisão do `RaceTraitsByKey`.
		_ = json.Unmarshal(bruto, &bestiario)
	})
	return bestiario
}

// CreatureFilter são os quatro critérios da tela.
type CreatureFilter struct {
	Busca string
	// Tipos VAZIO significa TODOS, e não nenhum. É a convenção da tela: sem
	// crachá aceso, o filtro não filtra por tipo — tratar vazio como "nenhum"
	// mostraria bestiário vazio a quem não escolheu nada.
	Tipos []string
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
// O desempate por nome usa COLLATION pt-BR, como o `sortInitiative`, e não
// `strings.Compare`. Em bytes, "Á" (0xC3 0x81) vem depois de "Z" — então
// "Águia" cairia no fim da faixa em vez de no começo. Medi contra as 80
// criaturas de hoje e as duas ordens coincidem, mas isso é acidente do dado:
// nenhum nome do livro começa com acento AINDA, e a linha que consertaria isso
// depois seria escrita por quem visse a lista errada sem saber por quê.
// O collator nasce por chamada porque não é seguro para concorrência.
func FilterCreatures(todas []Entry, f CreatureFilter) []Entry {
	fora := make([]Entry, 0, len(todas))
	for _, m := range todas {
		if !search.Matches([]string{m.Name}, f.Busca) {
			continue
		}
		if len(f.Tipos) > 0 && !slices.Contains(f.Tipos, m.Tipo) {
			continue
		}
		if m.ND < f.NDMin || m.ND > f.NDMax {
			continue
		}
		fora = append(fora, m)
	}
	col := collate.New(language.BrazilianPortuguese)
	slices.SortStableFunc(fora, func(a, b Entry) int {
		if c := cmp.Compare(a.ND, b.ND); c != 0 {
			return c
		}
		return col.CompareString(a.Name, b.Name)
	})
	return fora
}

// faixaDeND aperta o que veio da URL para dentro dos limites do livro.
//
// Um 999 digitado ou um texto que não é número esconderia TODAS as criaturas, e
// a tela leria como "bestiário vazio" em vez de "filtro absurdo". É o mesmo
// `clampToRange` que a SPA aplica na entrada dos dois campos — só que aqui a
// entrada é a URL, que qualquer um edita à mão.
//
// A faixa INVERTIDA (min 10, max 2) devolve lista vazia, e isso é PORTE e não
// descuido: é o que a SPA faz hoje, e a tela já diz "Nenhuma criatura casa com
// os filtros", que é resposta honesta. Consertar para "faixa inteira" faria o
// filtro MENTIR — pedir 10..2 e receber tudo é pior que receber nada.
func CRRange(minBruto, maxBruto string) (float64, float64) {
	return numberOrDefault(minBruto, CRMin), numberOrDefault(maxBruto, CRMax)
}

func numberOrDefault(bruto string, padrao float64) float64 {
	if bruto == "" {
		return padrao
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(bruto), 64)
	if err != nil || n < CRMin || n > CRMax {
		return padrao
	}
	return n
}

// ── como o livro escreve ─────────────────────────────────────────────────────

// ndEscrito: abaixo de 1 o livro usa FRAÇÃO, não decimal. "ND 0.25" não existe
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

// A ordem do trilho de tipos é a do catálogo da SPA, e não alfabética: ela vai
// do mais comum na mesa para o mais raro.
var CreatureTypes = []string{"humanoide", "animal", "monstro", "morto-vivo", "construto", "espirito", "planar"}

func TypeName(tipo string) string {
	if r, ok := TypeLabels[tipo]; ok {
		return r
	}
	return tipo
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
// Portado do `xpForNd` da SPA, que cita Cap 8 p326 — a página é herdada dali e
// eu NÃO a reconferi contra o livro nesta fatia, o que fica dito porque o guia
// do `engine-go` pede citação conferida e uma repetida sem conferir parece uma
// conferida.
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
	todas := Creatures()
	for i := range todas {
		if todas[i].ID == id {
			return &todas[i]
		}
	}
	return nil
}

// ── ABRIR a ficha na hora certa (ALE-264) ────────────────────────────────────
//
// O clique NÃO abre mais a ficha; quem abre é o SERVIDOR, depois de o conteúdo
// estar remendado. O defeito que isso conserta foi visto pelo dono e medido
// depois: clicar numa linha NÃO selecionada fazia a ficha abrir na hora com a
// criatura ANTERIOR e trocar um quadro depois. Amostrado no navegador — a 0ms a
// ficha dizia "Bandido", a 16ms dizia "Lobo".
//
// Clicar na linha JÁ selecionada não piscava, e foi essa diferença que o dono
// isolou sozinho: lá o conteúdo já estava certo, então não havia troca para ver.
// É a assinatura de conteúdo obsoleto exibido antes do novo, e não de
// renderização lenta — a lentidão apareceria nas duas linhas.
//
// Custa uma ida ao servidor antes de a ficha aparecer, e é o preço certo: 16ms
// medidos contra um quadro mostrando a criatura errada. Mostrar o errado rápido
// é pior que mostrar o certo um quadro depois.
