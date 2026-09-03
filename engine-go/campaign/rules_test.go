package campaign

import (
	"strings"
	"testing"
)

// Os guardas das regras da CAMPANHA (ALE-246; o pacote é da ALE-278).
//
// O que se protege aqui é o que a virada quase perdeu: o limite da DESCRIÇÃO
// morava só no `campaign-schema.ts` da SPA, e a tela nova é do servidor.

// O nome é aparado ANTES de medido, e é isso que faz um nome de puros espaços
// ser recusado em vez de virar campanha sem título no livro.
func TestACampaignNameIsTrimmedBeforeItIsMeasured(t *testing.T) {
	if _, erros := Name("   "); len(erros) == 0 {
		t.Error("nome de puros espaços passou — a campanha nasceria sem título")
	}
	nome, erros := Name("  A Queda de Tauron  ")
	if len(erros) > 0 {
		t.Fatalf("nome válido recusado: %v", erros)
	}
	if nome != "A Queda de Tauron" {
		t.Errorf("nome = %q, queria sem os espaços das pontas", nome)
	}
}

// A medida é em RUNAS e não em bytes. Um limite que encolhe conforme os acentos
// é um limite que mente: "Coração" tem 7 caracteres para quem escreve e 8 bytes
// para quem conta errado.
func TestTheLimitsCountCharactersAndNotBytes(t *testing.T) {
	// 120 runas acentuadas = 240 bytes. Se a conta fosse em bytes, este nome
	// legítimo seria recusado.
	nome := strings.Repeat("ç", MaxNameLength)
	if _, erros := Name(nome); len(erros) > 0 {
		t.Errorf("nome de %d caracteres acentuados recusado: %v", MaxNameLength, erros)
	}
	if _, erros := Name(nome + "ç"); len(erros) == 0 {
		t.Error("nome com uma runa a mais que o teto passou")
	}

	texto := strings.Repeat("ã", MaxDescriptionLength)
	if _, erros := Description(&texto); len(erros) > 0 {
		t.Errorf("descrição de %d caracteres acentuados recusada: %v", MaxDescriptionLength, erros)
	}
}

// A LACUNA QUE ESTA FATIA FECHOU: o teto de 2000 do texto existia só na SPA, e
// o servidor aceitava qualquer tamanho. Com a tela virando do servidor, a regra
// teria sumido junto com o formulário que a carregava.
func TestTheDescriptionCeilingIsOnTheServerAndNotOnlyOnTheScreen(t *testing.T) {
	longa := strings.Repeat("a", MaxDescriptionLength+1)
	if _, erros := Description(&longa); len(erros) == 0 {
		t.Errorf("descrição de %d caracteres passou — o teto vivia só no cliente", len(longa))
	}
}

// Descrição vazia (ou de puros espaços) sai VAZIA, e quem grava a traduz para
// NULL nos DOIS caminhos, criar e editar. Sem isso o cliente lê "" de um e null
// do outro para a mesma entrada.
func TestAnEmptyDescriptionIsNullAndNotAnEmptyString(t *testing.T) {
	for _, entrada := range []string{"", "   ", "\n\t "} {
		got, erros := Description(&entrada)
		if len(erros) > 0 {
			t.Fatalf("descrição %q recusada: %v", entrada, erros)
		}
		if got != "" {
			t.Errorf("descrição %q virou %q em vez de vazia", entrada, got)
		}
	}
	// Ausente também é vazia, e é caso diferente: "não mandei o campo". Quem
	// grava é que traduz vazio para NULL — a regra devolve texto.
	if got, _ := Description(nil); got != "" {
		t.Error("descrição ausente virou valor")
	}
}
