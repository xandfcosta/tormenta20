package api

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Redefinição de senha (ALE-120). O que precisa ser verdade: o link troca a
// senha DE VERDADE (a antiga para de valer), serve uma vez só, e nada disso
// depende de estar logado — quem esqueceu a senha não consegue autenticar.

// resetLinkFor cunha o link pela REGRA, e não pela rota do admin.
//
// As duas rotas que estes casos dirigiam — `POST /admin/users/{id}/password-reset`
// e `POST /auth/reset-password` — saíram na ALE-277. O que eles prendem nunca foi
// o transporte: é o link valer UMA vez, a corrida de dois pedidos gastá-lo uma
// vez só, e a senha fraca ser recusada no SERVIDOR. A porta em Datastar troca a
// senha pelo mesmo `ResetPassword`, e a administração cunha pelo mesmo
// `mintPasswordReset`.
func resetLinkFor(t *testing.T, s *Server, adminID, UserID int64) string {
	t.Helper()
	reset, err := s.adminHost().mintPasswordReset(context.Background(), UserID, adminID)
	if err != nil {
		t.Fatalf("gerar link: %v", err)
	}
	return reset.Token
}

// trocaASenha é o gesto que a PORTA faz: um token e uma senha nova.
func trocaASenha(t *testing.T, s *Server, token, senha string) bool {
	t.Helper()
	return s.ResetPassword(context.Background(), token, senha)
}

func passwordOf(t *testing.T, s *Server, UserID int64) string {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), UserID)
	if err != nil {
		t.Fatalf("carregar usuário: %v", err)
	}
	return user.Passwordhash
}

func TestTheResetLinkActuallyChangesThePassword(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)
	before := passwordOf(t, s, player)

	if !trocaASenha(t, s, token, "nova-senha-do-jogador") {
		t.Fatal("o link não trocou a senha")
	}
	after := passwordOf(t, s, player)
	if after == before {
		t.Fatal("o hash não mudou — o link não trocou senha nenhuma")
	}
	if bcrypt.CompareHashAndPassword([]byte(after), []byte("nova-senha-do-jogador")) != nil {
		t.Error("a senha nova não abre a conta")
	}
}

// Um link gasto que continuasse valendo seria uma senha que qualquer um com o
// histórico da conversa pode trocar.
func TestTheResetLinkWorksOnlyOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)
	if !trocaASenha(t, s, token, "outra-senha-longa") {
		t.Fatal("o primeiro uso do link falhou")
	}
	if trocaASenha(t, s, token, "outra-senha-longa") {
		t.Error("o link gasto trocou a senha de novo")
	}
}

// O uso sequencial acima é barrado na LEITURA, então ele não prova o
// `usedAt IS NULL` do UPDATE. O que a cláusula protege é a corrida: todos
// passam pela leitura, e só um UPDATE encontra o link por gastar.
func TestConcurrentResetsSpendTheLinkOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)

	const racers = 4
	codes := make(chan int, racers)
	start := make(chan struct{})
	for range racers {
		go func() {
			<-start
			if trocaASenha(t, s, token, "senha-da-corrida") {
				codes <- 1
			} else {
				codes <- 0
			}
		}()
	}
	close(start)

	won := 0
	for range racers {
		if <-codes == 1 {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d redefinições aceitas com UM link, esperado exatamente 1", won)
	}
}

// A tela pergunta antes de mostrar o formulário: um link vencido diz isso em
// vez de falhar no envio com a senha já digitada duas vezes.
func TestResolvingAResetLinkNamesTheAccount(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)

	// A PERGUNTA da porta, e não a rota JSON: `ResetLinkOwner` é o que a cena
	// chama para escrever "você está trocando a senha de fulano" antes do
	// formulário.
	email, achou := s.ResetLinkOwner(context.Background(), token)
	if !achou || email != "jogador@t20.local" {
		t.Errorf("dono do link = %q (achou=%v), esperado a conta do link", email, achou)
	}
	if _, achou := s.ResetLinkOwner(context.Background(), "nao-existe"); achou {
		t.Error("um link inventado devolveu dono")
	}
}

// Aqui morava o TestAResetRefusesAWeakPassword. A rota `POST /auth/reset-password`
// saiu na ALE-277, e a garantia está em DUAS camadas que não morreram: a regra é
// do `account` (`ValidatePassword`, com a frase em pt-BR), e quem a chama antes
// de trocar é a PORTA — `TestTheDoorSaysValidationRefusalsInPortuguese` e
// `TestTheDoorRefusesPasswordsThatDoNotMatchOnTheServer`.
//
// Vale dizer o que isso significa para o `ResetPassword` da porta: ele NÃO valida
// força de senha, e nunca validou — quem valida é quem chama. Um segundo
// chamador que esquecesse disso trocaria a senha por "123", e é por isso que a
// linha fica escrita aqui em vez de sumir com o teste.
