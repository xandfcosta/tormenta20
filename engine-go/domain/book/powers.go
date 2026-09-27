package book

import (
	"encoding/json"
	"sync"
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
}

type GrantByChoice struct {
	Field string `json:"field"`
	Value string `json:"value"`
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
