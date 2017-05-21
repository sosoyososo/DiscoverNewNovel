package FileAction

import (
	"fmt"
	"os"
)

func readFileContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if nil != err {
		return "", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if nil != err {
		return "", err
	}

	size := fileInfo.Size()
	buffer := make([]byte, size)
	_, err = file.Read(buffer)
	if nil != err {
		return "", err
	}

	return string(buffer), nil
}

func saveFileContent(content string, fileName string) error {
	fmt.Printf("保存文件 %s\n", fileName)
	savePath := novConfig.Config.SavePath
	filePath := savePath + fileName
	file, err := os.Open(filePath)
	if err != nil {
		file, err = os.Create(filePath)
		if err != nil {
			return err
		}
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}
