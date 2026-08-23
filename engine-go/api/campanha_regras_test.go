package api

import (
	"errors"
	"strings"
	"testing"
)

// Os guardas das regras da crônica (ALE-246).
//
// O que se protege aqui é o que a virada quase perdeu: o limite da DESCRIÇÃO
// morava só no `campaign-schema.ts` da SPA, e a tela nova é do servidor.

// O nome é aparado ANTES de medido, e é isso que faz um nome de puros espaços
// ser recusado em vez de virar crônica sem título no livro.
func TestNomeDeCampanhaEhAparadoAntesDeMedido(t *testing.T) {
	if _, err := nomeDeCampanha("   "); !errors.Is(err, errNomeDeCampanha) {
		t.Error("nome de puros espaços passou — a crônica nasceria sem título")
	}
	nome, err := nomeDeCampanha("  A Queda de Tauron  ")
	if err != nil {
		t.Fatalf("nome válido recusado: %v", err)
	}
	if nome != "A Queda de Tauron" {
		t.Errorf("nome = %q, queria sem os espaços das pontas", nome)
	}
}

// A medida é em RUNAS e não em bytes. Um limite que encolhe conforme os acentos
// é um limite que mente: "Coração" tem 7 caracteres para quem escreve e 8 bytes
// para quem conta errado.
func TestOsLimitesContamCARACTERESENaoBytes(t *testing.T) {
	// 120 runas acentuadas = 240 bytes. Se a conta fosse em bytes, este nome
	// legítimo seria recusado.
	nome := strings.Repeat("ç", nomeDeCampanhaMax)
	if _, err := nomeDeCampanha(nome); err != nil {
		t.Errorf("nome de %d caracteres acentuados recusado: %v", nomeDeCampanhaMax, err)
	}
	if _, err := nomeDeCampanha(nome + "ç"); !errors.Is(err, errNomeDeCampanha) {
		t.Error("nome com uma runa a mais que o teto passou")
	}

	texto := strings.Repeat("ã", descricaoDeCampanhaMax)
	if _, err := descricaoDeCampanha(&texto); err != nil {
		t.Errorf("descrição de %d caracteres acentuados recusada: %v", descricaoDeCampanhaMax, err)
	}
}

// A LACUNA QUE ESTA FATIA FECHOU: o teto de 2000 do texto existia só na SPA, e
// o servidor aceitava qualquer tamanho. Com a tela virando do servidor, a regra
// teria sumido junto com o formulário que a carregava.
func TestADescricaoTemTetoNoSERVIDOREnaoSoNaTela(t *testing.T) {
	longa := strings.Repeat("a", descricaoDeCampanhaMax+1)
	if _, err := descricaoDeCampanha(&longa); !errors.Is(err, errDescricaoDeCampanha) {
		t.Errorf("descrição de %d caracteres passou — o teto vivia só no cliente", len(longa))
	}
}

// Descrição vazia (ou de puros espaços) é NULL nos DOIS caminhos, criar e
// editar. Sem isso o cliente lê "" de um e null do outro para a mesma entrada.
func TestDescricaoVaziaEhNuloENaoStringVazia(t *testing.T) {
	for _, entrada := range []string{"", "   ", "\n\t "} {
		got, err := descricaoDeCampanha(&entrada)
		if err != nil {
			t.Fatalf("descrição %q recusada: %v", entrada, err)
		}
		if got.Valid {
			t.Errorf("descrição %q virou %q em vez de NULL", entrada, got.String)
		}
	}
	// Ausente também é NULL, e é caso diferente de vazia: "não mandei o campo".
	if got, _ := descricaoDeCampanha(nil); got.Valid {
		t.Error("descrição ausente virou valor")
	}
}
