package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
)

type Frame struct {
	PictType string `xml:"pict_type,attr"`
	KeyFrame int    `xml:"key_frame,attr"`
	PktSize  int    `xml:"pkt_size,attr"`
	Count    int
}

func main() {
	f, err := os.Open("frames.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := xml.NewDecoder(f)
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
			plots[frame.PictType] = f
		}
		fmt.Fprintf(f, "%d %d\n", frame.Count, frame.PktSize*8/1000)
	}

	// Generate the gnu plot script
	f, err = os.CreateTemp("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	const header = `set title "video frame sizes"
set xlabel "frame time"
set ylabel "frame size (Kbits)"
set grid
set terminal x11
	`
	fmt.Fprintln(f, header)
	fmt.Fprint(f, "plot ")

	linecolors := map[string]string{
		"I": "red",
		"P": "green",
		"B": "blue",
	}
	sep := ""
	for pictType, val := range plots {
		fmt.Fprintf(f, "%s \"%s\" title \"%s frames\" with impulses linecolor rgb \"%s\"",
			sep, val.Name(), pictType, linecolors[pictType])
		sep = ","
	}

	// Run gunplot
	err = exec.Command("gnuplot", "--persist", f.Name()).Run()
	if err != nil {
		log.Fatal(err)
	}
}
