package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/structs"

	liquid "gopkg.in/osteele/liquid.v1"
)

var renderEngine *liquid.Engine
var templates map[string][]byte

var serviceURL = flag.String("url", "", "service url")
var noCacheFlag = flag.Bool("nocache", false, "disable cache")

func main() {
	flag.Parse()

	renderEngine = liquid.NewEngine()
	renderEngine.RegisterFilter("push", func(a []interface{}, v interface{}) []interface{} {
		return append(a, v)
	})

	renderEngine.RegisterFilter("limitTo", limitTo)

	files, err := ioutil.ReadDir("./")
	if err != nil {
		log.Fatal(err)
	}

	if noCacheFlag == nil || *noCacheFlag == false {
		templates = make(map[string][]byte)
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".svg") {
				name, bytes, err := loadTemplateFromFS(file.Name())
				if err != nil {
					log.Panic(err)
				}
				templates[name] = bytes
			}
		}
	}

	http.HandleFunc("/svg", handleSvg)
	// http.HandleFunc("/png", handlePng)

	http.HandleFunc("/", handleHelp)

	port := os.Getenv("PORT")

	if port == "" {
		port = os.Getenv("HTTP_PLATFORM_PORT")
		if port == "" {
			log.Fatal("$PORT must be set")
		}
	}

	log.Printf("Listening on :%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func loadTemplateFromFS(name string) (string, []byte, error) {
	if strings.HasSuffix(name, ".svg") {
		f, err := os.Open(name)
		if err != nil {
			return "", nil, err
		}

		var buf bytes.Buffer
		_, err = io.Copy(&buf, f)
		if err != nil {
			return "", nil, err
		}
		return name[:len(name)-4], buf.Bytes(), nil
	}
	return "", nil, errors.New("not found")
}

func limitTo(a []string, maxSize int) (result []string) {
	for i := 0; i < len(a); i++ {
		if len(a[i]) <= maxSize {
			result = append(result, a[i])
		} else {
			result = append(result, a[i][:maxSize])
			result = append(result, limitTo([]string{a[i][maxSize:]}, maxSize)...)
		}
	}
	return result
}

type binding struct {
	Lines        []string `structs:",omitempty"`
	Width        int      `structs:",omitempty"`
	Radius       int      `structs:",omitempty"`
	Padding      int      `structs:",omitempty"`
	PaddingColor string   `structs:",omitempty"`
	Template     string   `structs:",omitempty"`
}

func parseUrlValues(w http.ResponseWriter, r *http.Request) (binding binding) {
	binding.Template = "terminal"
	values := r.URL.Query()
	var err error
	for k, v := range values {
		if len(v) > 0 {
			switch strings.ToLower(k) {
			case "lines":
				binding.Lines = strings.Split(v[0], "\n")
			case "line":
				binding.Lines = []string{v[0]}
			case "width":
				binding.Width, err = strconv.Atoi(v[0])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "radius":
				binding.Radius, err = strconv.Atoi(v[0])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "padding":
				binding.Padding, err = strconv.Atoi(v[0])
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "padding-color":
				binding.PaddingColor = v[0]
			case "template":
				binding.Template = v[0]
			}
		}
	}
	return binding
}

func handleSvg(w http.ResponseWriter, r *http.Request) {
	binding := parseUrlValues(w, r)

	w.Header().Set("Content-Type", "image/svg+xml")

	if err := writeSvg(binding, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// func handlePng(w http.ResponseWriter, r *http.Request) {
// 	binding := parseUrlValues(w, r)

// 	var buf bytes.Buffer

// 	if err := writeSvg(binding, &buf); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	svgFile, err := ioutil.TempFile("", "makeconsole-svg-")
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to create svgFile: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}
// 	_, err = io.Copy(svgFile, &buf)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to write svgFile: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}

// 	if err := svgFile.Close(); err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to close svgFile: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}
// 	defer os.Remove(svgFile.Name())

// 	pngFile, err := ioutil.TempFile("", "makeconsole-png-")
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to create pngFile: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}
// 	if err := pngFile.Close(); err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to close pngFile: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}

// 	defer os.Remove(pngFile.Name())

// 	err = exec.Command("inkscape", fmt.Sprintf("--file=%s", svgFile.Name()), fmt.Sprintf("--export-png=%s", pngFile.Name())).Run()
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to call inkscape: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}

// 	f, err := os.Open(pngFile.Name())
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Unable to open png: %v", err.Error()), http.StatusInternalServerError)
// 		return
// 	}
// 	defer f.Close()

// 	w.Header().Set("Content-Type", "image/png")

// 	io.Copy(w, f)
// }

func handleHelp(w http.ResponseWriter, r *http.Request) {
	binding := binding{
		Lines: []string{
			"Usage:",
			"  Call",
			"",
			fmt.Sprintf("      %s/svg", *serviceURL),
			"",
			"With these url parameters:",
			"  lines=Hello%20World%0AThis%20is%20a%20new%20Line!",
			"  width=120           Set a fixed width",
			"  template=terminal   Use a template (templates available: terminal, browser)",
			"  padding=4           Set a padding",
			"  padding-color=      Set the color of the padding",
			"  radius=8            Set a custom radius",
			"",
			"Example:",
			fmt.Sprintf("      %s/svg?lines=Hello%%20World", *serviceURL),
			"",
			"Version 1.0.1",
			"Source Code:",
			"      https://github.com/Eun/makeconsole",
		},
		Template: "terminal",
	}
	w.Header().Set("Content-Type", "image/svg+xml")

	if err := writeSvg(binding, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func writeSvg(binding binding, w io.Writer) error {
	var buf []byte
	if noCacheFlag == nil || *noCacheFlag == false {
		var ok bool
		buf, ok = templates[binding.Template]
		if !ok {
			buf, ok = templates["terminal"]
			if !ok {
				return errors.New("unable to find default template `terminal'")
			}
			binding.Lines = []string{fmt.Sprintf("No such template `%s'", binding.Template)}
			binding.Width = 0
		}
	} else {
		var err error
		_, buf, err = loadTemplateFromFS(binding.Template + ".svg")
		if err != nil {
			return err
		}
	}

	// encode data
	for i, line := range binding.Lines {
		binding.Lines[i] = html.EscapeString(line)
	}

	out, sourceErr := renderEngine.ParseAndRender(buf, structs.Map(binding))
	if sourceErr != nil {
		return sourceErr.Cause()
	}

	_, err := w.Write(out)
	return err
}
