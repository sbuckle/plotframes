package main

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	flag "github.com/spf13/pflag"
)

type Frame struct {
	PictType string `xml:"pict_type,attr"`
	KeyFrame int    `xml:"key_frame,attr"`
	PktSize  int    `xml:"pkt_size,attr"`
	Count    int
}

//go:embed script.tmpl
var tmpl string

var (
	term   string
	output string
	stream string
)

func init() {
	flag.StringVarP(&term, "terminal", "t", DefaultTerminal, "Terminal type")
	flag.StringVarP(&output, "output", "o", "", "Set the name of the output file")
	flag.StringVarP(&stream, "stream", "s", "v", `Specify stream. Default value is "v"`)
}

func main() {
	for _, p := range []string{"ffprobe", "gnuplot"} {
		if _, err := exec.LookPath(p); err != nil {
			fmt.Fprintf(os.Stderr, "'%s' must be installed. Couldn't find it on the system path.\n", p)
			os.Exit(1)
		}
	}

	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println("Usage: plotframes [FILE]")
		os.Exit(1)
	}

	args := []string{
		"-show_entries",
		"frame=pict_type,key_frame,pkt_size",
		"-select_streams",
		stream,
		"-of",
		"xml",
		flag.Arg(0),
	}
	out, err := exec.Command("ffprobe", args...).Output()
	if err != nil {
		log.Fatal(err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(out))
	var elem string
	var frames []Frame
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			elem = se.Name.Local
			if elem == "frame" {
				var f Frame
				decoder.DecodeElement(&f, &se)
				frames = append(frames, f)
			}
		}
	}

	plots := map[string]*os.File{}
	for idx, frame := range frames {
		frame.Count = idx
		f, ok := plots[frame.PictType]
		if !ok {
			f, err = os.CreateTemp("", "dat")
			if err != nil {
				log.Fatal(err) // abort
			}
			defer os.Remove(f.Name())
			plots[frame.PictType] = f
		}
		fmt.Fprintf(f, "%d %d\n", frame.Count, frame.PktSize*8/1000)
	}

	var sb strings.Builder
	linecolors := map[string]string{
		"I": "red",
		"P": "green",
		"B": "blue",
	}
	sep := ""
	for pictType, val := range plots {
		fmt.Fprintf(&sb, "%s \"%s\" title \"%s frames\" with impulses linecolor rgb \"%s\"",
			sep, val.Name(), pictType, linecolors[pictType])
		sep = ","
	}

	// Generate the gnu plot script
	f, err := os.CreateTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	t := template.Must(template.New("script").Parse(tmpl))
	var data struct {
		Cmd    string
		Term   string
		Output string
	}
	data.Cmd = sb.String()
	data.Term = term
	data.Output = output

	err = t.Execute(f, data)
	if err != nil {
		log.Fatal(err)
	}

	// Run gunplot
	err = exec.Command("gnuplot", "--persist", f.Name()).Run()
	if err != nil {
		log.Fatal(err)
	}
}
