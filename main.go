package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gocarina/gocsv/v2"
)

type FASTA struct {
	ShortName string  `csv:"ShortName"`
	Organism1 string  `csv:"Organism1"`
	Hit1      float64 `csv:"PERCENTIDENT1"`
	File1     string  `csv:"FILE1"`
	Organism2 string  `csv:"Organism2"`
	Hit2      float64 `csv:"PERCENTIDENT2"`
	File2     string  `csv:"FILE2"`
}

type Sequence struct {
	OrganismName string
	Sequence     string
	Full         string
	Hit          float64
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
	e := strings.LastIndex(str, end)
	return str[s:e]
}

func buildHitMap(fileName string) map[string]float64 {
	dat, err := ioutil.ReadFile(fileName)
	check(err)

	hitMap := make(map[string]float64)

	lines := strings.Split(string(dat), "\n")

	for _, line := range lines {
		columns := strings.Split(line, ",")
		if len(columns) < 3 {
			continue
		}
		hit, err := strconv.ParseFloat(columns[2], 64)
		check(err)

		hitMap[columns[1]] = hit
	}

	return hitMap
}

func getStrippedName(name string) string {
	r, err := regexp.Compile(".*[A-Z].*|\\d")
	check(err)

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

	return finalName
}

func buildMap(fileName, hitFile string) map[string]Sequence {
	dat, err := ioutil.ReadFile(fileName)
	check(err)

	hitMap := buildHitMap(hitFile)

	replaced := strings.Replace(string(dat), "\r\n", "", -1)
	replaced = strings.Replace(string(dat), "\n", "", -1)

	fastas := strings.FieldsFunc(replaced, func(r rune) bool {
		return r == '>'
	})

	mapped := make(map[string]Sequence)
	for _, field := range fastas {
		if strings.Index(field, "LOW QUALITY PROTEIN") != -1 {
			continue
		}
		if strings.Index(field, "partial") != -1 {
			continue
		}

		accession := strings.Split(field, " ")[0]
		hitPercent := hitMap[accession]

		name := GetStringInBetween(field, "[", "]")
		strippedName := getStrippedName(name)

		seqs := strings.Split(field, "]")
		if len(seqs) < 2 {
			continue
		}
		seq := seqs[1]

		existing, ok := mapped[strippedName]
		if !ok {
			mapped[strippedName] = Sequence{
				OrganismName: name,
				Sequence:     seq,
				Full:         field,
				Hit:          hitPercent,
			}
		} else {
			if hitPercent > existing.Hit {
				mapped[strippedName] = Sequence{
					OrganismName: name,
					Sequence:     seq,
					Full:         field,
					Hit:          hitPercent,
				}
			}
		}
	}
	return mapped
}

func main() {

	file1 := flag.String("file1", "", "FASTA file")
	hit1 := flag.String("hit1", "", "HitFile")
	file2 := flag.String("file2", "", "FASTA file")
	hit2 := flag.String("hit2", "", "HitFile")
	outFile := flag.String("outfile", "out.csv", "Output csv file")

	flag.Parse()
	if file1 == nil || file2 == nil || hit1 == nil || hit2 == nil || outFile == nil {
		log.Fatal("Format is: -file1=filename.txt -hit1=hit.csv -file2=filename2.txt -hit2=hit2.csv -outfile=output.csv")
	}

	fmt.Printf("FILE1=%s HIT1=%s FILE2=%s HIT2=%s", *file1, *hit1, *file2, *hit2)
	var first map[string]Sequence
	var second map[string]Sequence
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		first = buildMap(*file1, *hit1)
		wg.Done()
	}()

	go func() {
		second = buildMap(*file2, *hit2)
		wg.Done()
	}()

	wg.Wait()

	count := 0
	fastas := []*FASTA{}
	for k, v := range first {
		sv, ok := second[k]
		if ok {
			count++
			fastas = append(fastas, &FASTA{
				ShortName: k,
				Organism1: v.OrganismName,
				File1:     v.Full,
				Hit1:      v.Hit,
				Organism2: sv.OrganismName,
				File2:     sv.Full,
				Hit2:      sv.Hit,
			})
		}
	}

	file, err := os.Create(*outFile)
	check(err)
	gocsv.MarshalFile(fastas, file)

	file.Close()
	check(err)
}
