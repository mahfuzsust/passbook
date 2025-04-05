package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"passbook/crypto"
	"passbook/models"
	"strings"
)

var listItems []string
var listItemsPulled bool

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
	if listItemsPulled {
		return listItems
	}
	listItemsPulled = true
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
	var filtered []string = listItems
	if len(s) > 2 {
		for _, item := range listItems {
			if strings.Contains(strings.ToLower(item), s) {
				filtered = append(filtered, item)
			}
		}
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
