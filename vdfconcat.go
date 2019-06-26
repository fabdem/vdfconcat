package main

//	F.Demurger 2019-02
//  	2 args - 2 vdf files names file1, file2
//                 output = file1 + file2
//  UTF8 and UTF16 encoding of both files need to be the same.
//  Header of file1 is used for concatanated result
//
//	cross compilation AMD64:  env GOOS=windows GOARCH=amd64 go build vdfconcat.go

import (
	//"bufio"
	"bytes"
	"flag"
	"fmt"
//	"io"
	"log"
	"os"
)

// Code for:
var cParenth []byte
var cCloseParenth []byte
var cDbleQuote []byte
var cDbleSlash []byte
var cBackSlash []byte
var cLineFeed []byte
var cCarriageRet []byte
var cCRLF []byte
var cTab []byte

var bom []byte

var UTF16bomle []byte
var UTF16bombe []byte
var UTF8bom    []byte


func main() {

	var versionFlg bool
	const usageVersion   = "Display Version"

	UTF16bomle = []byte{0xFF, 0xFE}
	UTF16bombe = []byte{0xFE, 0xFF}
	UTF8bom = []byte{0xEF, 0xBB, 0xBF}

	flag.BoolVar(&versionFlg, "version", false, usageVersion)
	flag.BoolVar(&versionFlg, "v", false, usageVersion + " (shorthand)")
	flag.Usage = usageIs  // Display app usage

	flag.Parse()

	if versionFlg {
		fmt.Printf("Version %s\n", "2019-06  v0.1.4")
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		usageIs()  // Display usage
		log.Fatalf("Missing parameters\n")
	}

	firstFileName := os.Args[1]
	secondFileName := os.Args[2]

	// Let's start with checking both file's formats (does it starts with FFFE?)
	// Game files are apparently UTF16 BOM the rest is UTF8 so check that and consistency

	f1, err1 := os.Open(firstFileName)
	if err1 != nil {
		log.Fatalf("Can't open : %s - %s", firstFileName, err1)
	}
	defer f1.Close()

	fi1, err1 := f1.Stat() // Get file size
	if err1 != nil {
		log.Fatalf("Can't read file size : %s - %s", firstFileName, err1)
	}
	encoding := guessEncoding(f1)
	// fmt.Printf("Encoding f1= %s \n", encoding)

	f2, err2 := os.Open(secondFileName)
	if err2 != nil {
		log.Fatalf("Can't open : %s - %s", secondFileName, err2)
	}
	defer f2.Close()

	fi2, err2 := f2.Stat() // Get file size
	if err2 != nil {
		log.Fatalf("Can't read file size : %s - %s", secondFileName, err2)
	}

	switch {
		case encoding == "UTF16bomle":
		case encoding == "UTF16bombe":
		case encoding == "UTF8bom":
		case encoding == "UTF8":
		default:
			log.Fatalf("Only files encoded in UTF16 BOM LE/BE or UTF8/BOM can be processed.\n")
	}

	// Guess encoding of f2 and remove the code header is any
	if encoding != guessEncoding(f2) {
		log.Fatalf("Files have different encoding!\n")
	}


	// Defines the encoding of the special chars
	switch {
		case encoding == "UTF8":
			// Set special char encoding for UTF8:
			cParenth = []byte{'{'}
			cCloseParenth = []byte{'}'}
			cDbleQuote = []byte{'"'}
			cDbleSlash = []byte{'/', '/'}
			cTab = []byte{'\t'}
			cBackSlash = []byte{'\\'}
			cLineFeed, cCarriageRet = []byte{'\n'}, []byte{'\r'}
			cCRLF = append([]byte{'\r'}, []byte{'\n'}...)

		case encoding == "UTF8bom":
			// Set special char encoding for UTF8bom:
			cParenth = []byte{'{'}
			cCloseParenth = []byte{'}'}
			cDbleQuote = []byte{'"'}
			cDbleSlash = []byte{'/', '/'}
			cTab = []byte{'\t'}
			cBackSlash = []byte{'\\'}
			cLineFeed, cCarriageRet = []byte{'\n'}, []byte{'\r'}
			cCRLF = append([]byte{'\r'}, []byte{'\n'}...)
			bom = []byte{0xEF, 0xBB, 0xBF}

		case encoding == "UTF16bomle":
			// Set special char encoding for UTF16bomle:
			cParenth = []byte{'{', 0x00}
			cCloseParenth = []byte{'}', 0x00}
			cDbleQuote = []byte{'"', 0x00}
			cDbleSlash = []byte{'/', 0x00, '/', 0x00}
			cTab = []byte{'\t', 0x00}
			cBackSlash = []byte{'\\', 0x00}
			cLineFeed, cCarriageRet = []byte{'\n', 0x00}, []byte{'\r', 0x00}
			cCRLF = append([]byte{'\r', 0x00}, []byte{'\n', 0x00}...)
			bom = []byte{0xFF, 0xFE}

		case encoding == "UTF16bombe":
			// Set special char encoding for UTF16bomle:
			cParenth = []byte{0x00, '{'}
			cCloseParenth = []byte{0x00, '}'}
			cDbleQuote = []byte{0x00, '"'}
			cDbleSlash = []byte{0x00, '/', 0x00, '/'}
			cTab = []byte{0x00, '\t'}
			cBackSlash = []byte{0x00, '\\'}
			cLineFeed, cCarriageRet = []byte{0x00,'\n'}, []byte{0x00,'\r'}
			cCRLF = append([]byte{0x00,'\r'}, []byte{0x00,'\n'}...)
			bom = []byte{0xFE, 0xFF}

		default:
			// We should never end up here
			log.Fatalf("Only files encoded in UTF16 BOM LE or UTF8/bom can be processed.\n")
	}

	// Skip header of 2nd file
	skipHeader(f2)


	//adjust the capacity (file max characters) - the default size is 4096 bytes!!!!
	var maxCapacity int64

	if fi1.Size() >= fi2.Size() { // ...from the largest of the 2 files
		maxCapacity = fi1.Size() + 1024
	} else {
		maxCapacity = fi2.Size() + 1024
	}

	buf := make([]byte, maxCapacity)

    // fmt.Printf("Buffer size %d\n", maxCapacity)

    var sizeRead int
    var err error

    // Read entirety of f1 in buffer
	if sizeRead, err = f1.Read(buf); err != nil {
		log.Fatalf("Can't read file %s - %s\n",firstFileName, err)
	}

  // fmt.Printf("Lenght read %d\n", sizeRead)

	// Look for the last 2 x '}' of the footer in order to remove them
	i := sizeRead - len(cCloseParenth) // last idx
  firstParenth, done := false, false

	for ;! done; {
	      switch {
            case bytes.Equal(buf[i:i+len(cCloseParenth)], cCloseParenth):
                if firstParenth {
                    done = true
										break // found the 2nd } we're done i points on the last '}'
                } else {
                    firstParenth = true
                }
                i -= len(cCloseParenth)
                if i == 0 {
                    log.Fatalf("Format error file %s\n",firstFileName)
                }

            default:
                i -= len(cCloseParenth)
                if i == 0 {
                    log.Fatalf("Format error file %s\n",firstFileName)
                }
        }
    }

	// Output file encoding
	switch {
		case encoding == "UTF16bomle":
			os.Stdout.Write(UTF16bomle[0:])
		case encoding == "UTF16bombe":
			os.Stdout.Write(UTF16bombe[0:])
		case encoding == "UTF8bom":
			os.Stdout.Write(UTF8bom[0:])
			//fmt.Printf("Add utf8 BOM\n")
		case encoding == "UTF8":
			// nada
	}

    os.Stdout.Write(buf[0:i])


    // Read entirety of f2 in buffer
	if sizeRead, err = f2.Read(buf); err != nil {
		log.Fatalf("Can't read file %s - %s\n",secondFileName, err)
	}

    //fmt.Printf("Lenght read %d\n", sizeRead)

    os.Stdout.Write(buf[0:sizeRead])
		os.Stdout.Write([]byte("\n"))  // Add a newline at the end of the file

}


// Check the encoding of the text file passed as a parameter and move the file
// pointer to the begining of the data (removing the bom in effect).
//
// Very basic check - detection utf8 could be improved
// Returns a string "UTF16bomle" "UTF16bomle", "UTF8bom" or "UTF8"
func guessEncoding(theFile *os.File) string {

	theFile.Seek(0, 0) // rewind the file - BOM is at the begining

	const BUFF_SIZE = 4

	firstFewBytes := make([]byte, BUFF_SIZE)
	if _, err := theFile.Read(firstFewBytes); err != nil {
		log.Fatalf("guessEncoding() - can't read file - %s", err)
	}

	switch {
	case bytes.Equal(firstFewBytes[0:len(UTF16bomle)], UTF16bomle):
		theFile.Seek(int64(len(UTF16bomle)-BUFF_SIZE), 1) // remove the BOM (relative to position)
		return ("UTF16bomle")
	case bytes.Equal(firstFewBytes[0:len(UTF16bombe)], UTF16bombe):
		theFile.Seek(int64(len(UTF16bombe)-BUFF_SIZE), 1) // remove the BOM (relative to position)
		return ("UTF16bombe")
	case bytes.Equal(firstFewBytes[0:len(UTF8bom)], UTF8bom):
		theFile.Seek(int64(len(UTF8bom)-BUFF_SIZE), 1) // remove the BOM (relative to position)
		return ("UTF8bom")
	default:
		theFile.Seek(int64(-BUFF_SIZE), 1) // rewind file
		return ("UTF8")
	}
}


// Skip header by moving the file beginning past the header
//
// Not very eleguant: search for any '{' outside "" strings
// and move the begining of the file to the last occurence
// Could be improved: inc by 1 char
//
func skipHeader(theFile *os.File) {
	const BUFF_SIZE = 1024  // Read 1KB of data. That should contain the entire header
	var sizeRead int
	var err error
	buf := make([]byte, BUFF_SIZE + 4) // lazy (add few bytes to avoid out-of-range issues)
	if sizeRead, err = theFile.Read(buf); err != nil {
		log.Fatalf("skipHeader() - can't read file - %s", err)
	}

	// fmt.Printf("\n\n\nScan header-size=%d",sizeRead)
	// fmt.Printf("\n\n\n-------------------------\n")
	// os.Stdout.Write(buf[0:sizeRead - 1])
	// fmt.Printf("\n\n\n-------------------------\n")

	relPosition := sizeRead    // Reader offset

	// Scan all the data read
	for i:=0;i < sizeRead; {
        //fmt.Printf("%s",string(buf[i]))
		opening := true

		// Skip strings between "
		for  {
			// Is this a double quote?
			if bytes.Equal(buf[i:i+len(cDbleQuote)], cDbleQuote) {
				if !opening {
					i+=len(cDbleQuote)
					break
				} else {
					opening = false
				}
			}
			i++
			if i >= BUFF_SIZE { // When we reach the end of the buffer we're done
				return
			}
		}

		// Look for a '{'
		for  {
			// Is this a double quote?
			if bytes.Equal(buf[i:i+len(cDbleQuote)], cDbleQuote) {
				break
			}
			// Is this a '{'?
			if bytes.Equal(buf[i:i+len(cParenth)], cParenth) {
				theFile.Seek(int64(i + len(cParenth) - relPosition), 1) // skip the header not sure why i had to add len(cParenth)?????
                // fmt.Printf("\nSeek relative=%d and i=%d\n",int64(i + len(cParenth) - relPosition),i)
				relPosition = i + len(cParenth)  // update reader offset
                // fmt.Printf("relPosition=%d\n", relPosition)
                // fmt.Printf("\nbuf=%s\n",buf[i-5:i+5])
			}
			i++
			if i >= BUFF_SIZE {
				return  // When we reach the end of the buffer we're done
			}
		}
	}
}

func usageIs () {
	fmt.Printf("Usage: %s <file1> <file2> \n",os.Args[0])
	fmt.Printf("Output: Merge both files.\n")
	fmt.Printf("Flag: -version	Display prg version.\n")
	fmt.Printf("\n")
}
