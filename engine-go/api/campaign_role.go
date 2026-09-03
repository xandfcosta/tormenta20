package api

// papelNaCampanha ficou aqui quando o resto do arquivo virou `web/ui/identity.go`
// (ALE-278).
//
// Os três companheiros dele — o matiz, o gradiente e o monograma — são função de
// `string` para `string` e não sabem do domínio, então foram para o kit. Este
// sabe: "Mestrando", "Jogando" e "Mesa de X" é regra de QUEM É O QUÊ numa
// campanha, e o kit de apresentação não pode conhecer isso. Ele é lido por um
// arquivo só, o da cena de campanhas, e vai junto quando ela sair.

// papelNaCampanha é o `roleLabel`: a POSTURA de quem olha.
//
// Uma mesa que é de OUTRA pessoa — que só um admin chega a ver listada — diz de
// quem ela é em vez da postura. O servidor entrega o papel `gm` ali, e escrever
// "Mestrando" faria parecer que a mesa é de quem está lendo (ALE-120).
func papelNaCampanha(papel string, dono *string) string {
	if dono != nil && *dono != "" {
		return "Mesa de " + *dono
	}
	if papel == "gm" {
		return "Mestrando"
	}
	return "Jogando"
}
