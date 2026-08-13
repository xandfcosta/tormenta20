//go:generate go run ../cmd/tsgen

package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Geração dos tipos TypeScript da FRONTEIRA do motor — as formas que atravessam
// o WASM e o `/characters/:id/sheet`.
//
// Por que só a fronteira, e não os catálogos (ALE-108): o Go serve os catálogos
// como BYTES CRUS — lê o arquivo e escreve, sem parsear — então para a maioria
// deles não existe struct nenhuma aqui, e as três que existem (Spell, Item,
// Activation) são subconjuntos DELIBERADOS do que os caminhos de conjurar e
// aplicar leem. Gerar TS a partir delas produziria tipos sem `name`, `bookPage`
// ou `baseEffect`; completá-las obrigaria o motor a carregar campos que ele
// nunca lê, invertendo o motivo de serem subconjunto. Esses ficam escritos à
// mão em `shared/api`.
//
// Aqui é o contrário: o Go É a autoridade, as structs são completas, e uma
// divergência não dá erro de compilação — dá número errado na tela. Hoje os
// oráculos de paridade pegariam, mas eles morrem junto com o TS na fatia 5.
//
// O arquivo gerado é COMMITADO e conferido por `TestGeneratedTypesAreCurrent`,
// que falha mandando regenerar — o mesmo contrato de espelho dos oráculos.

// tsWireOverrides mapeia um tipo com `MarshalJSON` próprio para a struct que ele
// de fato serializa. Sem isto o emissor descreveria a forma EM MEMÓRIA — para o
// ItemEffects, `Flags: Record<string, boolean>` em vez do `flags: string[]` que
// vai no fio. Um tipo gerado que mente é pior que um escrito à mão, então o
// emissor EXIGE a entrada e falha sem ela.
var tsWireOverrides = map[string]reflect.Type{
	"ItemEffects": reflect.TypeOf(ItemEffectsWire{}),
}

// tsKeyOverrides estreita a CHAVE de um Record onde o Go só sabe dizer "string".
// `attributes` é indexado por atributo, não por qualquer string, e perder essa
// união no TS troca um erro de compilação por um `undefined` em runtime.
// Chaveado por "Tipo.campo" (o nome já no formato do fio).
var tsKeyOverrides = map[string]string{
	"ComputedSheetV2.attributes":         "AttributeKey",
	"ComputedSheetV2.spellCdByAttribute": "AttributeKey",
	"VitalContext.attrTotals":            "AttributeKey",
}

// tsFieldTypeOverrides estreita o TIPO de um campo pelo mesmo motivo do
// tsKeyOverrides: o Go guarda o atributo como `string`, mas o campo só admite os
// seis. O tipo escrito à mão dizia `AttributeKey` aqui, e perder isso quebrou
// três telas que indexam a tabela de abreviações — o typecheck acusou na hora.
var tsFieldTypeOverrides = map[string]string{
	"ExpertiseBreakdown.attribute": "AttributeKey",
	"WeaponCard.attribute":         "AttributeKey",
	// O bonusType decide o EMPILHAMENTO — um valor fora da lista viraria um
	// balde novo, somando em silêncio em vez de competir. Como `string` isso não
	// dá erro nenhum.
	"Modifier.bonusType":                "BonusType",
	"Contribution.bonusType":            "BonusType",
	"ConditionalEffect.bonusType":       "BonusType",
	"ConditionalDisplayInput.bonusType": "BonusType",
	"ActiveItem.equipped":               "EquipSlot | null",
}

// tsImported são tipos que o arquivo gerado REFERENCIA mas não declara — eles
// vivem no TS escrito à mão porque são uniões de literais, que uma struct Go não
// consegue expressar.
var tsImported = []string{"AttributeKey"}

// tsImportedFrom diz de qual módulo escrito à mão cada tipo importado vem.
var tsBonusImported = []string{"BonusType"}

// tsItemImported vem do vocabulário de item/modificador escrito à mão.
var tsItemImported = []string{"ModifierTarget"}

// tsNeverEmit são tipos que o arquivo gerado REFERENCIA mas nunca declara —
// eles vêm dos módulos escritos à mão porque são uniões que uma struct Go não
// expressa. Sem esta lista o emissor os declararia na forma achatada do Go e o
// TS acusaria conflito com o import.
var tsNeverEmit = map[string]bool{
	"ModifierTarget": true,
}

// tsRootTypes são as raízes da fronteira: tudo o que o WASM devolve, mais as
// entradas que ele recebe. O emissor caminha os campos e arrasta o que for
// preciso, então acrescentar um campo novo não pede mudança aqui.
func tsRootTypes() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(ComputedSheetV2{}),
		reflect.TypeOf(ItemEffects{}),
		reflect.TypeOf(VitalPools{}),
		reflect.TypeOf(VitalContext{}),
		reflect.TypeOf(EquippedFlag{}),
		reflect.TypeOf(WeaponCard{}),
		reflect.TypeOf(PointBuyStatus{}),
		reflect.TypeOf(ConditionalDisplayRow{}),
		reflect.TypeOf(ConditionalDisplayInput{}),
	}
}

const tsHeader = `// GERADO por engine-go — NÃO EDITE À MÃO.
//
// Regenere com:  cd engine-go && go generate ./engine
//
// As formas que atravessam a fronteira do motor (WASM + /sheet). O Go é a
// autoridade delas; os tipos de CATÁLOGO ficam à mão em rules-tables.ts e
// catalog.ts, porque o servidor os entrega como bytes crus e não tem struct
// para eles (ALE-108).
`

// GenerateTypeScript emite as declarações TS da fronteira, em ordem estável.
func GenerateTypeScript() string {
	emitted := map[string]string{}
	for _, root := range tsRootTypes() {
		collectTSType(root, emitted)
	}
	names := make([]string, 0, len(emitted))
	for name := range emitted {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString(tsHeader)
	b.WriteString("\nimport type { " + strings.Join(tsImported, ", ") + " } from './attribute-keys'\n")
	b.WriteString("import type { " + strings.Join(tsBonusImported, ", ") + " } from './bonus-types'\n")
	b.WriteString("import type { " + strings.Join(tsItemImported, ", ") + " } from './item-types'\n")
	for _, name := range names {
		b.WriteString("\n")
		b.WriteString(emitted[name])
	}
	return b.String()
}

// collectTSType registra a declaração de um struct e, recursivamente, a de todo
// struct que ele alcança.
func collectTSType(t reflect.Type, emitted map[string]string) {
	t = derefPointerType(t)
	if t.Kind() != reflect.Struct || t.Name() == "" {
		return
	}
	if _, done := emitted[t.Name()]; done {
		return
	}
	if tsNeverEmit[t.Name()] {
		return
	}
	name := t.Name()
	emitted[name] = "" // marca antes de descer, para tolerar ciclo

	// Um `MarshalJSON` próprio desliga a reflexão sobre a struct: o que vale é a
	// forma declarada no override.
	shape := t
	if wire, ok := tsWireOverrides[name]; ok {
		shape = wire
	} else if hasCustomMarshaler(t) {
		panic(fmt.Sprintf(
			"tsgen: %s tem MarshalJSON próprio e não está em tsWireOverrides — "+
				"o tipo gerado descreveria a struct em memória, não o que vai no fio", name))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "export type %s = {\n", name)
	for i := 0; i < shape.NumField(); i++ {
		field := shape.Field(i)
		if !field.IsExported() {
			continue
		}
		name, optional, skip := tsFieldName(field)
		if skip {
			continue
		}
		qualified := t.Name() + "." + name
		// O override é consultado ANTES de traduzir o tipo: `tsTypeOf` REGISTRA
		// os structs que encontra, então perguntar depois já teria emitido a
		// declaração que o override existe para evitar.
		ts, overridden := tsFieldTypeOverrides[qualified]
		if !overridden {
			ts = tsTypeOf(field.Type, emitted)
			if key, ok := tsKeyOverrides[qualified]; ok {
				ts = narrowRecordKey(ts, key)
			}
		}
		fmt.Fprintf(&b, "  %s%s: %s\n", name, optional, ts)
	}
	b.WriteString("}\n")
	emitted[name] = b.String()
}

// narrowRecordKey troca a chave de um `Record<string, X>` pela união pedida.
func narrowRecordKey(ts, key string) string {
	const prefix = "Record<string, "
	if !strings.HasPrefix(ts, prefix) {
		return ts
	}
	return "Record<" + key + ", " + ts[len(prefix):]
}

// hasCustomMarshaler reporta se o tipo (ou seu ponteiro) implementa json.Marshaler.
func hasCustomMarshaler(t reflect.Type) bool {
	marshaler := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	return t.Implements(marshaler) || reflect.PointerTo(t).Implements(marshaler)
}

// tsFieldName lê a tag json: nome, `omitempty` (vira opcional) e `-` (some).
func tsFieldName(field reflect.StructField) (name, optional string, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", "", true
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	if name == "" {
		name = field.Name
	}
	for _, opt := range parts[1:] {
		if opt == "omitempty" {
			optional = "?"
		}
	}
	return name, optional, false
}

// tsTypeOf traduz um tipo Go para TypeScript, registrando structs alcançados.
func tsTypeOf(t reflect.Type, emitted map[string]string) string {
	switch t.Kind() {
	case reflect.Ptr:
		// Ponteiro é o "pode faltar" do Go — vira união com null, que é como o
		// `encoding/json` de fato serializa.
		return tsTypeOf(t.Elem(), emitted) + " | null"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return tsTypeOf(t.Elem(), emitted) + "[]"
	case reflect.Map:
		return fmt.Sprintf("Record<%s, %s>", tsTypeOf(t.Key(), emitted), tsTypeOf(t.Elem(), emitted))
	case reflect.Struct:
		if t.Name() == "" {
			return "unknown"
		}
		collectTSType(t, emitted)
		return t.Name()
	default:
		return "unknown"
	}
}

// derefPointerType descasca ponteiros até chegar no tipo apontado.
func derefPointerType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
