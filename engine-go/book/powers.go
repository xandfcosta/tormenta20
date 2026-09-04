package book

import (
	"sync"
)

type Origin struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Benefits []OriginBenefit `json:"benefits"`
	// PoderUnico é o poder exclusivo da origem, e ele NÃO está na lista de
	// benefícios — é um campo à parte no catálogo. Ele conta como um dos dois
	// que a pessoa leva (p85), e esquecê-lo torna o poder da origem inescolhível.
	PoderUnico OriginBenefit `json:"poderUnico"`
	BookPage   int           `json:"bookPage"`
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
	origensPorNome        map[string]Origin
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
		origensPorNome = map[string]Origin{}
		for _, o := range ListOf[Origin]("origins") {
			origensPorNome[o.Name] = o
		}
		poderesDeClassePorID = map[string]ClassPower{}
		for _, p := range ListOf[ClassPower]("class-powers") {
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
	return origensPorNome
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
