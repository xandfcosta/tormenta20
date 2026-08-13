package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// O arquivo gerado é commitado, então precisa de um espelho — senão alguém muda
// uma struct da fronteira, esquece de regenerar, e o TS descreve uma forma que o
// motor não produz mais. O typechecker não pega isso: ele confere o código
// contra o TIPO, não o tipo contra o servidor.
//
// Mesmo contrato dos oráculos de paridade: falha mandando regenerar.
func TestGeneratedTypesAreCurrent(t *testing.T) {
	path := filepath.Clean(filepath.Join(mustWd(t), "..", "..",
		"frontend", "src", "shared", "api", "engine-types.ts"))
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler os tipos gerados: %v (rode `go run ./cmd/tsgen`)", err)
	}
	if got := GenerateTypeScript(); got != string(onDisk) {
		t.Errorf("engine-types.ts está desatualizado — rode `cd engine-go && go run ./cmd/tsgen`\n%s",
			firstDiffLine(got, string(onDisk)))
	}
}

// firstDiffLine aponta a primeira linha divergente, que é o que diz QUAL struct
// mudou — um diff inteiro de 200 linhas não ajuda ninguém.
func firstDiffLine(got, want string) string {
	g, w := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return "primeira diferença na linha " + itoa(i+1) + ":\n  gerado: " + g[i] + "\n  arquivo: " + w[i]
		}
	}
	return "o arquivo tem " + itoa(len(w)) + " linhas e o gerado tem " + itoa(len(g))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// A armadilha que motivou o `tsWireOverrides`: um `MarshalJSON` próprio troca a
// forma que vai no fio, e refletir a struct em memória produz um tipo que MENTE.
// O ItemEffects guarda as flags num Set e serializa um array ordenado.
func TestWireOverrideDescribesWhatGoesOnTheWire(t *testing.T) {
	ts := GenerateTypeScript()

	if !strings.Contains(ts, "flags: string[]") {
		t.Error("ItemEffects deveria declarar `flags: string[]` — é o que o MarshalJSON emite")
	}
	if strings.Contains(ts, "Flags: Record<string, boolean>") {
		t.Error("ItemEffects saiu com a forma EM MEMÓRIA — o override não foi aplicado")
	}

	// E todo tipo com marshaller próprio tem de estar declarado, senão o emissor
	// volta a descrever a struct por acidente.
	for _, root := range tsRootTypes() {
		if hasCustomMarshaler(root) {
			if _, ok := tsWireOverrides[root.Name()]; !ok {
				t.Errorf("%s tem MarshalJSON e não está em tsWireOverrides", root.Name())
			}
		}
	}
}

// O override tem de continuar batendo com o que o MarshalJSON realmente monta:
// alguém pode acrescentar um campo lá e esquecer da struct de fio.
func TestItemEffectsWireMatchesItsMarshalOutput(t *testing.T) {
	raw, err := ItemEffects{Flags: map[string]bool{"armadura-pesada": true}}.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wireFields := map[string]bool{}
	wire := reflect.TypeOf(ItemEffectsWire{})
	for i := 0; i < wire.NumField(); i++ {
		name, _, _ := tsFieldName(wire.Field(i))
		wireFields[name] = true
	}
	for _, key := range []string{"byTarget", "flags", "conditional"} {
		if !wireFields[key] {
			t.Errorf("ItemEffectsWire não declara %q", key)
		}
		if !strings.Contains(string(raw), `"`+key+`"`) {
			t.Errorf("o JSON emitido não traz %q", key)
		}
	}
}
