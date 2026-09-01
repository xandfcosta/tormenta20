package ui

import (
	"bytes"
	"context"
	"fmt"

	"github.com/a-h/templ"
)

// RenderFragment devolve o HTML de um componente, para viajar num
// `datastar-patch-elements`.
//
// Ela morava no `api` e veio para cá na ALE-278: renderizar componente é
// apresentação, e toda cena que sair vai precisar dela.
func RenderFragment(ctx context.Context, c templ.Component) (string, error) {
	var buf bytes.Buffer
	if err := c.Render(ctx, &buf); err != nil {
		return "", fmt.Errorf("render de fragmento: %w", err)
	}
	return buf.String(), nil
}
