//go:build !windows || wv2runtime.error

package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	linuxoptions "github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wr "github.com/wailsapp/wails/v2/pkg/runtime"
	"io/fs"
	"os"
	"path/filepath"
	"termiashell/internal/app"
	"termiashell/internal/platform"
)

//go:embed all:frontend
var assets embed.FS

type desktopUI struct{ ctx context.Context }

func (d *desktopUI) Emit(event string, args ...interface{}) {
	if d.ctx != nil {
		wr.EventsEmit(d.ctx, event, args...)
	}
}
func (d *desktopUI) OpenFile() (string, error) {
	return wr.OpenFileDialog(d.ctx, wr.OpenDialogOptions{Title: "Selecionar arquivo para upload"})
}
func (d *desktopUI) OpenCSV() (string, error) {
	return wr.OpenFileDialog(d.ctx, wr.OpenDialogOptions{Title: "Importar conexões SSH por CSV", Filters: []wr.FileFilter{{DisplayName: "CSV (*.csv)", Pattern: "*.csv"}}})
}
func (d *desktopUI) SaveFile(name string) (string, error) {
	return wr.SaveFileDialog(d.ctx, wr.SaveDialogOptions{Title: "Salvar download — escolha um nome livre", DefaultFilename: name})
}
func run() error {
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	dir := flag.String("data-dir", filepath.Join(filepath.Dir(exe), "data"), "Pasta portátil de dados")
	flag.Parse()
	data, e := filepath.Abs(*dir)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(data, 0700); e != nil {
		return e
	}
	unlock, e := platform.Lock(data)
	if e != nil {
		return e
	}
	defer unlock()
	ui := &desktopUI{}
	backend, e := app.New(data, ui)
	if e != nil {
		return e
	}
	defer backend.Shutdown()
	lifecycle := newLifecycle(backend.HasActiveWork, func() { ui.Emit("app:close-requested") }, func() { wr.Quit(ui.ctx) })
	front, e := fs.Sub(assets, "frontend")
	if e != nil {
		return e
	}
	iconPNG, _ := os.ReadFile(filepath.Join(filepath.Dir(exe), "termiaShell.png"))
	return wails.Run(&options.App{Title: "termiaShell", Width: 1200, Height: 760, MinWidth: 850, MinHeight: 560, StartHidden: true, BackgroundColour: options.NewRGB(11, 16, 26), AssetServer: &assetserver.Options{Assets: front}, Bind: []interface{}{backend, lifecycle}, OnStartup: func(ctx context.Context) { ui.ctx = ctx }, OnDomReady: func(ctx context.Context) {
		platform.SetAppIcon(filepath.Join(filepath.Dir(exe), "termiaShell.ico"))
		wr.WindowCenter(ctx)
		wr.WindowShow(ctx)
	}, OnBeforeClose: func(context.Context) bool { return lifecycle.beforeClose() }, Windows: &windows.Options{Theme: windows.SystemDefault, WebviewUserDataPath: filepath.Join(data, "webview"), DisablePinchZoom: true, Messages: &windows.Messages{ContactAdmin: "WebView2 existente é necessário. Nenhum componente será instalado.", Error: "termiaShell — pré-requisito ausente", Webview2NotInstalled: "WebView2 não disponível"}}, Linux: &linuxoptions.Options{Icon: iconPNG, ProgramName: "termiaShell"}})
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		exe, _ := os.Executable()
		os.WriteFile(filepath.Join(filepath.Dir(exe), "startup-error.txt"), []byte(e.Error()), 0600)
		os.Exit(1)
	}
}
