package main

import (
	"code.google.com/p/goauth2/oauth"
	drive "code.google.com/p/google-api-go-client/drive/v2"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

// OAuth
var client *http.Client
var config = &oauth.Config{
	ClientId:     "", // Set by --clientid
	ClientSecret: "", // Set by --secret
	Scope:        "", // filled in per-API
	AuthURL:      "https://accounts.google.com/o/oauth2/auth",
	TokenURL:     "https://accounts.google.com/o/oauth2/token",
}

// Flags
var (
	clientId   = flag.String("clientid", "", "OAuth Client ID")
	secret     = flag.String("secret", "", "OAuth Client Secret")
	mimeType   = flag.String("mimeType", "text/html", "MIME type to download")
	cacheToken = flag.Bool("cachetoken", true, "cache the OAuth token")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: gdriver [flags] <fileId>\n")
	os.Exit(2)
}

// DownloadFile downloads the content of a given file object
func DownloadFile(url string, out io.WriteCloser) {

	// Make the request.
	r, err := client.Get(url)
	if err != nil {
		log.Fatal("Get:", err)
	}
	defer r.Body.Close()

	// Write the response to standard output.
	io.Copy(out, r.Body)
}

func getExportLink(file *drive.File, mimeType string) string {
	// TODO - return file.ExportLinks[*mimeType]
	var format string
	switch mimeType {
	case "text/html":
		format = "html"
	}
	return fmt.Sprintf("https://docs.google.com/feeds/download/documents/export/Export?id=%s&exportFormat=%s", file.Id, format)
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}

	fileId := flag.Arg(0)

	config.Scope = drive.DriveScope
	config.ClientId = *clientId
	config.ClientSecret = *secret

	client = getOAuthClient(config)

	service, _ := drive.New(client)

	driveFile, err := service.Files.Get(fileId).Do()
	log.Printf("Got file %s (err: %#v)", driveFile.Title, err)

	downloadUrl := getExportLink(driveFile, *mimeType)

	inFormat := "html"
	outFile := fmt.Sprintf("%s.pdf", driveFile.Title)
	cmd := exec.Command("pandoc", "-f", inFormat, "-o", outFile)

	pandocIn, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Downloading %s ...", downloadUrl)
	DownloadFile(downloadUrl, pandocIn)
	pandocIn.Close()

	log.Printf("Creating for %s ...", outFile)

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
