package door

import (
	"t20engine/plataforma"
)

// A PORTA como dado (ALE-229): entrar, criar conta e redefinir senha.
//
// Cada tela tem a própria struct, e não uma só com campos opcionais, porque um
// `Convite` preenchido na tela de entrar — ou um `Token` de redefinição na de
// criar conta — é um estado que não deveria ser representável.
//
// Todas carregam de volta o que o jogador digitou, MENOS a senha. Devolver uma
// senha ao navegador é mandar de volta o que ele mandou, e depois de uma recusa
// redigitá-la é o comportamento que qualquer pessoa espera.

// signInView é a tela de login.
type signInView struct {
	Email string
	// Destino é o `?redirect=` que o guarda de rota da SPA carregava: quem foi
	// mandado para a porta volta para onde estava. Ele viaja em campo OCULTO e
	// não na URL do POST para o formulário ser o dono do próprio contexto.
	Destination string
	Errors      plataforma.FieldErrorMap
	// Aviso é a recusa do formulário inteiro, quando nenhum campo é o dono do
	// problema — "E-mail ou senha incorretos" não é culpa de um dos dois.
	Notice string
}

// signUpView é o registro, que só existe com um convite na mão.
type signUpView struct {
	Email  string
	Name   string
	Invite string
	Errors plataforma.FieldErrorMap
	Notice string
}

// resetView é a outra ponta do link que o admin gera (ALE-120).
type resetView struct {
	Token string
	// AccountEmail é a ÚNICA coisa que esta rota anônima revela, e é o que diz
	// ao jogador que ele está redefinindo a conta certa. Vazio quando o link não
	// vale — e aí não há formulário para mostrar.
	AccountEmail string
	LinkIsValid  bool
	Errors       plataforma.FieldErrorMap
	Notice       string
}

// As frases da porta. Ficam juntas porque são a VOZ da tela — o servidor
// responde a mesma recusa para casos diferentes de propósito, e a redação é
// parte dessa decisão.
const (
	// A mesma resposta para "não existe conta" e "senha errada": distinguir os
	// dois entrega a quem sonda a lista de quem tem conta aqui.
	noticeBadCredentials = "E-mail ou senha incorretos."
	// O servidor recusa convite inexistente, gasto e vencido com o mesmo 403,
	// pelo mesmo motivo. O jogador lê isto em vez do inglês da API.
	noticeBadInvite  = "Convite inválido ou expirado. Peça um link novo a quem administra a mesa."
	noticeDeadLink   = "Este link não vale mais — ele serve uma vez só e expira em 24 horas. Peça outro a quem administra a mesa."
	noticeEmailTaken = "Já existe uma conta com este e-mail."
	// A confirmação de senha é do FORMULÁRIO e não da API: o `confirmar` não
	// existe no corpo JSON, existe para pegar o typo antes de uma senha que o
	// jogador não consegue reproduzir virar a senha da conta.
	noticePasswordMismatch = "As senhas não conferem"
)
