package main

import (
	"app.lazygit/internal/client"
	"app.lazygit/internal/session_manager"
)

func main() {
	manager := session_manager.InitSessionManager()
	defer manager.CurrentSession().Cleanup()

	client.StartApp()
}
