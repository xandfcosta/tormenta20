package book

import (
	"encoding/json"
	"sync"

	"t20engine/domain/catalog"
)

// O REGISTRO DE ATIVAÇÕES: o que um poder custa para ser usado.
//
// A tabela vem do `catalog/data/activations.json` — 411 entradas — e é o que
// responde "este poder se ativa, e com quê". Ela mora aqui e não na cena porque
// **o destino de uma função é a DEPENDÊNCIA dela**: quem lê o catálogo é do
// livro.
//
// O que NÃO mora aqui é a metade que a cena decide: se o botão está ativo, que
// crachá o limite desenha, e a frase da recusa. Aquilo lê esta tabela e a ficha,
// e a voz é da tela.

// Activation é a entrada do registro de ativações: id, nome, tipo, PM e página,
// mais a AÇÃO que o uso consome, o limite, a flag que o gatilho exige e a escala
// da postura.
type Activation struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
	// PmCost é CRU porque o catálogo escreve duas coisas nele: um número, ou a
	// palavra "variavel" — 33 das 411 ativações são variáveis. Tipar como `int`
	// NÃO estoura: o `json.Unmarshal` de uma lista guarda o erro de tipo daquele
	// campo, segue em frente e deixa ZERO no lugar. O poder de custo variável
	// apareceria como "0 PM" com o botão ATIVO, e usá-lo não cobraria nada.
	PmCost json.RawMessage `json:"pmCost"`
	// Uses é `null`, "cena", "dia" ou um número — por isso ele é cru: os três
	// significam coisas diferentes e só dois são cobrados.
	Uses         json.RawMessage  `json:"uses"`
	RequiresFlag string           `json:"requiresFlag"`
	Scaling      *ActivationScale `json:"scaling"`
	Grant        *PowerGrant      `json:"grant"`
	BookPage     int              `json:"bookPage"`
}

// ActivationScale é a postura que sobe de degrau com o nível.
type ActivationScale struct {
	BasePm          int    `json:"basePm"`
	StepPm          int    `json:"stepPm"`
	StepLabel       string `json:"stepLabel"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
}

// PowerGrant é o que a ativação APLICA na ficha — o único caso hoje é a
// reserva de PV temporários da Alma de Bronze.
type PowerGrant struct {
	Kind      string `json:"kind"`
	Scope     string `json:"scope"`
	Attribute string `json:"attribute"`
}

var (
	ativacoesUmaVez  sync.Once
	ativacoesPorID   map[string]Activation
	ativacoesPorNome map[string]Activation
)

func loadTheActivations() {
	ativacoesUmaVez.Do(func() {
		ativacoesPorID = map[string]Activation{}
		ativacoesPorNome = map[string]Activation{}
		for _, a := range Activations() {
			ativacoesPorID[a.ID] = a
			ativacoesPorNome[a.Name] = a
		}
	})
}

func Activations() []Activation {
	bruto, ok := catalog.Resource("activations")
	if !ok {
		return nil
	}
	var lista []Activation
	_ = json.Unmarshal(bruto, &lista)
	return lista
}

// ActivationOf acha a ativação de um poder pelo ID e, se falhar, pelo NOME.
//
// A queda para o nome não é preguiça: os poderes de classe seguem a convenção
// `class.<classe>.<slug>` e casam por id, mas as habilidades de raça e os
// benefícios de origem têm ids próprios (`humano-versatil`,
// `origin-acolito-...`) que o registro de ativações não usa.
//
// E há o caso dos DEGRAUS: "Inspiração +1" e "Fúria +3" são linhas de mesma
// postura, e o nome delas traz o sufixo do degrau. Sem tirar o sufixo, cada
// degrau cairia como passiva silenciosa em vez de resolver para a postura.
func ActivationOf(id, nome string) *Activation {
	loadTheActivations()
	if spec, tem := ativacoesPorID[id]; tem && id != "" {
		return &spec
	}
	if spec, tem := ativacoesPorNome[nome]; tem {
		return &spec
	}
	if semDegrau := suffixStepSem(nome); semDegrau != nome {
		if spec, tem := ativacoesPorNome[semDegrau]; tem {
			return &spec
		}
	}
	return nil
}

// suffixStepSem tira o " +N" do fim de "Inspiração +2".
func suffixStepSem(nome string) string {
	if len(nome) < 4 {
		return nome
	}
	fim := len(nome)
	for fim > 0 && nome[fim-1] >= '0' && nome[fim-1] <= '9' {
		fim--
	}
	if fim == len(nome) || fim < 2 || nome[fim-1] != '+' || nome[fim-2] != ' ' {
		return nome
	}
	return nome[:fim-2]
}
