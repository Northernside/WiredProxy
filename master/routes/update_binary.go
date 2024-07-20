package routes

// utilize post multipart form to upload binary

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"wired.rip/wiredutils/config"
)

func UpdateBinary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	archQuery := r.URL.Query().Get("arch")
	if archQuery == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "arch is required"}`))
		return
	}

	// get file from form
	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "Error getting file"}`))
		return
	}
	defer file.Close()

	// save file
	err = saveFile("updates/wirednode-"+archQuery, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Error saving file"}`))
		return
	}

	// get file hash
	hash := getFileHash("updates/wirednode-" + archQuery)
	if hash == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Error getting file hash"}`))
		return
	}

	config.SetCurrentNodeHash(hash, archQuery)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Binary uploaded"}`))
}

func saveFile(fileName string, file multipart.File) error {
	// create directory
	err := os.MkdirAll("updates", os.ModePerm)
	if err != nil {
		return err
	}

	// create file
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	// copy file
	_, err = io.Copy(f, file)
	if err != nil {
		return err
	}

	return nil
}

func getFileHash(fileName string) string {
	// check if file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return ""
	}

	cmd := exec.Command("sha256sum", fileName)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// format output (remove every backslash)
	// and split by space

	hash := strings.Split(strings.ReplaceAll(string(out), "\\", ""), " ")[0]
	return hash
}
