package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"passbook/crypto"
	"passbook/models"
	"path/filepath"
	"strings"
)

func MissingDirectory(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			fmt.Println("Error creating store directory:", err)
			return true
		}
	}
	return false
}

func UpdateList(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return []string{}
	}

	listItems := []string{}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue

		}
		listItems = append(listItems, file.Name())
	}
	return listItems
}

func GetFilteredList(s string, listItems []string) []string {
	var filtered []string = []string{}
	if len(s) > 2 {
		for _, item := range listItems {
			if strings.Contains(strings.ToLower(item), s) {
				filtered = append(filtered, item)
			}
		}
	} else {
		filtered = listItems
	}
	return filtered
}

func LoadFileContent(filePath string, passwordHash string) (models.FileDetails, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return models.FileDetails{}, err
	}
	decryptedData, err := crypto.Decrypt(data, passwordHash)
	if err != nil {
		return models.FileDetails{}, err
	}
	var details models.FileDetails
	if err := json.Unmarshal(decryptedData, &details); err != nil {
		return models.FileDetails{}, err
	}
	return details, nil
}

func SaveFileContent(filePath string, passwordHash string, detail models.FileDetails) (bool, error) {
	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return false, err
	}

	encryptedData, err := crypto.Encrypt(data, passwordHash)
	if err != nil {
		return false, err
	}

	err = os.WriteFile(filePath, encryptedData, 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func SaveSettings(detail models.Settings) (bool, error) {
	configFilePath := filepath.Join(os.Getenv("HOME"), ".passbook", ".settings.json")
	MissingDirectory(filepath.Join(os.Getenv("HOME"), ".passbook"))

	data, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return false, err
	}

	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func LoadSettings() (models.Settings, error) {
	passBookDir := filepath.Join(os.Getenv("HOME"), ".passbook")
	MissingDirectory(passBookDir)

	configFilePath := filepath.Join(passBookDir, ".settings.json")

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return models.Settings{}, err
	}

	var settings models.Settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return models.Settings{}, err
	}
	return settings, nil
}

func GetDefaultDirectory() string {
	passBookDir := filepath.Join(os.Getenv("HOME"), ".passbook")
	return passBookDir
}
