package book

import (
	"encoding/json"
	"sync"

	"t20engine/domain/engine"
)

type Origin struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Benefits []OriginBenefit `json:"benefits"`
	// UniquePower é o poder exclusivo da origem, e ele NÃO está na lista de
	// benefícios — é um campo à parte no catálogo. Ele conta como um dos dois
	// que a pessoa leva (p85), e esquecê-lo torna o poder da origem inescolhível.
	UniquePower OriginBenefit `json:"poderUnico"`
	BookPage    int           `json:"bookPage"`
}

type OriginBenefit struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ClassPower struct {
	ID          string `json:"id"`
	ClassName   string `json:"className"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// GrantedAtLevel e GrantedByChoice são o que a REGRA DE POSSE lê — o nível
	// que concede, ou a escolha de classe que concede. Eles ficam com os campos
	// de texto porque a pergunta "eu tenho este poder?" e a pergunta "o que ele
	// faz?" são feitas na mesma linha da tela.
	GrantedAtLevel  *int           `json:"grantedAtLevel"`
	GrantedByChoice *GrantByChoice `json:"grantedByChoice"`
	BookPage        int            `json:"bookPage"`
	// Choice é a escolha que o poder pede — o atributo do Aumento de Atributo
	// (p38), o animal do Totem. Ela chega pela concessão que APONTA, porque o
	// `classPowersResolved` funde o verbete nela.
	Choice *PowerChoice `json:"choice"`
}

// PowerChoice é a escolha de um poder, como o catálogo a escreve.
type PowerChoice struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	// Repeatable é o "você pode escolher este poder várias vezes" do livro.
	Repeatable bool                `json:"repeatable"`
	Options    []PowerChoiceOption `json:"options"`
}

type PowerChoiceOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Offered são as opções que a escolha oferece. A lista do catálogo, quando ele
// a escreve; para a escolha de ARMA (Especialização em Arma, p65), as armas do
// catálogo de itens — escrevê-las de novo no verbete seria a segunda cópia de
// uma lista que já tem dono.
//
//	ClassPowers()["class.guerreiro.especializacao-em-arma"].Choice.Offered()
//	// [{adaga Adaga} {alabarda Alabarda} …]
func (c PowerChoice) Offered() []PowerChoiceOption {
	if len(c.Options) > 0 || c.Kind != "weapon" {
		return c.Options
	}
	out := []PowerChoiceOption{}
	for _, item := range Catalogs().Items {
		// O item COMPOSTO é a arma específica do capítulo 8 (a Avalanche, p337):
		// ela herda a forma da arma mundana e por isso tem `Weapon`, mas "escolha
		// uma arma" é o tipo — a espada longa, e não uma espada encantada.
		if item.Weapon != nil && item.ComposedOf == nil {
			out = append(out, PowerChoiceOption{ID: item.ID, Name: item.Name})
		}
	}
	return out
}

type GrantByChoice struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// ForEngine traduz o verbete para a forma que a REGRA DE POSSE lê.
//
// A ponte mora aqui e não em quem pergunta porque ela já tinha DOIS lugares
// para estar — a aba Poderes e o caso de uso que sobe o bônus cumulativo —, e
// o motor é puro: ele não pode importar este pacote, então alguém tem de
// atravessar. Um campo novo em `ClassPower` que não atravesse some em silêncio,
// que é o mesmo defeito que a sombra do `Modifier` já custou.
func (p ClassPower) ForEngine() *engine.ClassPower {
	out := &engine.ClassPower{
		ID: p.ID, ClassName: p.ClassName, Name: p.Name, GrantedAtLevel: p.GrantedAtLevel,
	}
	if p.GrantedByChoice != nil {
		out.GrantedByChoice = &engine.GrantedByChoice{
			Field: p.GrantedByChoice.Field, Value: p.GrantedByChoice.Value,
		}
	}
	return out
}

type GeneralPower struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description"`
	BookPage    int    `json:"bookPage"`
}

var (
	acervoDePoderesUmaVez sync.Once
	origensPorId          map[string]Origin
	poderesDeClassePorID  map[string]ClassPower
	poderesGeraisPorID    map[string]GeneralPower
)

// PowerCatalogs lê os três catálogos UMA vez, indexados por id.
//
// Por id e não por lista porque toda pergunta desta aba é "quem é este id" — a
// varredura linear que a Mochila faz custaria 462 comparações por poder numa
// ficha de nível 20, que tem trinta e poucos.
func PowerCatalogs() {
	acervoDePoderesUmaVez.Do(func() {
		// Por ID e não por nome (ALE-404): o `dto.Origin` da ficha guarda o
		// slug, e indexar pelo rótulo só funcionava enquanto o id do catálogo
		// ERA o rótulo.
		origensPorId = map[string]Origin{}
		for _, o := range ListOf[Origin]("origins") {
			origensPorId[o.ID] = o
		}
		poderesDeClassePorID = map[string]ClassPower{}
		// Não é `ListOf`: o arquivo guarda verbete e concessão em linhas
		// separadas desde a ALE-403, e quem as junta é o `classPowersResolved`.
		var poderesDeClasse []ClassPower
		if raw := classPowersResolved(); raw != nil {
			_ = json.Unmarshal(raw, &poderesDeClasse)
		}
		for _, p := range poderesDeClasse {
			poderesDeClassePorID[p.ID] = p
		}
		poderesGeraisPorID = map[string]GeneralPower{}
		for _, p := range ListOf[GeneralPower]("general-powers") {
			poderesGeraisPorID[p.ID] = p
		}
		for _, p := range ListOf[GeneralPower]("tormenta-powers") {
			p.Kind = "tormenta"
			poderesGeraisPorID[p.ID] = p
		}
	})
}

func Origins() map[string]Origin {
	PowerCatalogs()
	return origensPorId
}

func ClassPowers() map[string]ClassPower {
	PowerCatalogs()
	return poderesDeClassePorID
}

func GeneralPowers() map[string]GeneralPower {
	PowerCatalogs()
	return poderesGeraisPorID
}

// ── o que a RAÇA pede escolher ───────────────────────────────────────────────

// OriginLabel devolve o rótulo da origem a partir do que a ficha guarda.
//
// Exemplo: `OriginLabel("heroi-campones")` devolve "Herói Camponês". Devolve a
// própria chave quando o catálogo não a conhece, pela mesma razão do
// `RaceLabel`.
func OriginLabel(originKey string) string {
	if origin, ok := Origins()[originKey]; ok && origin.Name != "" {
		return origin.Name
	}
	return originKey
}
