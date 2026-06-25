package ui

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// imageRightPad is extra pixels added to the right of text images to prevent
// glyph ink from bleeding past the advance width boundary (common in "%" etc).
const imageRightPad = 4

type FontCache struct {
	mu       sync.RWMutex
	faces    map[string]font.Face
	rawFonts map[string][]byte
	fontDir  string
}

func NewFontCache(fontDir string) *FontCache {
	fc := &FontCache{
		faces:    make(map[string]font.Face),
		rawFonts: make(map[string][]byte),
		fontDir:  fontDir,
	}
	fc.loadAll()
	return fc
}

func (fc *FontCache) loadAll() {
	if _, err := os.Stat(fc.fontDir); os.IsNotExist(err) {
		log.Printf("AVISO: Diretório de fontes '%s' não encontrado.", fc.fontDir)
		return
	}
	filepath.WalkDir(fc.fontDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lower := strings.ToLower(d.Name())
		if !strings.HasSuffix(lower, ".ttf") && !strings.HasSuffix(lower, ".otf") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Erro ao carregar fonte %s: %v", path, err)
			return nil
		}
		relPath, err := filepath.Rel(fc.fontDir, path)
		if err != nil {
			return nil
		}
		fc.rawFonts[filepath.ToSlash(relPath)] = data
		return nil
	})
	log.Printf("Fontes carregadas em memória: %d", len(fc.rawFonts))
}

func (fc *FontCache) AvailableFonts() []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	fonts := make([]string, 0, len(fc.rawFonts))
	for path := range fc.rawFonts {
		fonts = append(fonts, path)
	}
	sort.Strings(fonts)
	return fonts
}

func (fc *FontCache) GetFace(fontPath string, fontSize int) (font.Face, error) {
	if fontSize <= 0 {
		fontSize = 14
	}
	key := fmt.Sprintf("%s:%d", fontPath, fontSize)

	fc.mu.RLock()
	if face, ok := fc.faces[key]; ok {
		fc.mu.RUnlock()
		return face, nil
	}
	fc.mu.RUnlock()

	fc.mu.Lock()
	defer fc.mu.Unlock()

	if face, ok := fc.faces[key]; ok {
		return face, nil
	}

	raw, ok := fc.rawFonts[fontPath]
	if !ok {
		return nil, fmt.Errorf("fonte não encontrada: %s", fontPath)
	}

	parsed, err := opentype.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear fonte %s: %w", fontPath, err)
	}

	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(fontSize),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar face %s@%d: %w", fontPath, fontSize, err)
	}

	fc.faces[key] = face
	return face, nil
}

func (fc *FontCache) MeasureText(text string, fontPath string, fontSize int) (int, int, error) {
	face, err := fc.GetFace(fontPath, fontSize)
	if err != nil {
		return 0, 0, err
	}
	drawer := &font.Drawer{Face: face}
	advance := drawer.MeasureString(text)
	width := advance.Ceil()
	height := face.Metrics().Height.Ceil()
	return width, height, nil
}

func (fc *FontCache) TextWidth(text string, fontPath string, fontSize int) fixed.Int26_6 {
	face, err := fc.GetFace(fontPath, fontSize)
	if err != nil {
		return 0
	}
	drawer := &font.Drawer{Face: face}
	return drawer.MeasureString(text)
}