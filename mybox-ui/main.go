package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:dist
var assets embed.FS

func main() {
	app := NewApp()

	// Sanal dosya sisteminin kök dizinini 'dist' klasörünün içi olarak ayarlıyoruz
	assetsRoot, err := fs.Sub(assets, "dist")
	if err != nil {
		log.Fatal("Gömülü dosyalar ayarlanamadı:", err)
	}

	err = wails.Run(&options.App{
		Title:  "MyBox Manager",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assetsRoot, // Artık doğrudan dist klasörünün içeriğini görüyor
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Hata:", err.Error())
	}
}