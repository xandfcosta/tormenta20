package bookui

import "testing"

// O ENDEREÇO DO LEITOR, provado onde ele é montado. Servir o PDF de verdade por
// HTTP é outra camada e outra pergunta, e continua no `api`.
//
// O que se prende aqui é a REGRA do endereço: livro não configurado não produz
// link nenhum (e não um link quebrado), e o termo entra escapado para o leitor
// destacar.

// O endereço aponta para o LEITOR da casa e não para o PDF cru com `#page=N`, e
// a troca foi medida: o visualizador do Chrome ignora `#search=` (não há destaque
// possível por URL) e transfere o arquivo inteiro — 85 MiB contra 1 MiB do
// leitor, contados na interface de loopback.
func TestTheButtonOpensTheReaderAtThePrintedPageWithTheTerm(t *testing.T) {
	book := BookAddress{Base: "/livro?v=abc", Opening: 6}
	if got := book.AtPage(289, "Lobo"); got != "/livro/ler?p=289&t=Lobo" {
		t.Errorf("o botão do Lobo aponta para %q", got)
	}
	// O termo vai ESCAPADO: "Bola de Fogo" tem espaço, e nome de verbete com
	// "&" quebraria a consulta inteira.
	if got := book.AtPage(180, "Bola de Fogo"); got != "/livro/ler?p=180&t=Bola+de+Fogo" {
		t.Errorf("o termo não foi escapado: %q", got)
	}
	// A ABERTURA não entra no endereço: quem soma é o leitor, que fala em página
	// impressa com quem lê e em página de arquivo com o pdf.js.
	if got := book.AtPage(289, ""); got != "/livro/ler?p=289" {
		t.Errorf("sem termo o endereço devia ser só a página, e foi %q", got)
	}
}

// O zero valor não produz link quebrado.
func TestWithoutAConfiguredBookThereIsNoAddress(t *testing.T) {
	if got := (BookAddress{}).AtPage(289, "Lobo"); got != "" {
		t.Errorf("sem livro o endereço devia ser vazio, e foi %q", got)
	}
	if got := (BookAddress{Base: "/livro"}).AtPage(0, "Lobo"); got != "" {
		t.Errorf("criatura sem página no livro devia ficar sem endereço, e ficou %q", got)
	}
}
