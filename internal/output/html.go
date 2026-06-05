package output

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func ExtractText(htmlContent []byte) string {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return string(htmlContent)
	}

	var buf strings.Builder
	extractFromNode(&buf, doc)
	return strings.TrimSpace(buf.String())
}

func extractFromNode(w io.StringWriter, n *html.Node) {
	if n.Type == html.TextNode {
		text := strings.TrimSpace(n.Data)
		if text != "" {
			w.WriteString(text)
			w.WriteString(" ")
		}
		return
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "script", "style", "nav", "header", "footer":
			return
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr", "br":
			w.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractFromNode(w, c)
	}

	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr":
			w.WriteString("\n")
		}
	}
}
