package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const FAVORITES_PATH = "/mnt/SDCARD/Media/Music/.miyoopod_favorites.json"

type favoritesFile struct {
	Paths []string `json:"paths"`
}

func (app *MiyooPod) loadFavorites() {
	data, err := os.ReadFile(FAVORITES_PATH)
	if err != nil {
		app.Favorites = []string{}
		return
	}
	var f favoritesFile
	if err := json.Unmarshal(data, &f); err != nil {
		app.Favorites = []string{}
		return
	}
	app.Favorites = f.Paths
	logMsg(fmt.Sprintf("INFO: Loaded %d favorites", len(app.Favorites)))
}

func (app *MiyooPod) saveFavorites() {
	f := favoritesFile{Paths: app.Favorites}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(FAVORITES_PATH, data, 0644)
}

func (app *MiyooPod) isTrackFavorite(track *Track) bool {
	for _, p := range app.Favorites {
		if p == track.Path {
			return true
		}
	}
	return false
}

func (app *MiyooPod) toggleFavorite() {
	if app.Playing == nil || app.Playing.Track == nil {
		return
	}
	track := app.Playing.Track
	for i, p := range app.Favorites {
		if p == track.Path {
			app.Favorites = append(app.Favorites[:i], app.Favorites[i+1:]...)
			go app.saveFavorites()
			app.invalidateFavoritesMenu()
			app.showFavToast(false)
			return
		}
	}
	app.Favorites = append(app.Favorites, track.Path)
	go app.saveFavorites()
	app.invalidateFavoritesMenu()
	app.showFavToast(true)
}

func (app *MiyooPod) invalidateFavoritesMenu() {
	if app.FavoritesMenu != nil {
		app.FavoritesMenu.Built = false
	}
}

func (app *MiyooPod) showFavToast(added bool) {
	if app.FavToastTimer != nil {
		app.FavToastTimer.Stop()
	}
	app.FavToastAdded = added
	app.FavToastVisible = true
	app.FavToastTimer = time.AfterFunc(2*time.Second, func() {
		app.FavToastVisible = false
		app.requestRedraw()
	})
}

func (app *MiyooPod) buildFavoritesMenuItems() []*MenuItem {
	var tracks []*Track
	for _, path := range app.Favorites {
		if t, ok := app.Library.TracksByPath[path]; ok {
			tracks = append(tracks, t)
		}
	}
	if len(tracks) == 0 {
		return []*MenuItem{}
	}
	return app.buildTrackMenuItems(tracks)
}
