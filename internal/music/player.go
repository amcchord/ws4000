package music

import (
	"io/fs"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/austinmcchord/ws4000/internal/assets"
	"github.com/veandco/go-sdl2/mix"
)

type Player struct {
	tracks  []string
	current int
	volume  int
	playing bool
}

func NewPlayer(volume float64, musicDir string) (*Player, error) {
	if err := mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 2, 4096); err != nil {
		return nil, err
	}
	p := &Player{volume: int(volume * 128)}
	if p.volume < 0 {
		p.volume = 0
	}
	if p.volume > 128 {
		p.volume = 128
	}
	mix.VolumeMusic(p.volume)

	if musicDir != "" {
		_ = filepath.WalkDir(musicDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(path), ".mp3") {
				p.tracks = append(p.tracks, path)
			}
			return nil
		})
	} else {
		root, err := assets.Root()
		if err == nil {
			musicRoot := filepath.Join(root, "music")
			_ = filepath.WalkDir(musicRoot, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if strings.HasSuffix(strings.ToLower(path), ".mp3") {
					p.tracks = append(p.tracks, path)
				}
				return nil
			})
		}
	}

	if len(p.tracks) > 0 {
		rand.Shuffle(len(p.tracks), func(i, j int) { p.tracks[i], p.tracks[j] = p.tracks[j], p.tracks[i] })
	}
	return p, nil
}

func (p *Player) Play() error {
	if len(p.tracks) == 0 || p.volume == 0 {
		return nil
	}
	path := p.tracks[p.current%len(p.tracks)]
	music, err := mix.LoadMUS(path)
	if err != nil {
		return err
	}
	if err := music.Play(-1); err != nil {
		return err
	}
	p.playing = true
	return nil
}

func (p *Player) Stop() {
	mix.HaltMusic()
	p.playing = false
}

func (p *Player) TogglePlay() {
	if p.playing {
		p.Stop()
		return
	}
	_ = p.Play()
}

func (p *Player) CurrentTrack() string {
	if len(p.tracks) == 0 {
		return "Not playing"
	}
	return filepath.Base(p.tracks[p.current%len(p.tracks)])
}

func (p *Player) Close() {
	p.Stop()
	mix.CloseAudio()
}
