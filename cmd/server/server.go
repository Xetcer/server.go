package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	// "strings"
	// "server.go/testpack"
)

// https://translated.turbopages.org/proxy_u/en-ru.ru.e44ca1b8-66c587b0-6eaa3156-74722d776562/https/www.geeksforgeeks.org/how-to-build-a-simple-web-server-with-golang/
// http://localhost:8080 обращаться по этому адресу.

func ReadJsonFile(filePath string) (string, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("Файл не существует", err)
		return "", err
	}

	// Читаем JSON файл
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Ошибка при открытии файла:", err)
		return "", err
	}
	defer file.Close()

	//Считываем содержимое файла
	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("Ошибка при чтении файла:", err)
		return "", err
	}
	return string(byteValue), nil
}

func WriteJsonToFile(jsonStr string, filePath string) error {
	err := os.WriteFile(filePath, []byte(jsonStr), 0777)
	if err != nil {
		fmt.Println("Ошибка при записи файла:", err)
		return err
	}
	return nil
}

func errorResponse(w http.ResponseWriter, message string, httpStatusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, _ := json.Marshal(resp)
	w.Write(jsonResp)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		headerContentTtype := r.Header.Get("Content-Type")
		switch {
		case headerContentTtype == "text/plain;charset=UTF-8":
			fallthrough
		case headerContentTtype == "application/json":
			fmt.Println("Content type:", headerContentTtype)
			fmt.Println("filepath URL:", r.URL.Path)
			filePath := "static" + r.URL.Path

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Could not read request body", http.StatusBadRequest)
				return
			}
			defer r.Body.Close()
			JsonChangesStr := string(body)
			if len(JsonChangesStr) != 0 {
				fmt.Println("POST-ed json string is: ", JsonChangesStr)
			} else {
				fmt.Println("POST-ed empty string: ")
				errorResponse(w, "", http.StatusOK)
				return
			}

			// Если файл существует, то загрузим все из него
			fileExist := false
			jsonString, err := ReadJsonFile(filePath)
			if err != nil {
				fmt.Println("Cant read file " + filePath + ", error:" + err.Error())
			} else {
				fmt.Println(filePath + " is loaded")
				fileExist = true
			}
			// Сохраняем исходную строку
			originalJsonStr := jsonString

			// Декодируем JSON в мапы
			var original, changes map[string]interface{}
			if fileExist {
				original, err = jsonToMap(originalJsonStr)
				if err != nil {
					fmt.Println("Decoing original JSON error:", err)
					http.Error(w, "Decoing original JSON error:", http.StatusBadRequest)
					return
				}
			}
			changes, err = jsonToMap(JsonChangesStr)
			if err != nil {
				fmt.Println("Decoing changed JSON error:", err)
				http.Error(w, "Decoing changed JSON error:", http.StatusBadRequest)
				return
			}

			// Заменяем значения в оригинальном JSON
			if fileExist {
				replaceValues(original, changes)
			} else {
				original = changes
			}

			// Кодируем обратно в JSON
			// читабельный формат для отладки
			newJson, _ := json.MarshalIndent(original, "", "    ")
			fmt.Println(string(newJson))
			shortJson, err := json.Marshal(original)
			if err != nil {
				fmt.Println("Marshal complite JSON error:", err)
				http.Error(w, "Marshal complite JSON error:", http.StatusNotModified)
			} else {
				err = WriteJsonToFile(string(shortJson), filePath)
				if err != nil {
					fmt.Println("Writing complite JSON-file error:", err)
					http.Error(w, "Writing complite JSON-file error:", http.StatusNotModified)
				} else {
					fmt.Println("JSON data is updated in file: " + filePath)
				}
			}
		default:
			fmt.Println("Uncknown content-type:", headerContentTtype)
			errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
			return
		}
	} else if r.Method == "GET" {
		switch r.RequestURI {
		case "/":
			http.ServeFile(w, r, "static/index.html")
		default:
			filepath := "static" + r.RequestURI
			http.ServeFile(w, r, filepath)
		}
	}
}

func jsonToMap(jsonStr string) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
		return jsonMap, err
	}
	return jsonMap, nil
}

// Рекурсивная замена значений в source по структуре из template
func replaceValues(source map[string]interface{}, template map[string]interface{}) {
	for key, value := range template {
		if _, ok := value.(map[string]interface{}); ok {
			// Если значение в шаблоне — это объект
			if _, exists := source[key]; exists {
				if subSource, isMap := source[key].(map[string]interface{}); isMap {
					replaceValues(subSource, value.(map[string]interface{}))
				} else {
					// Если ключ существует, но это не объект, заменяем
					source[key] = value
				}
			} else {
				// Если ключ отсутствует в source, добавляем его
				source[key] = value
			}
		} else {
			// Если значение в шаблоне — это обычное значение, заменяем
			source[key] = value
		}
	}
}

func IsPortCorrect(port string) bool {
	// проверяем что порт это цифры от 1 - 5
	b := []byte(port)
	re := regexp.MustCompile(`^\d{1,5}$`)
	return re.Match(b)
}

// запуск из командной строки с параметрами .\server.exe 80
func main() {
	// port := flag.String("port", "8080", "Порт")
	// flag.Parse()
	port := "8080"
	args := os.Args
	if len(args) == 2 {
		if IsPortCorrect(args[1]) {
			port = args[1]
		}
	}
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	http.HandleFunc("/", httpHandler)

	fmt.Println("Server started on  http://" + addr + "/ 'ctr+c' for close server")
	log.Fatal("ListenAndServe: ", http.ListenAndServe(addr, nil))
}
