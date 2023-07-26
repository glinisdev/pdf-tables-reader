package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var splitNumber int

func main() {
    jarPath := "tabula.jar"
    splitNumber = 2

    filePasswords := map[string]string{
        "1686874240844-inbound829254281471377347 772950.pdf": "772950",
	}

    inputFolderPath := "out"
	outputFolderPath := "out-csv"
    
    for inputFile, password := range filePasswords {        
        decrypt(inputFile, password)
        split("decrypted.pdf", splitNumber)
           
        err := processPDFs(jarPath, inputFolderPath, outputFolderPath)
        if err != nil {
            log.Fatal(err)
        }
        
        deleteFolderContents("out")
        deleteFile("decrypted.pdf")
    }
}

func processPDFs(jarPath, inputFolderPath, outputFolderPath string) error {
	var wg sync.WaitGroup

	pdfFiles, err := getPDFFiles(inputFolderPath)
	if err != nil {
		return err
	}
    
    wg.Add(splitNumber)

	for _, file := range pdfFiles {
		
		go func(file string) {
			defer wg.Done()

			pdfPath := filepath.Join(inputFolderPath, file)
			outputPath := filepath.Join(outputFolderPath, file[:len(file)-4]+".csv")

			cmd := exec.Command("java", "-Xmx2g", "-jar", jarPath, pdfPath, "-o", outputPath, "--pages", "all", "--lattice")
			err := cmd.Run()
			
            if err != nil {
				log.Printf("Tabula Java command failed for file %s: %v", file, err)
				return 
			}
		}(file)
	}

	wg.Wait()

	return nil
}

func decrypt(filePath string, password string) error {
    _, err := pdfcpu.ReadContextFile(filePath)
 
    if err != nil {
        conf := model.NewAESConfiguration(password, password, 256)
        err = pdfcpu.DecryptFile(filePath, "decrypted.pdf", conf)
        if err != nil {
            return fmt.Errorf("failed to decrypt file: %w", err)
        }
        return fmt.Errorf("failed to read file: %w", err)
    } else {
        newPath := filepath.Join(filepath.Dir(filePath), "decrypted.pdf")
        err = os.Rename(filePath, newPath)
        if err != nil {
            return fmt.Errorf("failed to rename file: %w", err)
        }
    }
     
    return nil
}

func split(filepath string, splitNumber int) error {
    pages, err := pdfcpu.PageCountFile(filepath)

    if err != nil {
        return err
    }

    divider := calculateParts(pages, splitNumber)

    pdfcpu.SplitFile(filepath, "out", divider, nil)

    return nil
}

func calculateParts(pages int, splitNumber int) int {
	remainder := pages % splitNumber
    divider := (pages - remainder) / splitNumber  + 1

    return divider
}

func getPDFFiles(folderPath string) ([]string, error) {
    files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	return fileNames, nil
}

func deleteFolderContents(folderPath string) error {
	err := os.RemoveAll(folderPath)
	if err != nil {
		return err
	}

	err = os.Mkdir(folderPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func deleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}
