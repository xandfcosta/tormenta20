package engine

import (
	"fmt"
	"sort"
)

// O OBJETO COMO ALVO: a Tabela 5-4, "Estatísticas de Objetos" (p239).
//
// Ela existe porque o livro manda atacar coisa que não é criatura: *"Para
// objetos soltos, faça um ataque contra a Defesa do objeto, definida por sua
// categoria de tamanho. Se o objeto estiver em movimento, recebe +5 na Defesa.
// [...] objetos normalmente têm redução de dano, dependendo de seu material. Um
// objeto reduzido a 0 ou menos PV é destruído."*
//
// # Por que isto não é ECS, e a decisão é a do guia
//
// *ECS para o que COMPÕE, função para o que calcula* — e o `engine-go/CLAUDE.md`
// nomeia esta saída por extenso: "uma tabela fechada que só se lê". Não há
// entidade aqui. A Defesa de uma porta não é composta de parcelas que sistemas
// diferentes contribuem: ela É o degrau do tamanho dela, e um sistema que
// copiasse um número de um mapa para um componente seria cerimônia com um
// guarda de denominador de brinde.
//
// O que COMPÕE é o ataque CONTRA o objeto, e esse mundo já existe — por isso o
// +5 do objeto em movimento é uma linha da Tabela 5-3 (`TargetObjectInMotion`) e
// não um `if` deste arquivo. A alternativa poria o mesmo mecanismo em dois
// lugares, e o número que a mesa lê na decomposição viria só de um deles.
//
// # Duas escadas e uma coluna
//
// O livro imprime quatro colunas e parece transcrição. Duas são REGRA: a página
// diz "definida por sua categoria de tamanho" com essas palavras, e diz que a RD
// depende do material. Guardar Defesa e RD por verbete daria duas grafias do
// mesmo número — a armadilha que o `material_rules_test.go` já documenta para o
// preço dos materiais especiais —, e a que diverge sozinha é a que ninguém abre.
//
// Sobra o PV, que é o único número que o livro dá linha a linha, e é por isso que
// o `ObjectExample` tem três campos e não cinco.

// defenseByObjectSize é a coluna "Def" da Tab. 5-4, lida como o que ela é: uma
// função do tamanho. Ela DESCE — objeto grande é mais fácil de acertar.
//
// Chaveada pela forma normalizada do tamanho, que é a mesma do
// `FootprintForSize`: o bestiário guarda "medio" e a ficha guarda "Médio", e
// normalizar num lugar só é mais barato que descobrir a divergência na tela.
var defenseByObjectSize = map[string]int{
	"minusculo": 15,
	"pequeno":   12,
	"medio":     10,
	"grande":    8,
	"enorme":    5,
	"colossal":  0,
}

// damageReductionByObjectMaterial é a coluna "RD", e ela é DERIVADA das 11
// linhas impressas da primeira metade da tabela — a página diz que a RD "depende
// do material" e não tabula o material, então a escada é leitura e não cópia.
//
// Quem a defende é o `TestEveryPrintedRowOfTheObjectTableIsReproduced`, que
// confere os cinco degraus contra as 11 linhas do livro e reprova nomeando a
// linha que discordar. O auditor `scripts/audit-objects.py` relê a página.
//
// PAPEL E FIBRA são dois degraus no mesmo zero, e não um degrau ausente: o
// pergaminho e a corda têm RD 0 impressa, e um mapa sem eles obrigaria o mestre
// a mentir o material de uma corda para conseguir a RD certa.
//
// O `material` de um OBJETO não é o `material` de um ITEM de ficha — mitral e
// aço-rubi são a p166, e esta é a p239. A colisão é a C10 do GLOSSARY.md, e ela
// está registrada em vez de resolvida por decisão do dono: as duas palavras vêm
// do livro e nunca aparecem na mesma tela.
var damageReductionByObjectMaterial = map[string]int{
	"papel":   0,
	"fibra":   0,
	"madeira": 5,
	"pedra":   8,
	"metal":   10,
}

// ObjectDefense é a Defesa de um objeto, pelo tamanho dele (p239).
//
// Recusa o tamanho que não conhece, com o valor ofensor: um `default` devolvendo
// zero daria Defesa 0 — acerto garantido — para um tamanho escrito errado, e em
// silêncio. Lista de PERMITIDOS, pela mesma razão.
//
//	defesa, err := engine.ObjectDefense("Grande") // → 8
func ObjectDefense(size string) (int, error) {
	if defense, known := defenseByObjectSize[normalizeSizeKey(size)]; known {
		return defense, nil
	}
	return 0, fmt.Errorf("%q não é um tamanho de objeto: a Tab. 5-4 (p239) tem %v",
		size, objectSizesOfTheBook())
}

// ObjectDamageReduction é a RD de um objeto, pelo material dele (p239).
//
//	rd, err := engine.ObjectDamageReduction("madeira") // → 5
func ObjectDamageReduction(material string) (int, error) {
	if rd, known := damageReductionByObjectMaterial[normalizeSizeKey(material)]; known {
		return rd, nil
	}
	return 0, fmt.Errorf("%q não é um material de objeto: a Tab. 5-4 (p239) tem %v "+
		"(material especial de item é outra coisa — ver a colisão C10)",
		material, ObjectMaterialsOfTheBook())
}

// ObjectTarget monta o alvo de um ataque contra um objeto SOLTO (p239).
//
// Ele não sabe do +5 do objeto em movimento de propósito: isso é uma linha da
// Tabela 5-3, chega como `TargetObjectInMotion`, e quem decide se a peça está
// rolando é o tabuleiro.
//
//	alvo, err := engine.ObjectTarget("Grande", "madeira") // porta: Defesa 8, RD 5
func ObjectTarget(size, material string) (AttackTarget, error) {
	defense, err := ObjectDefense(size)
	if err != nil {
		return AttackTarget{}, err
	}
	rd, err := ObjectDamageReduction(material)
	if err != nil {
		return AttackTarget{}, err
	}
	// CritImmune porque objeto não tem ponto vital — e isto NÃO é invenção
	// nossa: a p231 dá a imunidade a "certas criaturas", e o objeto não é
	// criatura nenhuma. Deixá-lo crítico faria um barril sangrar o dobro.
	return AttackTarget{Defense: defense, DamageReduction: rd, CritImmune: true}, nil
}

// ObjectIsDestroyed: *"Um objeto reduzido a 0 ou menos PV é destruído"* (p239).
//
// O limiar é ZERO e não um negativo. Objeto não sangra e não tem o −10 da p236:
// a criatura a 0 PV cai e pode ser estabilizada, e a porta a 0 PV acabou.
func ObjectIsDestroyed(hitPoints int) bool { return hitPoints <= 0 }

// ObjectExample é uma das 11 linhas da primeira metade da Tab. 5-4.
//
// Três campos e não cinco: Defesa e RD são as escadas acima, e repeti-las aqui
// seria a segunda grafia. O PV é o que só o verbete sabe.
type ObjectExample struct {
	// ID é o slug, para a tela apontar sem carregar o rótulo.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Size e Material na grafia do LIVRO, que é a que a tela mostra — as escadas
	// normalizam na entrada.
	Size      string `json:"size"`
	Material  string `json:"material"`
	HitPoints int    `json:"hitPoints"`
}

// objectExamplesOfTheBook é a primeira metade da Tab. 5-4, transcrita.
//
// Em Go e não em JSON, pela mesma razão da `specialSituationTable` ao lado: a
// lista existe sobretudo como EVIDÊNCIA das duas escadas — é ela que o caso de
// decomposição percorre —, e um catálogo em `domain/catalog/data/` pediria
// carregador, esquema e auditor para onze linhas cujo outro leitor é um menu.
//
// A SEGUNDA METADE da tabela (armas, armaduras e escudos) não está aqui, e a
// ausência é escopo e não esquecimento: ela é o alvo da manobra QUEBRAR, que
// atinge "um item que a criatura esteja segurando" (p234) e ainda não tem onde
// guardar dano — `CharacterItem` não tem PV. A nota de pé da tabela (÷2
// reduzido, ×2 aumentado, ×5 gigante) tem asterisco só naquela metade, e vem
// com ela.
var objectExamplesOfTheBook = []ObjectExample{
	{ID: "pergaminho", Name: "Pergaminho", Size: "Minúsculo", Material: "papel", HitPoints: 1},
	{ID: "corda", Name: "Corda", Size: "Minúsculo", Material: "fibra", HitPoints: 2},
	{ID: "corrente", Name: "Corrente", Size: "Minúsculo", Material: "metal", HitPoints: 2},
	{ID: "cadeira", Name: "Cadeira", Size: "Pequeno", Material: "madeira", HitPoints: 5},
	{ID: "barril", Name: "Barril", Size: "Médio", Material: "madeira", HitPoints: 10},
	{ID: "porta-de-madeira", Name: "Porta de madeira", Size: "Grande", Material: "madeira", HitPoints: 20},
	{ID: "porta-de-pedra", Name: "Porta de pedra", Size: "Grande", Material: "pedra", HitPoints: 100},
	{ID: "porta-de-ferro", Name: "Porta de ferro", Size: "Grande", Material: "metal", HitPoints: 100},
	{ID: "carroca", Name: "Carroça", Size: "Grande", Material: "madeira", HitPoints: 50},
	{ID: "casebre", Name: "Casebre", Size: "Enorme", Material: "madeira", HitPoints: 100},
	{ID: "celeiro", Name: "Celeiro", Size: "Colossal", Material: "madeira", HitPoints: 200},
}

// ObjectExamplesOfTheBook são os 11 exemplos, na ordem do livro.
//
// Eles são ATALHO e não lista fechada: a coluna da tabela se chama *Exemplo*, e
// o braço de estátua que o livro não imprimiu também é objeto. Quem escolhe
// tamanho, material e PV é o mestre; isto preenche os três de uma vez.
//
//	for _, exemplo := range engine.ObjectExamplesOfTheBook() { … }
func ObjectExamplesOfTheBook() []ObjectExample {
	out := make([]ObjectExample, len(objectExamplesOfTheBook))
	copy(out, objectExamplesOfTheBook)
	return out
}

// ObjectMaterialsOfTheBook são os materiais que dão RD a um objeto, ordenados
// pela RD que cada um concede — que é a ordem em que a tela os oferece, porque é
// a única que significa algo para quem escolhe.
//
//	engine.ObjectMaterialsOfTheBook() // → [papel fibra madeira pedra metal]
func ObjectMaterialsOfTheBook() []string {
	out := make([]string, 0, len(damageReductionByObjectMaterial))
	for material := range damageReductionByObjectMaterial {
		out = append(out, material)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := damageReductionByObjectMaterial[out[i]], damageReductionByObjectMaterial[out[j]]
		if a != b {
			return a < b
		}
		return out[i] < out[j]
	})
	return out
}

// objectSizesOfTheBook são os seis tamanhos, do menor para o maior — que é a
// ordem da tabela e a INVERSA da Defesa.
func objectSizesOfTheBook() []string {
	out := make([]string, 0, len(defenseByObjectSize))
	for size := range defenseByObjectSize {
		out = append(out, size)
	}
	sort.Slice(out, func(i, j int) bool {
		return defenseByObjectSize[out[i]] > defenseByObjectSize[out[j]]
	})
	return out
}
