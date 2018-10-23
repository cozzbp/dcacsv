package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/gocarina/gocsv"
)

type FASTA struct {
	Organism string `csv:"Organism"`
	File1    string `csv:"FILE1"`
	File2    string `csv:"FILE2"`
}

type Sequence struct {
	Sequence string
	Full     string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func GetStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str, end)
	return str[s:e]
}

func buildMap(fileName string) map[string]Sequence {
	dat, err := ioutil.ReadFile(fileName)
	check(err)

	replaced := strings.Replace(string(dat), "\r\n", "", -1)
	replaced = strings.Replace(string(dat), "\n", "", -1)

	fastas := strings.FieldsFunc(replaced, func(r rune) bool {
		return r == '>'
	})

	r, err := regexp.Compile(".*[A-Z].*|\\d")
	check(err)

	mapped := make(map[string]Sequence)
	for _, field := range fastas {
		name := GetStringInBetween(field, "[", "]")
		nameWords := strings.Split(name, " ")
		finalName := ""
		for i, word := range nameWords {
			if i == 0 {
				finalName = word
			} else if !r.MatchString(word) {
				finalName += " " + word
			} else {
				break
			}
		}
		seqs := strings.Split(field, "]")
		if len(seqs) < 2 {
			continue
		}
		seq := seqs[1]

		existing, ok := mapped[finalName]
		if !ok {
			mapped[finalName] = Sequence{
				Sequence: seq,
				Full:     field,
			}
		} else {
			if len(seq) > len(existing.Sequence) {
				mapped[finalName] = Sequence{
					Sequence: seq,
					Full:     field,
				}
			}
		}
	}
	return mapped
}

func main() {

	firstFile := flag.String("first", "asdf", "FASTA file")
	secondFile := flag.String("second", "asdf", "FASTA file")
	outFile := flag.String("outfile", "out.csv", "Output csv file")

	flag.Parse()
	if firstFile == nil || secondFile == nil || outFile == nil {
		log.Fatal("Format is: first=filename.txt second=filename2.txt outfile=output.csv")
	}

	fmt.Printf("FIRST FILE=%s SECOND FILE=%s", *firstFile, *secondFile)
	first := buildMap(*firstFile)
	second := buildMap(*secondFile)

	count := 0
	fastas := []*FASTA{}
	for k, v := range first {
		sv, ok := second[k]
		if ok {
			count++
			fastas = append(fastas, &FASTA{
				Organism: k,
				File1:    v.Full,
				File2:    sv.Full,
			})
		}
	}

	file, err := os.Create(*outFile)
	check(err)
	gocsv.MarshalFile(fastas, file)

	file.Close()
	check(err)
}

func setStructTag(f *reflect.StructField) {
	f.Tag = "`json:name-field`"
}

func getStructTag(f reflect.StructField) string {
	return string(f.Tag)
}
