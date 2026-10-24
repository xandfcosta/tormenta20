package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

// A NOTA DO QUE NÃO FOI MODELADO NÃO PODE ESTAR PELA METADE (ALE-440).
//
// Uma delas estava: a do mitral terminava em "…ramifica pelo peso e não", e a
// frase cortada passou por duas fatias sendo lida como registro cuidadoso.
//
// # O que ESTE guarda alcança, e o que não alcança
//
// Ele não julga se o motivo é VERDADE — "o motor não tem redução por tipo" era
// verdade quando foi escrito e virou mentira três fatias depois, e nenhuma
// varredura de texto responde isso. O que ele julga é FORMA: uma nota que acaba
// numa conjunção é uma frase que alguém não terminou de escrever.
//
// A outra metade — a nota que ENVELHECEU — se confere no olho, e o lugar onde
// ela aparece é o relatório do auditor, que imprime todas elas a cada corrida.
func TestNoUnmodeledNoteEndsMidSentence(t *testing.T) {
	// CONECTIVOS, e não uma lista de pontuação: a nota termina sem ponto final
	// de propósito (ela é um motivo, não uma frase de tela), então o que
	// denuncia o corte é a última PALAVRA pedir uma continuação.
	cortada := map[string]bool{
		"e": true, "ou": true, "mas": true, "não": true, "nem": true, "que": true,
		"porque": true, "pois": true, "com": true, "sem": true, "de": true, "da": true,
		"do": true, "em": true, "na": true, "no": true, "para": true, "por": true,
		"a": true, "o": true, "as": true, "os": true,
	}

	medidas := 0
	for _, entrada := range todasAsNotas(t) {
		medidas++
		palavras := strings.Fields(entrada.nota)
		if len(palavras) == 0 {
			t.Errorf("%s: a nota do que não foi modelado está vazia", entrada.onde)
			continue
		}
		ultima := strings.ToLower(strings.Trim(palavras[len(palavras)-1], ".,;:—-"))
		if cortada[ultima] {
			t.Errorf("%s: a nota termina em %q, que pede continuação — a frase ficou "+
				"pela metade.\n    %s", entrada.onde, ultima, entrada.nota)
		}
	}
	// O DENOMINADOR: sem ele, uma varredura que parasse de achar notas passaria
	// verde dizendo que nenhuma está cortada.
	if medidas < 30 {
		t.Fatalf("o guarda viu só %d notas de `unmodeled`: ele parou de achar o que "+
			"veio medir", medidas)
	}
	t.Logf("notas medidas: %d", medidas)
}

type notaDeVerbete struct{ onde, nota string }

// todasAsNotas colhe o `unmodeled` de todo item, nas DUAS formas que ele tem:
// texto no encanto e mapa por metade no material.
func todasAsNotas(t *testing.T) []notaDeVerbete {
	t.Helper()
	raw, err := resourceJSON("items")
	if err != nil {
		t.Fatalf("ler items.json: %v", err)
	}
	var itens []struct {
		Name      string          `json:"name"`
		Unmodeled json.RawMessage `json:"unmodeled"`
	}
	if err := json.Unmarshal(raw, &itens); err != nil {
		t.Fatalf("items.json: %v", err)
	}
	notas := []notaDeVerbete{}
	for _, item := range itens {
		if len(item.Unmodeled) == 0 {
			continue
		}
		var texto string
		if json.Unmarshal(item.Unmodeled, &texto) == nil {
			notas = append(notas, notaDeVerbete{onde: item.Name, nota: texto})
			continue
		}
		var metades map[string]string
		if err := json.Unmarshal(item.Unmodeled, &metades); err != nil {
			t.Errorf("%s: o `unmodeled` não é texto nem mapa por metade: %v", item.Name, err)
			continue
		}
		for metade, motivo := range metades {
			notas = append(notas, notaDeVerbete{onde: item.Name + "/" + metade, nota: motivo})
		}
	}
	return notas
}
