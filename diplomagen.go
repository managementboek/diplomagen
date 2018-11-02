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
		analyze         = kingpin.Command("analyze", "Analyze a template PDF").Default()
		analyzePDF      = analyze.Arg("pdf", "Input PDF").Required().ExistingFile()
		analyzeObjectID = analyze.Flag("objectid", "Object ID to decode").Short('n').Default("-1").Int()
	)

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Author("Managementboek.nl")
	kingpin.CommandLine.Help = "diplomagen is a program to replace texts and images in existing PDF files."

	switch kingpin.Parse() {
	case "analyze":
		kingpin.FatalIfError(inspectPdfObject(*analyzePDF, *analyzeObjectID), "Failed to analyze PDF")
	}
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
