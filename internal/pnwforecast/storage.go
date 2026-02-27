package pnwforecast

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type AreaStorage struct {
	Path string
}

func NewAreaStorage(path string) (*AreaStorage, error) {
	s := &AreaStorage{Path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		initial := AreasFile{Version: 1, Areas: []Area{}}
		if err := s.save(initial); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *AreaStorage) load() (AreasFile, error) {
	var data AreasFile
	raw, err := os.ReadFile(s.Path)
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, err
	}
	if data.Version == 0 {
		data.Version = 1
	}
	if data.Areas == nil {
		data.Areas = []Area{}
	}
	return data, nil
}

func (s *AreaStorage) save(v AreasFile) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path, raw, 0o644)
}

func (s *AreaStorage) ListAreas() ([]Area, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	return data.Areas, nil
}

func (s *AreaStorage) GetArea(name string) (*Area, error) {
	areas, err := s.ListAreas()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(name)
	for _, a := range areas {
		if strings.ToLower(a.Name) == needle {
			copy := a
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *AreaStorage) SaveArea(area Area) error {
	data, err := s.load()
	if err != nil {
		return err
	}
	needle := strings.ToLower(area.Name)
	updated := false
	for i := range data.Areas {
		if strings.ToLower(data.Areas[i].Name) == needle {
			data.Areas[i] = area
			updated = true
			break
		}
	}
	if !updated {
		data.Areas = append(data.Areas, area)
	}
	return s.save(data)
}

func (s *AreaStorage) DeleteArea(name string) (bool, error) {
	data, err := s.load()
	if err != nil {
		return false, err
	}
	needle := strings.ToLower(name)
	out := make([]Area, 0, len(data.Areas))
	deleted := false
	for _, a := range data.Areas {
		if strings.ToLower(a.Name) == needle {
			deleted = true
			continue
		}
		out = append(out, a)
	}
	if !deleted {
		return false, nil
	}
	data.Areas = out
	if err := s.save(data); err != nil {
		return false, err
	}
	return true, nil
}
