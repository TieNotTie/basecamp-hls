package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	fileName      string
	fileNameNoExt string
	fullUrlFile   string
)

func main() {

	const (
		port     = "8080"
		videoDir = "output"
	)

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/clean", cleanHandler)

	http.Handle("/", addHeaders(http.FileServer(http.Dir(videoDir))))

	fmt.Printf("Starting server at port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

func cleanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}
	err := os.RemoveAll("files")
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll("output")
	if err != nil {
		log.Fatal(err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	fullUrlFile = getUrl(w, r)
	if len(fullUrlFile) == 0 {
		return
	}

	// Build fileName from fullPath
	buildFileName()

	fmt.Fprintf(w, fileNameNoExt)

	downloadFile()

	chunkFile()
}

func getUrl(w http.ResponseWriter, r *http.Request) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return ""
	}
	if !strings.Contains(string(body), "url") {
		fmt.Fprintf(w, "ERROR: url can't be found")
		return ""
	}

	reg := regexp.MustCompile(`(http|ftp|https):\/\/([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`)

	index := reg.FindIndex(body)

	url := (string(body)[index[0]:index[1]])
	return url
}

func downloadFile() {

	folderName := "files"
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.Mkdir(folderName, 0755)
	}

	out, _ := os.Create(folderName + "/" + fileName)
	defer out.Close()

	resp, _ := http.Get(fullUrlFile)
	defer resp.Body.Close()

	io.Copy(out, resp.Body)
}

func chunkFile() {
	folderName := "output"
	if _, err := os.Stat(folderName); os.IsNotExist(err) {
		os.Mkdir(folderName, 0755)
	}

	ffmpegCommand := exec.Command("ffmpeg", "-i", "files/"+fileName, "-b:a", "128k", "-f", "segment", "-segment_time", "10", "-segment_list", "output/"+fileNameNoExt+".m3u8", "-segment_format", "mpegts", "-vcodec", "libx264", "-acodec", "aac", "output/"+fileNameNoExt+"%03d.ts")

	//ffmpegCommand.Stderr = os.Stderr
	//ffmpegCommand.Stdout = os.Stdout

	//fmt.Println(ffmpegCommand.Args)
	if err := ffmpegCommand.Run(); err != nil {
		fmt.Println("Error while executing command!")
	}
}

func buildFileName() {
	fileUrl, _ := url.Parse(fullUrlFile)

	path := fileUrl.Path

	segments := strings.Split(path, "/")

	//get file name
	fileNameNoExt = segments[len(segments)-1]

	fileNameNoExt = string(hex.EncodeToString([]byte(fileNameNoExt)))

	fileNameNoExt = fileNameNoExt[0:16]

	//get file extension
	fileType := filepath.Ext(fullUrlFile)

	fileName = fileNameNoExt + fileType
}
