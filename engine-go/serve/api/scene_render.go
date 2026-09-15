package api

import (
	"embed"
)

// Os ESTÁTICOS das cenas em Datastar (ALE-219). Embutidos, como os catálogos: o
// binário continua sendo UM arquivo, que é a premissa de produção deste
// projeto.
//
// Só os estáticos: desde a ALE-227 os templates são código Go gerado pelo
// `templ`, então não há mais `.html` para embutir — e some junto a classe de
// erro que o `template.Must` existia para pegar cedo. Template com sintaxe
// quebrada agora não COMPILA, que é mais cedo que o boot.
//
//go:embed assets/static/*
var assetsFS embed.FS
