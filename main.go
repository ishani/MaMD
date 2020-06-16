package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	tpl "html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var inputPath string
var outputPath string
var htmlTemplate *tpl.Template

func initArguments() {
	const (
		inputPathDefault  = ""
		inputPathHelp     = "root of path to walk for input files"
		outputPathDefault = ""
		outputPathHelp    = "where to build output"
	)
	flag.StringVar(&inputPath, "i", inputPathDefault, inputPathHelp)
	flag.StringVar(&outputPath, "o", outputPathDefault, outputPathHelp)

	flag.Parse()
	if inputPath == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func fileCopy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func filenameWithoutExtension(fn string) string {
	return strings.TrimSuffix(fn, path.Ext(fn))
}

func findMarkdown(searchPath string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if filepath.Ext(searchPath) != ".md" {
		return nil
	}

	fmt.Println(searchPath, info.Size())

	file, err := os.Open(searchPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithExtensions(ChromaExtension),
	)

	var htmlOut bytes.Buffer
	if err := md.Convert(fileContents, &htmlOut); err != nil {
		return err
	}

	newFilename := filenameWithoutExtension(filepath.Base(searchPath))
	newFilename += ".html"

	outputFile, err := os.Create(path.Join(outputPath, newFilename))
	if err != nil {
		return err
	}

	err = htmlTemplate.Execute(outputFile, struct{ Content template.HTML }{Content: template.HTML(string(htmlOut.Bytes()))})
	if err != nil {
		return err
	}

	return nil
}

const cssFilename = "mamd.css"

func main() {

	initArguments()

	cssCopyTarget := path.Join(outputPath, cssFilename)

	// check to see if we should overwrite the (existing) css file in the target
	// only copy if source CSS is newer; and also this prevents against damaging the source CSS
	// if you run MaMD and ask to output in the current directory
	if fileExists(cssCopyTarget) {

		sourceInfo, err := os.Stat(cssFilename)
		if err != nil {
			log.Panic(err)
		}

		targetInfo, err := os.Stat(cssCopyTarget)
		if err != nil {
			log.Panic(err)
		}

		sourceTime := sourceInfo.ModTime()
		targetTime := targetInfo.ModTime()
		cssDiff := targetTime.Sub(sourceTime)

		// only copy if we have a newer file
		if cssDiff < (time.Duration(1) * time.Second) {

			fmt.Println("Copying CSS to output...")

			err := fileCopy(cssFilename, cssCopyTarget)
			if err != nil {
				fmt.Printf("Could not copy default CSS file (%s), %v\n", cssFilename, err)
				os.Exit(1)
			}
		}
	}

	var err error

	// write to stdout
	htmlTemplate, err = tpl.ParseFiles("./template.html")
	if err != nil {
		fmt.Printf("Template load error, %v\n", err)
		os.Exit(1)
	}

	err = filepath.Walk(inputPath, findMarkdown)
	if err != nil {
		fmt.Printf("File walk error, %v\n", err)
		os.Exit(1)
	}
}
