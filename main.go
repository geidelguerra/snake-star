package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"snake-star/core"
	"snake-star/templates"
	"time"

	"github.com/benbjohnson/hashfs"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
)

//go:embed assets/*
var assetsFS embed.FS

func main() {
	engine := core.NewEngine()

	assetsFileSystem := hashfs.NewFS(assetsFS)
	assetsFileServer := hashfs.FileServer(assetsFileSystem)

	assetPathHandler := func(path string) string {
		return "/" + assetsFileSystem.HashName("assets/"+path)
	}

	port := "3223"
	router := chi.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("snake-star-user-id")

			if err != nil || cookie.Value == "" {
				http.SetCookie(w, &http.Cookie{
					Name:     "snake-star-user-id",
					Value:    uuid.New().String(),
					Secure:   true,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			next.ServeHTTP(w, r)
		})
	})

	router.Handle("/assets/*", assetsFileServer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		layout := templates.Layout(templates.LayoutProps{
			AssetPath: assetPathHandler,
		})

		layout.Render(r.Context(), w)
	})

	router.Get("/updates", func(w http.ResponseWriter, r *http.Request) {
		sse := datastar.NewSSE(w, r)

		cookie, err := r.Cookie("snake-star-user-id")

		if err != nil || cookie.Value == "" {
			http.Error(w, "No user ID cookie", http.StatusBadRequest)
			return
		}

		userID := cookie.Value

		rendererFn := func(state *core.Game) error {
			stateUpdate := templates.StateUpdate(templates.StateUpdateProps{
				State: state,
			})

			return sse.PatchElementTempl(stateUpdate)
		}

		game := engine.GetGame(userID)

		if game == nil {
			game, err = engine.CreateGame(userID)

			if err != nil {
				if err == core.MaxGamesReachedError {
					sse.PatchElementTempl(templates.MaxGamesReached())
					return
				}

				http.Error(w, "Failed to create game", http.StatusInternalServerError)
				return
			}
		}

		firstFrame := true

		for {
			if sse.IsClosed() {
				break
			}

			game := engine.GetGame(userID)

			if game == nil {
				sse.PatchElementTempl(templates.GameTimeout())
				break
			}

			game.Update(rendererFn, firstFrame)
			firstFrame = false

			time.Sleep(time.Millisecond * 16)
		}
	})

	router.Post("/input", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("snake-star-user-id")

		if err != nil || cookie.Value == "" {
			http.Error(w, "No user ID cookie", http.StatusBadRequest)
			return
		}

		userID := cookie.Value

		game := engine.GetGame(userID)

		if game == nil {
			http.Error(w, "No game found for user", http.StatusBadRequest)
			return
		}

		key := r.URL.Query().Get("key")

		switch key {
		case "ArrowLeft":
			game.SetPlayerDir(core.DIR_LEFT)
		case "ArrowUp":
			game.SetPlayerDir(core.DIR_UP)
		case "ArrowRight":
			game.SetPlayerDir(core.DIR_RIGHT)
		case "ArrowDown":
			game.SetPlayerDir(core.DIR_DOWN)
		case "Escape":
			game.TogglePause()
		case "r":
			game.Restart()
		case "h":
			game.ToggleHelp()
		}
	})

	httpServer := http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Listening on http://localhost:%s", port)

		for {
			engine.CleanupInactiveGames()

			var memStats runtime.MemStats
			numGoroutines := runtime.NumGoroutine()
			numOfGames := engine.GetNumOfGames()

			runtime.ReadMemStats(&memStats)

			fmt.Print("\033[H\033[2J")
			fmt.Printf("Listening on http://localhost:%s\n", port)
			fmt.Printf("Games: %d, Goroutines: %d, Alloc Memory: %dKB\n", numOfGames, numGoroutines, memStats.Alloc/1024)
			time.Sleep(time.Second * 1)
		}
	}()

	httpServer.ListenAndServe()
}
