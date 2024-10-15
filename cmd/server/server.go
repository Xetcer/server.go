package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	// "server.go/testpack"
)

// https://translated.turbopages.org/proxy_u/en-ru.ru.e44ca1b8-66c587b0-6eaa3156-74722d776562/https/www.geeksforgeeks.org/how-to-build-a-simple-web-server-with-golang/
// http://localhost:8080 обращаться по этому адресу.

//	func sayHello(w http.ResponseWriter, r *http.Request) {
//		fmt.Println(w, "Привет!")
//	}
//
// Тут храним исходный json считанный из файла и сюда же сохраняем изменения для записи в файл.
var jsonString string = ""

var jsonFilePath string = ""

func ReadJsonFile(filePath string) (string, error) {
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
	jsonString = string(byteValue)
	return jsonString, nil
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

func isJSONFile(filePath string) bool {
	return strings.HasSuffix(filePath, ".json")
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		headerContentTtype := r.Header.Get("Content-Type")
		if headerContentTtype != "application/json" {
			errorResponse(w, "Content Type is not application/json", http.StatusUnsupportedMediaType)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Could not read request body", http.StatusBadRequest)
		} else {
			JsonChangesStr := string(body)
			// JsonChangesStr = PrepareJsonStr(JsonChangesStr)
			fmt.Println("POST-ed json string is: ", JsonChangesStr)
			// Сохраняем исходную строку
			updatedJsonStr := jsonString

			// Декодируем JSON в мапы
			var original, replacements map[string]interface{}
			if err := json.Unmarshal([]byte(updatedJsonStr), &original); err != nil {
				fmt.Println("Decoing original JSON error:", err)
				http.Error(w, "Decoing original JSON error:", http.StatusBadRequest)
				return
			}
			if err := json.Unmarshal([]byte(JsonChangesStr), &replacements); err != nil {
				fmt.Println("Decoing changed JSON error:", err)
				http.Error(w, "Decoing changed JSON error:", http.StatusBadRequest)
				return
			}

			// Заменяем значения в оригинальном JSON
			replaceValues(original, replacements)
			// Кодируем обратно в JSON
			// читабельный формат для отладки
			newJson, _ := json.MarshalIndent(original, "", "    ")
			fmt.Println(string(newJson))
			shortJson, err := json.Marshal(original)
			if err != nil {
				fmt.Println("Marshal complite JSON error:", err)
				http.Error(w, "Marshal complite JSON error:", http.StatusNotModified)
			} else {
				err = WriteJsonToFile(string(shortJson), jsonFilePath)
				if err != nil {
					fmt.Println("Writing complite JSON-file error:", err)
					http.Error(w, "Writing complite JSON-file error:", http.StatusNotModified)
				} else {
					fmt.Println("JSON data is updated in file: " + jsonFilePath)
				}
			}
		}
		defer r.Body.Close()
		// http.ServeFile(w, r, jsonFilePath)
	} else if r.Method == "GET" {
		switch r.RequestURI {
		case "/":
			http.ServeFile(w, r, "static/index.html")
		// case "/TD_RELAY.json":
		// 	http.ServeFile(w, r, "static/TD_RELAY.json")
		default:
			filepath := "static" + r.RequestURI
			http.ServeFile(w, r, filepath)
			if isJSONFile(filepath) {
				jsonFilePath = filepath
				var err error
				jsonString, err = ReadJsonFile(jsonFilePath)
				if err != nil {
					fmt.Println("Cant read file " + jsonFilePath + ", error:" + err.Error())
				} else {
					fmt.Println(jsonFilePath + " is loaded")
				}
			}
		}
	}
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

func main() {
	port := flag.String("port", "8080", "Порт")
	// jsonFilePath = flag.String("jsonFilePath", "static/TD_RELAY.json", "Файл json")
	flag.Parse()
	addr := fmt.Sprintf("0.0.0.0:%s", *port)
	// var fileError error
	// jsonString, fileError = ReadJsonFile(*jsonFilePath)
	// if fileError != nil {
	// 	fmt.Println("there is error during find web files: " + fileError.Error())
	// } else {
	// 	fmt.Println(*jsonFilePath + " is loaded")
	// 	// clearedJson := PrepareJsonStr(jsonString)
	// 	// fmt.Println(clearedJson)
	// }
	http.HandleFunc("/", httpHandler)

	fmt.Println("Server started on  http://" + addr + "/ 'ctr+c' for close server")
	log.Fatal("ListenAndServe: ", http.ListenAndServe(addr, nil))
}
