package main

import (
	"strings"

	"github.com/alecthomas/chroma"
	html "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// ChromaCodeRenderer is a renderer.NodeRenderer implementation that
// renders Strikethrough nodes.
type ChromaCodeRenderer struct {
	ghtml.Config
}

// NewChromaCodeRenderer returns a new ChromaCodeRenderer.
func NewChromaCodeRenderer(opts ...ghtml.Option) renderer.NodeRenderer {
	r := &ChromaCodeRenderer{
		Config: ghtml.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

// RegisterFuncs implements renderer.NodeRenderer.RegisterFuncs.
func (r *ChromaCodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderChroma)
}

func (r *ChromaCodeRenderer) renderChroma(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {

	if entering {
		fcb := n.(*ast.FencedCodeBlock)

		var sb strings.Builder
		for i := 0; i < n.Lines().Len(); i++ {
			s := n.Lines().At(i)
			sb.WriteString(string(source[s.Start:s.Stop]))
		}

		inputText := sb.String()

		// Set up a lexer.
		var lexer chroma.Lexer

		// Read the language from the annotation.
		lang := string(fcb.Language(source))

		if lang != "" {
			lexer = lexers.Get(lang)
		} else {
			// Analyze when no language annotation is given.
			lexer = lexers.Analyse(inputText)
		}

		// If no annotation was found and couldn't be analyzed, fallback.
		if lexer == nil {
			lexer = lexers.Fallback
		}

		// Set a syntax highlighting theme
		style := styles.Get("monokailight")
		if style == nil {
			style = styles.Fallback
		}

		// Apply highlighting with Chroma.
		iterator, err := lexer.Tokenise(nil, inputText)
		if err != nil {
			return gast.WalkStop, err
		}

		formatter := html.New(html.WithClasses(false))

		err = formatter.Format(w, style, iterator)
		if err != nil {
			return gast.WalkStop, err
		}
	}
	return gast.WalkContinue, nil
}

type chromaExtension struct {
}

// ChromaExtension binds Chroma to Goldmark
var ChromaExtension = &chromaExtension{}

func (e *chromaExtension) Extend(m goldmark.Markdown) {
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(NewChromaCodeRenderer(), 500),
	))
}
