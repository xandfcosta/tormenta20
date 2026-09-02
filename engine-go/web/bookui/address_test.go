package bookui

import "testing"

// O ENDEREÇO DO LEITOR, provado onde ele é montado (ALE-278).
//
// Estes dois casos vieram do `api/piloto_livro_test.go` junto com o tipo. O
// resto de lá continua no `api`, porque serve o PDF de verdade por HTTP — outra
// camada, outra pergunta.
//
// O que se prende aqui é a REGRA do endereço: livro não configurado não produz
// link nenhum (e não um link quebrado), e o termo entra escapado para o leitor
// destacar.

// TestTheButtonOpensTheReaderAtThePrintedPageWithTheTerm.
//
// O endereço mudou na segunda fatia desta issue: ele apontava para o PDF cru com
// `#page=N`, e passou a apontar para o LEITOR da casa. A troca é medida — o
// visualizador do Chrome ignora `#search=` (não há destaque possível por URL) e
// transfere o arquivo inteiro; o leitor destaca o termo e custou 1 MiB contra
// 85 MiB, contados na interface de loopback.
func TestTheButtonOpensTheReaderAtThePrintedPageWithTheTerm(t *testing.T) {
	livro := BookAddress{Base: "/livro?v=abc", Abertura: 6}
	if got := livro.AtPage(289, "Lobo"); got != "/livro/ler?p=289&t=Lobo" {
		t.Errorf("o botão do Lobo aponta para %q", got)
	}
	// O termo vai ESCAPADO: "Bola de Fogo" tem espaço, e nome de verbete com
	// "&" quebraria a consulta inteira.
	if got := livro.AtPage(180, "Bola de Fogo"); got != "/livro/ler?p=180&t=Bola+de+Fogo" {
		t.Errorf("o termo não foi escapado: %q", got)
	}
	// A ABERTURA não entra no endereço: quem soma é o leitor, que fala em página
	// impressa com quem lê e em página de arquivo com o pdf.js.
	if got := livro.AtPage(289, ""); got != "/livro/ler?p=289" {
		t.Errorf("sem termo o endereço devia ser só a página, e foi %q", got)
	}
}

// TestWithoutAConfiguredBookThereIsNoAddress: o zero valor não produz link quebrado.
func TestWithoutAConfiguredBookThereIsNoAddress(t *testing.T) {
	if got := (BookAddress{}).AtPage(289, "Lobo"); got != "" {
		t.Errorf("sem livro o endereço devia ser vazio, e foi %q", got)
	}
	if got := (BookAddress{Base: "/livro"}).AtPage(0, "Lobo"); got != "" {
		t.Errorf("criatura sem página no livro devia ficar sem endereço, e ficou %q", got)
	}
}
