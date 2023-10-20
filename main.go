package main

import (
	"bytes"
	"cmp"
	_ "embed"
	"encoding/json"
	"flag"
	"go/build"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"golang.org/x/text/language"
)

//go:embed code.tmpl
var codeTmpl string

// Translation holds data for generated file
type Translation struct {
	Package string
	Lang    string
	Pairs   []Pair
}

// Pair of Key/Value translation
type Pair struct {
	Key   string
	Value string
}

// Pkg returns go package name
func Pkg(dir string) (string, error) {
	pkg, err := build.ImportDir(dir, build.IgnoreVendor)
	if err != nil {
		return "", err
	}
	return pkg.Name, nil
}

// Lang extracts language code from file name
func Lang(fname string) (string, error) {
	name := strings.TrimSuffix(fname, filepath.Ext(fname))
	if _, err := language.Parse(name); err != nil {
		return "", err
	}
	return name, nil
}

// ParseARB parses ARB file
func ParseARB(r io.Reader) (map[string]string, error) {
	data := make(map[string]string)
	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}
	for k := range data {
		if strings.HasPrefix(k, "@") {
			delete(data, k)
		}
	}
	return data, nil
}

// Generate translation file
func Generate(fname string) error {
	pkg, err := Pkg(".")
	if err != nil {
		return err
	}

	lang, err := Lang(fname)
	if err != nil {
		return err
	}

	fd, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer fd.Close()

	tr, err := ParseARB(fd)
	if err != nil {
		return err
	}

	t := Translation{Package: pkg, Lang: lang}
	for k, v := range tr {
		t.Pairs = append(t.Pairs, Pair{Key: k, Value: v})
	}
	slices.SortFunc(t.Pairs, func(a, b Pair) int {
		return cmp.Compare(a.Key, b.Key)
	})

	var buf bytes.Buffer
	if err := template.Must(template.New("").Parse(codeTmpl)).Execute(&buf, t); err != nil {
		return err
	}

	f, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	if err := os.WriteFile(lang+"_i18n.go", f, 0644); err != nil {
		return err
	}

	return nil
}

func main() {
	file := flag.String("file", "*.arb", "ARB files")
	flag.Parse()

	files, err := filepath.Glob(*file)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if err := Generate(f); err != nil {
			log.Fatal(err)
		}
	}
}
