package main

import (
	"github.com/alexwbaule/turing-screen/internal/ui"
)

func main() {
	// Cria uma nova instância da nossa aplicação de UI
	editorApp := ui.NewEditorApp()

	// Inicia e executa a aplicação
	editorApp.Run()
}
