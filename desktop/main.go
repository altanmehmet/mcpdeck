package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/altanmehmet/mcpdeck/cmd"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/altanmehmet/mcpdeck/internal/client"
	"github.com/altanmehmet/mcpdeck/internal/store"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist/client
var files embed.FS

func main() {
	prepareDesktopEnvironment()
	// Agent configurations point to this executable; keep stdio bridge headless.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-psn_") {
		if err := cmd.Execute(); err != nil {
			os.Exit(1)
		}
		return
	}
	s := store.Default()
	if path := os.Getenv("MCPDECK_CONFIG"); path != "" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Invalid MCPDeck configuration path.")
			os.Exit(1)
		}
		s.Path = absolute
	}
	app := client.New(s)
	if os.Getenv("MCPDECK_CLIENT_DEMO") == "1" {
		var cleanup func()
		var demoErr error
		app, cleanup, demoErr = client.NewDemo()
		if demoErr != nil {
			fmt.Fprintln(os.Stderr, "Could not prepare the isolated client demo.")
			os.Exit(1)
		}
		defer cleanup()
	}
	assets, err := fs.Sub(files, "frontend/dist/client")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Client assets unavailable.")
		os.Exit(1)
	}
	nativeMenu := menu.NewMenu()
	nativeMenu.Append(menu.AppMenu())
	nativeMenu.Append(menu.EditMenu())
	err = wails.Run(&options.App{
		Title: "MCPDeck", Width: 1280, Height: 912, MinWidth: 850, MinHeight: 660,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets}, Bind: []interface{}{app},
		Menu: nativeMenu, EnableDefaultContextMenu: true,
		OnShutdown: func(context.Context) { app.Close() },
		OnBeforeClose: func(ctx context.Context) bool {
			if !app.NeedsCloseConfirmation() {
				return false
			}
			answer, e := wruntime.MessageDialog(ctx, wruntime.MessageDialogOptions{
				Type: wruntime.QuestionDialog, Title: "Close MCPDeck?",
				Message: "Unsaved edits will be discarded and running setup will be cancelled. Completed setup effects are kept.",
				Buttons: []string{"Keep open", "Close"}, DefaultButton: "Keep open", CancelButton: "Keep open",
			})
			return e != nil || answer != "Close"
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "MCPDeck client could not start:", err)
		os.Exit(1)
	}
}
