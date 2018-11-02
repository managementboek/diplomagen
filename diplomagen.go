package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	pdfcore "github.com/unidoc/unidoc/pdf/core"
	pdf "github.com/unidoc/unidoc/pdf/model"
)

func main() {
	var (
		strings    = kingpin.Command("strings", "Extract strings from a template PDF").Default()
		stringsPDF = strings.Arg("pdf", "Input PDF").Required().ExistingFile()

		analyze         = kingpin.Command("analyze", "Analyze a template PDF")
		analyzePDF      = analyze.Arg("pdf", "Input PDF").Required().ExistingFile()
		analyzeObjectID = analyze.Flag("objectid", "Object ID to decode").Short('n').Default("-1").Int()

		patch         = kingpin.Command("patch", "Make some edits in a template PDF")
		patchInputPDF = patch.Flag("input", "Input (template) PDF file").Required().Short('i').ExistingFile()
		patchOutput   = patch.Flag("output", "Output PDF file").Required().Short('o').String()
		patchForce    = patch.Flag("overwrite", "Overwrite files without prompting").Short('f').Bool()
		patchActions  = patch.Arg("actions", "Replacements to perform").Strings()
	)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Author("Managementboek.nl")
	kingpin.CommandLine.Help = "diplomagen is a program to replace texts and images in existing PDF files."

	switch kingpin.Parse() {
	case "strings":
		kingpin.FatalIfError(listStrings(*stringsPDF), "Failed to extract usable strings from PDF")
	case "analyze":
		kingpin.FatalIfError(inspectPdfObject(*analyzePDF, *analyzeObjectID), "Failed to analyze PDF")
	case "patch":
		kingpin.FatalIfError(patchPdf(*patchOutput, *patchInputPDF, *patchActions, *patchForce), "Failed to patch PDF")
	}
}

func patchPdf(outputPath, inputPath string, actions []string, forceOverwrite bool) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	pdfReader, err := pdf.NewPdfReader(f)
	if err != nil {
		return err
	}
	trailer, err := pdfReader.GetTrailer()
	if err != nil {
		return err
	}

	patchset, err := ParsePatchset(actions)
	if err != nil {
		return err
	}

	// FIXME: parse version from input document
	out, err := NewObjWriter(outputPath, forceOverwrite, 1, 4)
	if err != nil {
		return err
	}

	nums := pdfReader.GetObjectNums()
	for _, n := range nums {
		o, err := pdfReader.GetIndirectObjectByNumber(n)
		if err != nil {
			return err
		}

		o, err = patchset.ApplyAll(n, o)
		if err != nil {
			return err
		}

		err = out.Write(n, o)
		if err != nil {
			return err
		}
	}

	out.Finalize(trailer)

	return nil
}

func listStrings(inputPath string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	defer f.Close()

	pdfReader, err := pdf.NewPdfReader(f)
	if err != nil {
		return err
	}

	nums := pdfReader.GetObjectNums()
	for _, n := range nums {
		o, err := pdfReader.GetIndirectObjectByNumber(n)
		if err != nil {
			return err
		}

		stream, ok := o.(*pdfcore.PdfObjectStream)
		if !ok {
			continue
		}

		st := stream.Get("Subtype")
		if st != nil {
			continue
		}

		decoded, err := pdfcore.DecodeStream(stream)
		if err != nil {
			return err
		}

		lineNo := 1
		lineStart := 0
		for i, c := range decoded {
			if c == '\n' {
				l := string(decoded[lineStart:i])
				if len(l) > 4 {
					if (l[0:1] == "[" && l[len(l)-3:] == "]TJ") ||
						(l[0:1] == "(" && l[len(l)-3:] == ")Tj") {
						fmt.Printf("%d:%d:%s\n", n, lineNo, l)
					}
				}
				lineNo++
				lineStart = i + 1
			}
		}
	}

	return nil
}

func inspectPdfObject(inputPath string, objNum int) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	defer f.Close()

	pdfReader, err := pdf.NewPdfReader(f)
	if err != nil {
		return err
	}

	// List all Object IDs
	if objNum == -1 {
		fmt.Printf("List of object IDs:\n\n")

		nums := pdfReader.GetObjectNums()
		for _, n := range nums {
			o, err := pdfReader.GetIndirectObjectByNumber(n)
			if err != nil {
				return err
			}

			switch obj := o.(type) {
			case *pdfcore.PdfObjectStream:
				fmt.Printf("Object %d: ", n)

				st := obj.Get("Subtype")
				if st == nil {
					fmt.Printf("Data stream")
				} else if subtype, ok := st.(*pdfcore.PdfObjectName); ok {
					switch *subtype {
					case "Image":
						w, h := 0, 0
						if width, ok := (obj.Get("Width")).(*pdfcore.PdfObjectInteger); ok {
							w = int(int64(*width))
						}
						if height, ok := (obj.Get("Height")).(*pdfcore.PdfObjectInteger); ok {
							h = int(int64(*height))
						}
						if comp, ok := (obj.Get("Filter")).(*pdfcore.PdfObjectName); ok {
							if *comp == "DCTDecode" {
								fmt.Printf("JPEG image %dx%d", w, h)
							} else {
								fmt.Printf("Image %dx%d", w, h)
							}
						} else {
							fmt.Printf("Image %s", o)
						}

					case "Type1C":
						fmt.Printf("Font")

					default:
						fmt.Printf("Stream of type %T %s %s", st, st, o)
					}
				} else {
					fmt.Printf("Stream of type %T %s %s", st, st, o)
				}
				fmt.Printf("\n")

			case *pdfcore.PdfIndirectObject:
				_ = true

			default:
				fmt.Printf("Object %d: %T\n", n, o)
			}
		}

		return nil
	}

	obj, err := pdfReader.GetIndirectObjectByNumber(objNum)
	if err != nil {
		return err
	}

	if stream, is := obj.(*pdfcore.PdfObjectStream); is {
		decoded, err := pdfcore.DecodeStream(stream)
		if err != nil {
			return err
		}
		os.Stdout.Write(decoded)

	} else if indObj, is := obj.(*pdfcore.PdfIndirectObject); is {
		fmt.Printf("Object %d: %s\n", objNum, obj.String())

		fmt.Printf("%T\n", indObj.PdfObject)
		fmt.Printf("%s\n", indObj.PdfObject.String())
	}

	return nil
}

type ObjWriter struct {
	out     *os.File
	offsets []int64
}

func NewObjWriter(path string, force bool, maj, min int) (*ObjWriter, error) {
	mode := os.O_RDWR | os.O_CREATE
	if force {
		mode |= os.O_TRUNC
	}
	out, err := os.OpenFile(path, mode, 0666)
	if err != nil {
		return nil, err
	}

	_, err = fmt.Fprintf(out, "%%PDF-%d.%d\n%%\xe2\xe3\xcf\xd3\n", maj, min)
	if err != nil {
		return nil, err
	}

	rv := &ObjWriter{
		out:     out,
		offsets: make([]int64, 0),
	}
	return rv, nil
}

func (w *ObjWriter) Write(index int, obj pdfcore.PdfObject) error {
	offset, _ := w.out.Seek(0, os.SEEK_CUR)
	w.offsets = append(w.offsets, offset)

	var err error

	switch object := obj.(type) {
	case *pdfcore.PdfObjectStream:
		fmt.Fprintf(w.out, "%d 0 obj\n%s\nstream\n", index, object.PdfObjectDictionary.DefaultWriteString())
		_, err = w.out.Write(object.Stream)
		fmt.Fprintf(w.out, "\nendstream\nendobj\n")

	case *pdfcore.PdfIndirectObject:
		_, err = fmt.Fprintf(w.out, "%d 0 obj\n%s\nendobj\n", index, object.PdfObject.DefaultWriteString())

	default:
		_, err = fmt.Fprintf(w.out, "%d 0 obj\n%s\nendobj\n", index, obj.DefaultWriteString())
	}
	return err
}

func (w *ObjWriter) Finalize(trailer *pdfcore.PdfObjectDictionary) error {
	// Write xref table.
	xrefOffset, _ := w.out.Seek(0, os.SEEK_CUR)
	_, err := fmt.Fprintf(w.out, "%d %d\r\n%.10d %.5d f\r\n", 0, len(w.offsets)+1, 0, 65535)
	if err != nil {
		return err
	}

	for _, offset := range w.offsets {
		_, err := fmt.Fprintf(w.out, "%.10d %.5d n\r\n", offset, 0)
		if err != nil {
			return err
		}
	}

	// Write trailer
	_, err = fmt.Fprintf(w.out, "trailer\n%s\n", trailer.DefaultWriteString())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w.out, "startxref\n%d\n%%%%EOF\n", xrefOffset)
	if err != nil {
		return err
	}

	return w.out.Close()
}

type Patchset []Patch

func (ps Patchset) ApplyAll(index int, obj pdfcore.PdfObject) (pdfcore.PdfObject, error) {
	var err error
	for _, p := range ps {
		if p.ObjectID() == index {
			obj, err = p.Apply(obj)
			if err != nil {
				return nil, err
			}
		}
	}
	return obj, nil
}

func ParsePatchset(patches []string) (Patchset, error) {
	rv := make([]Patch, 0)

	for _, s := range patches {
		switch s[0] {
		case 'S':
			p := ModifyLine{}
			_, err := fmt.Sscanf(s, "S%d:%d:", &p.OID, &p.Line)
			if err != nil {
				return nil, fmt.Errorf("syntax error in replace line command: '%s'", s)
			}
			colons := 0
			var i int
			var c rune
			for i, c = range s {
				if c == ':' {
					colons++
					if colons == 2 {
						break
					}
				}
			}
			p.NewContents = []byte(s[i+1:])

			rv = append(rv, p)
		default:
			return nil, fmt.Errorf("unknown patch command '%s'", s)
		}
	}

	return Patchset(rv), nil
}

// A patch represents one modification operation
type Patch interface {
	// ObjectID returns the object ID this patch applies to
	ObjectID() int

	// Apply applies the patch to the PDF object, and returns a modified version of the object
	Apply(obj pdfcore.PdfObject) (pdfcore.PdfObject, error)
}

type ModifyLine struct {
	OID, Line   int
	NewContents []byte
}

func (m ModifyLine) ObjectID() int {
	return m.OID
}

func (m ModifyLine) Apply(obj pdfcore.PdfObject) (pdfcore.PdfObject, error) {
	str, ok := obj.(*pdfcore.PdfObjectStream)
	if !ok {
		return nil, fmt.Errorf("object is not a stream: %T", obj)
	}

	streamContents, err := pdfcore.DecodeStream(str)
	if err != nil {
		return nil, err
	}

	line := 1
	preambleLength := 0
	trailerOffset := len(streamContents)

	for i, c := range streamContents {
		if c == '\n' {
			line++
			if line == m.Line {
				preambleLength = i + 1
			} else if line == (m.Line + 1) {
				trailerOffset = i
			}
		}
	}

	buf := make([]byte, preambleLength+len(m.NewContents)+(len(streamContents)-trailerOffset))
	copy(buf, streamContents[:preambleLength])
	copy(buf[preambleLength:], m.NewContents)
	copy(buf[preambleLength+len(m.NewContents):], streamContents[trailerOffset:])

	str.Stream = buf
	err = pdfcore.EncodeStream(str)
	if err != nil {
		return nil, err
	}

	return str, nil
}
