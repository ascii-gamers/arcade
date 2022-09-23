package arcade

import (
	"encoding/json"
	"io"
	"os"
	"path"
)

const PROFILE_FILENAME = ".asciiarcade"

type Profile struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func LoadProfile() (*Profile, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return nil, err
	}

	configPath := path.Join(homeDir, PROFILE_FILENAME)
	f, err := os.Open(configPath)

	if err != nil {
		return nil, err
	}

	defer f.Close()
	data, err := io.ReadAll(f)

	if err != nil {
		return nil, err
	}

	p := &Profile{}

	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Profile) Save() error {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return err
	}

	configPath := path.Join(homeDir, PROFILE_FILENAME)
	data, err := json.MarshalIndent(p, "", " ")

	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	return nil
}
