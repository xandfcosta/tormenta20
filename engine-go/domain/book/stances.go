package book

import (
	"strconv"
	"sync"
)

// AS POSTURAS DO CATÁLOGO, ligadas à flag que cada uma acende.

// Stance é uma postura do livro vista de fora: a flag que ela acende, o nome, o
// custo e a página.
type Stance struct {
	Flag string
	Name string
	PM   int
	Page int
}

var (
	stancesOnce   sync.Once
	stancesByFlag map[string]Stance
)

// StancesFromCatalog liga as ativações de `kind: "stance"` à flag que elas
// acendem.
//
// Ela morava na cena da ficha e subiu na ALE-351, pela mesma razão do
// `activation_rules.go`: quem decide entrar numa postura é o caso de uso, e ele
// não alcança o `serve/web`.
//
// A FLAG NÃO É ADIVINHADA do id: ela sai do poder de MESMO id, lendo o
// `condition.flag` dos modificadores dele. Derivar do último pedaço do id
// acertaria as duas de hoje e erraria calado na terceira.
func StancesFromCatalog() map[string]Stance {
	stancesOnce.Do(func() {
		stancesByFlag = map[string]Stance{}
		flags := ClassPowerFlags()
		for _, a := range Activations() {
			if a.Kind != "stance" {
				continue
			}
			flag := flags[a.ID]
			if flag == "" {
				flag = flags[a.ID+stanceStep(flags, a.ID)]
			}
			if flag == "" {
				continue
			}
			stancesByFlag[flag] = Stance{Flag: flag, Name: a.Name, PM: ActivationPm(a), Page: a.BookPage}
		}
	})
	return stancesByFlag
}

// stanceStep acha o sufixo do DEGRAU quando a postura não declara a flag no
// poder de id exato.
//
// O catálogo trata as duas posturas de formas DIFERENTES:
// `class.barbaro.furia` carrega os modificadores no poder de id exato, enquanto
// `class.bardo.inspiracao` os põe nos degraus numerados (`inspiracao-1`, `-2`,
// …) e deixa o id base sem modificador nenhum. Ligar só pelo id exato acha UMA
// das duas, e passa calado: a outra simplesmente não aparece na lista.
//
// O sufixo aceito é `-<dígitos>` e MAIS NADA. Um prefixo solto casaria
// `class.barbaro.furia-da-savana`, que é outro poder — e no dia em que ele
// ligasse uma flag, a postura errada herdaria a dele.
func stanceStep(flags map[string]string, base string) string {
	for i := 1; i <= 9; i++ {
		suffix := "-" + strconv.Itoa(i)
		if flags[base+suffix] != "" {
			return suffix
		}
	}
	return ""
}

// FlagGrants são as ativações de gatilho daquela flag que CONCEDEM algo.
//
// Ela NÃO filtra pelo que o personagem possui, e isso é uma folga deliberada:
// quem chega aqui já entrou na postura, e a postura é de uma classe. Filtrar
// duas vezes daria uma segunda leitura da posse.
func FlagGrants(flag string) []Activation {
	outside := []Activation{}
	for _, spec := range Activations() {
		if spec.RequiresFlag == flag && spec.Grant != nil {
			outside = append(outside, spec)
		}
	}
	return outside
}
