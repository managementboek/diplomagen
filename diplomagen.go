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

	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		return err
	}

	if isEncrypted {
		// If encrypted, try decrypting with an empty one.
		// Can also specify a user/owner password here by modifying the line below.
		auth, err := pdfReader.Decrypt([]byte(""))
		if err != nil {
			fmt.Printf("Decryption error: %v\n", err)
			return err
		}
		if !auth {
			fmt.Println(" This file is encrypted with opening password. Modify the code to specify the password.")
			return nil
		}
	}

	// Print trailer
	if objNum == -1 {
		trailer, err := pdfReader.GetTrailer()
		if err != nil {
			return err
		}

		fmt.Printf("Trailer: %s\n", trailer.String())
		return nil
	}

	obj, err := pdfReader.GetIndirectObjectByNumber(objNum)
	if err != nil {
		return err
	}

	fmt.Printf("Object %d: %s\n", objNum, obj.String())

	if stream, is := obj.(*pdfcore.PdfObjectStream); is {
		decoded, err := pdfcore.DecodeStream(stream)
		if err != nil {
			return err
		}
		fmt.Printf("Decoded:\n%s", decoded)
	} else if indObj, is := obj.(*pdfcore.PdfIndirectObject); is {
		fmt.Printf("%T\n", indObj.PdfObject)
		fmt.Printf("%s\n", indObj.PdfObject.String())
	}

	return nil
}
