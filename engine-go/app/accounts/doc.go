// Package accounts são os CASOS DE USO de uma conta: entrar, cadastrar-se
// gastando um convite, e trocar a senha por um link de uso único.
//
// # Por que `accounts` e não `account`
//
// O plural é o padrão da casa para o pacote de aplicação não colidir com o do
// domínio — foi assim que a ALE-344 escolheu `app/boards` ao lado de
// `domain/board`. Aqui ele funciona, e nas campanhas não funcionou: lá o plural
// colidia com a CENA (`serve/web/campaigns`), e a ALE-348 teve de ir no
// singular contra o padrão.
//
// Contado antes de escolher: `app/account` no singular colidiria com
// `domain/account` em CINCO arquivos; o plural não colide com nada, porque as
// cenas desta família se chamam `door` e `admin`.
//
// # O que é daqui e o que é do `domain/account`
//
// O `domain/account` é a REGRA pura — o que é um e-mail, o que é uma senha
// aceitável, a forma dos dois pedidos. Aqui mora a ORQUESTRAÇÃO: resolver o
// convite, gerar o hash, escrever a linha, assinar a sessão.
//
// # O `bcrypt` mora aqui, e isso é fronteira
//
// O custo do bcrypt é decisão de SEGURANÇA do servidor, não de quem desenha o
// formulário — a porta da cena já dizia isso por escrito antes de haver camada.
// Gerar o hash na tela obrigaria a cena a carregar uma constante criptográfica
// para fazer trabalho que não é dela.
package accounts
