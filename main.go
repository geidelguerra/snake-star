package main

import (
	"embed"
	"log"
	"net/http"
	"snake-star/game"
	"snake-star/templates"
	"time"

	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

//go:embed assets/*
var assetsFS embed.FS

func main() {
	assetsFileSystem := hashfs.NewFS(assetsFS)
	assetsFileServer := hashfs.FileServer(assetsFileSystem)

	assetPathHandler := func(path string) string {
		return "/" + assetsFileSystem.HashName("assets/"+path)
	}

	port := "3223"
	router := chi.NewRouter()

	router.Handle("/assets/*", assetsFileServer)

	state := game.NewGame(10, 10)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		layout := templates.Layout(templates.LayoutProps{
			AssetPath: assetPathHandler,
		})

		layout.Render(r.Context(), w)
	})

	router.Get("/updates", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		rendererFn := func(state *game.Game) error {
			stateUpdate := templates.StateUpdate(templates.StateUpdateProps{
				State: state,
			})

			return sse.PatchElementTempl(stateUpdate)
		}

		state.Update(rendererFn, true)

		for {
			if sse.IsClosed() {
				break
			}

			state.Update(rendererFn, false)

			time.Sleep(time.Millisecond * 16)
		}
	})

	router.Post("/input", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")

		switch key {
		case "ArrowLeft":
			state.SetPlayerDir(game.DIR_LEFT)
		case "ArrowUp":
			state.SetPlayerDir(game.DIR_UP)
		case "ArrowRight":
			state.SetPlayerDir(game.DIR_RIGHT)
		case "ArrowDown":
			state.SetPlayerDir(game.DIR_DOWN)
		case "Escape":
			state.TogglePause()
		case "r":
			state.Restart()
		case "h":
			state.ToggleHelp()
		}
	})

	httpServer := http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("Listening on http://localhost:%s", port)

	httpServer.ListenAndServe()
}
