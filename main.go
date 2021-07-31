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
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var inputPath string
var outputPath string
var htmlTemplate *tpl.Template

const cssFilename = "mamd.css"

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

	inputPath = filepath.FromSlash(inputPath)
	outputPath = filepath.FromSlash(outputPath)
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

	// get the relative search path (relative to inputPath)
	relPath, err := filepath.Rel(inputPath, searchPath)
	if err != nil {
		return err
	}

	// remove filename from relative path
	relPath = strings.TrimSuffix(relPath, filepath.Base(searchPath))

	// count number of subdirectories so we know how many to offset the .css reference
	numSubdirs := 0
	for _, subdir := range strings.Split(relPath, string(filepath.Separator)) {
		if subdir != "" {
			numSubdirs++
		}
	}
	// create string with .. for every subdirectory
	cssOffset := strings.Repeat("../", numSubdirs)

	// add any subdirectories to the output path
	relPath = filepath.FromSlash(path.Join(strings.TrimRight(outputPath, "\\"), relPath))

	fmt.Printf("%40v | ", searchPath)
	fmt.Printf("%7v | ", info.Size())
	fmt.Printf("%7v | ", numSubdirs)
	fmt.Println(relPath)

	// create the output directory if it doesn't exist
	if !fileExists(relPath) {
		err = os.MkdirAll(relPath, 0755)
		if err != nil {
			return err
		}
	}

	file, err := os.Open(searchPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithExtensions(ChromaExtension),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
			html.WithXHTML(),
		),
	)

	var htmlOut bytes.Buffer
	if err := md.Convert(fileContents, &htmlOut); err != nil {
		return err
	}

	newFilename := filenameWithoutExtension(filepath.Base(searchPath))
	fileTitle := newFilename
	newFilename += ".html"

	outputFile, err := os.Create(path.Join(relPath, newFilename))
	if err != nil {
		return err
	}

	err = htmlTemplate.Execute(outputFile, struct {
		Content           template.HTML
		Title             string
		RelativeCssOffset string
	}{
		Content:           template.HTML(htmlOut.String()),
		Title:             fileTitle,
		RelativeCssOffset: cssOffset,
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {

	initArguments()

	cssCopyTarget := path.Join(outputPath, cssFilename)
	shouldCopyCSS := true

	// check to see if we should overwrite the (existing) css file in the target
	// only copy if source CSS is newer; and also this prevents against damaging the source CSS
	// if you run MaMD and ask to output in the current directory
	if fileExists(cssCopyTarget) {

		shouldCopyCSS = false

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
		if cssDiff < (time.Duration(0) * time.Second) {
			shouldCopyCSS = true
		}
	}

	if shouldCopyCSS {

		fmt.Println("Copying CSS to output...")

		err := fileCopy(cssFilename, cssCopyTarget)
		if err != nil {
			fmt.Printf("Could not copy default CSS file (%s), %v\n", cssFilename, err)
			os.Exit(1)
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
