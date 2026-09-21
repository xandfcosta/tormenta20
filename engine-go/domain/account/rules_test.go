package account

import "testing"

// A REGRA DE SENHA é de PRODUTO, e o teste dela mora onde ela mora (ALE-278).
//
// Ele veio dos testes de autenticação do `serve/api` — arquivo que não existe
// mais —, onde prendia a cópia ERRADA: havia duas famílias de validação, e
// aquele caso chamava a exportada, cujas mensagens saíam em inglês e que nenhuma
// tela usava. Mudar o mínimo na cópia viva o deixava VERDE.
//
// Os números e as frases estão escritos à mão de propósito. Derivá-los das
// constantes do pacote faria o teste andar junto com o defeito — trocar `8` por
// `6` em `ValidatePassword` passaria, porque a asserção teria mudado junto.
func TestThePasswordRuleRefusesOutsideTheRange(t *testing.T) {
	cases := []struct {
		name     string
		password string
		refusal  bool
		message  string
	}{
		{"sete caracteres é curta demais", "1234567", true, "A senha precisa ter ao menos 8 caracteres"},
		{"oito é o mínimo, e passa", "12345678", false, ""},
		{"cento e vinte e oito é o máximo, e passa", string(make([]byte, 128)), false, ""},
		{"cento e vinte e nove é longa demais", string(make([]byte, 129)), true, "A senha pode ter no máximo 128 caracteres"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := ValidatePassword(tc.password)
			if !tc.refusal {
				if len(fields) != 0 {
					t.Fatalf("senha de %d caracteres foi recusada: %v", len(tc.password), fields)
				}
				return
			}
			msgs := fields["password"]
			if len(msgs) == 0 {
				t.Fatalf("senha de %d caracteres passou, e devia ser recusada", len(tc.password))
			}
			if msgs[0] != tc.message {
				t.Errorf("a recusa diz %q, e o jogador devia ler %q", msgs[0], tc.message)
			}
		})
	}
}

// O LOGIN NÃO CONFERE A FAIXA, e isso não é esquecimento.
//
// Quem já tem uma senha longa gravada precisa conseguir entrar com ela. Recusar
// no login o que o registro aceitou tranca a conta em vez de proteger, e o
// sintoma seria "minha senha parou de funcionar" sem nada no log.
func TestTheLoginOnlyRequiresThatThePasswordExists(t *testing.T) {
	if fields := ValidateLogin(LoginBody{Email: "mestre@t20.local", Password: "1234567"}); len(fields) != 0 {
		t.Fatalf("o login recusou uma senha curta que já existe na conta: %v", fields)
	}
	inputs := ValidateLogin(LoginBody{Email: "mestre@t20.local", Password: ""})
	if len(inputs["password"]) == 0 || inputs["password"][0] != "Informe sua senha" {
		t.Errorf("senha vazia devia pedir 'Informe sua senha', e deu %v", inputs)
	}
}

func TestTheEmailShapeRefusesWhatCannotReceiveAnInvite(t *testing.T) {
	for _, good := range []string{"mestre@t20.local", "a.b+c@sub.dominio.com.br"} {
		if !IsEmail(good) {
			t.Errorf("%q é um e-mail e foi recusado", good)
		}
	}
	for _, bad := range []string{"bad", "sem@ponto", "com espaco@t20.local", "@t20.local"} {
		if IsEmail(bad) {
			t.Errorf("%q não é um e-mail e passou", bad)
		}
	}
}
