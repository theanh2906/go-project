package main

import (
	"fmt"
	"golang.org/x/net/html"
	"os"
	"sort"
	"strings"
)

type TranslateTag struct {
	Raw     string
	AttrVal string
}

func extractTranslateTags(doc *html.Node, attr string) []TranslateTag {
	var tags []TranslateTag
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "translate" {
			var val string
			for _, a := range n.Attr {
				if a.Key == attr {
					val = a.Val
					break
				}
			}
			var b strings.Builder
			err := html.Render(&b, n)
			if err != nil {
				return
			}
			tags = append(tags, TranslateTag{Raw: b.String(), AttrVal: val})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return tags
}

func replaceTranslateTags(doc *html.Node, sorted []*html.Node) {
	var idx int
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "translate" {
			parent := n.Parent
			if parent != nil && idx < len(sorted) {
				parent.InsertBefore(sorted[idx], n)
				parent.RemoveChild(n)
				idx++
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}

func cloneNode(n *html.Node) *html.Node {
	var b strings.Builder
	err := html.Render(&b, n)
	if err != nil {
		return nil
	}
	newNode, _ := html.ParseFragment(strings.NewReader(b.String()), nil)
	if len(newNode) > 0 {
		return newNode[0]
	}
	return nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: translate_sorter <file> <attribute> <order: 1|-1>")
		os.Exit(1)
	}
	filePath := os.Args[1]
	attr := os.Args[2]
	order := os.Args[3]
	ord := 1
	if order == "-1" {
		ord = -1
	} else if order != "1" {
		fmt.Println("Order must be 1 (asc) or -1 (desc).")
		os.Exit(1)
	}
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)
	inputBytes, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	doc, err := html.Parse(strings.NewReader(string(inputBytes)))
	if err != nil {
		fmt.Printf("Error parsing HTML: %v\n", err)
		os.Exit(1)
	}
	tags := extractTranslateTags(doc, attr)
	if len(tags) == 0 {
		fmt.Println("No <translate> tags found.")
		os.Exit(1)
	}
	sort.Slice(tags, func(i, j int) bool {
		if ord == 1 {
			return tags[i].AttrVal < tags[j].AttrVal
		}
		return tags[i].AttrVal > tags[j].AttrVal
	})
	// Write only sorted <translate> tags back to the file
	var b strings.Builder
	for _, t := range tags {
		b.WriteString(t.Raw)
		b.WriteString("\n")
	}
	err = os.WriteFile(filePath, []byte(b.String()), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}
}
