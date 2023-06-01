package main

import (
	"bufio"
	"diff-md/pkg/editdist"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Line struct {
	text *string
	next *Line
	prev *Line
}

type Lines struct {
	start *Line
	end   *Line
}

func (ls *Lines) String() string {
	var res []string
	ls.iterate(func(l *Line) *Line {
		res = append(res, *l.text)
		return nil
	})
	return strings.Join(res, "\n")
}

type RevisePair struct {
	id       string
	original Lines
	revision Lines
}

func (r *RevisePair) String() string {
	res := []string{
		fmt.Sprint("id:", r.id),
		r.original.String(),
		r.revision.String(),
	}
	return strings.Join(res, "\n")
}

func (ls *Lines) append(l *Line) {
	if ls.start == nil {
		ls.start = l
		ls.end = l
	} else {
		l.prev = ls.end
		ls.end.next = l
		ls.end = l
	}
}

func (ls *Lines) iterate(f func(l *Line) *Line) {
	var nxt *Line
	l := ls.start
	if l == nil {
		return
	}
	// FIXME: require infinite loop detection
	for {
		nxt = f(l)
		if nxt == nil {
			nxt = l.next
		}
		if l == ls.end {
			break
		}
		l = nxt
	}
}

func readFileAsLines(path string) Lines {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var lines Lines
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := scanner.Text()
		line := Line{
			text: &txt,
		}
		lines.append(&line)
	}
	return lines
}

const (
	markerStartReq = "%%% req-start "
	markerEndReq   = "%%% req-end %%%"
	markerStartRev = "%%% rev-start "
	markerEndRev   = "%%% rev-end %%%"
)

func extractID(l string) string {
	r, _ := regexp.Compile(" id:([a-zA-Z0-9]+) ")
	res := r.FindStringSubmatch(l)
	if len(res) != 2 {
		panic(fmt.Sprint("Wrong syntax: ", l))
	}
	return res[1]
}

func readBlock(l *Line, endMark string) Lines {
	if *l.next.text == endMark {
		panic(fmt.Sprintf("Empty block found: %s", *l.text))
	}
	ls := Lines{
		start: l,
	}
	for l != nil {
		if *l.text == endMark {
			return ls
		}
		l = l.next
		ls.end = l
	}
	panic("No endMark is found.")
}

func locateTargets(ls Lines) map[string]*RevisePair {
	ps := make(map[string]*RevisePair)
	ls.iterate(func(l *Line) *Line {
		if !strings.Contains(*l.text, "%%%") {
			return nil
		}
		start := *l.text
		switch {
		case strings.Contains(start, markerStartReq):
			blk := readBlock(l, markerEndReq)
			id := extractID(start)
			// assume that req appears before rev
			ps[id] = &RevisePair{
				id:       id,
				original: blk,
			}
			return blk.end.next
		case strings.Contains(start, markerStartRev):
			blk := readBlock(l, markerEndRev)
			id := extractID(start)
			pair := ps[id]
			pair.revision = blk
			return blk.end.next
		default:
			panic("Unexpected syntax. Please check if the file format is correct.")
		}
	})
	return ps
}

var separator map[rune]struct{} = map[rune]struct{}{
	' ':  {},
	'\n': {},
	',':  {},
	'.':  {},
	';':  {},
	'?':  {},
	'!':  {},
}

var separatorStr map[string]struct{} = map[string]struct{}{
	" ":  {},
	"\n": {},
	",":  {},
	".":  {},
	";":  {},
	"?":  {},
	"!":  {},
}

func tokenize(ls *Lines) []string {
	res := make([]string, 0)
	ls.iterate(func(l *Line) *Line {
		b := strings.Builder{}
		for _, c := range *l.text {
			_, prs := separator[c]
			if prs {
				if b.Len() != 0 {
					res = append(res, b.String())
				}
				if c != ' ' {
					res = append(res, string(c))
				}
				b.Reset()
			} else {
				b.WriteRune(c)
			}
		}
		return nil
	})
	return res
}

func addWord(b *strings.Builder, s string) {
	_, isSp := separatorStr[s]
	if b.Len() != 0 && !isSp {
		b.WriteByte(' ')
	}
	b.WriteString(s)
}

func reviseEditScript(edt []editdist.Edit) {
	/*
		- rpl -> del => del -> rpl
		- rpl -> rpl => del -> rpl(combined)
	*/
	for i := 0; i < len(edt)-1; i++ {
		if edt[i].Cmd != editdist.Rpl {
			continue
		}
		if edt[i+1].Cmd == editdist.Del {
			edt[i+1].Cmd = editdist.Rpl
			edt[i].Cmd = editdist.Del
			edt[i+1].Word = edt[i].Word
			edt[i].Word = ""
		} else if edt[i+1].Cmd == editdist.Rpl {
			edt[i+1].Cmd = editdist.Rpl
			edt[i].Cmd = editdist.Del
			edt[i+1].Word = fmt.Sprint(edt[i].Word, " ", edt[i+1].Word)
		}
	}
}

func isInsLike(c editdist.Command) bool {
	return c == editdist.Ins || c == editdist.Rpl
}

func rewriteText(ws []string, edt []editdist.Edit) string {
	reviseEditScript(edt)
	b := strings.Builder{}
	wi := 0
	preCmd := editdist.Unknown
	for _, e := range edt {
		if preCmd == editdist.Del && e.Cmd != editdist.Del && e.Cmd != editdist.Rpl {
			b.WriteString("~~")
		}
		if isInsLike(preCmd) && e.Cmd != editdist.Ins {
			b.WriteString("`")
		}
		switch e.Cmd {
		case editdist.Del:
			if preCmd == e.Cmd {
				addWord(&b, ws[wi])
			} else {
				addWord(&b, fmt.Sprintf("~~%s", ws[wi]))
			}
		case editdist.Ins:
			if isInsLike(preCmd) {
				addWord(&b, e.Word)
			} else {
				addWord(&b, fmt.Sprintf("`%s", e.Word))
			}
			preCmd = e.Cmd
			continue
		case editdist.Rpl:
			if preCmd == editdist.Del {
				addWord(&b, fmt.Sprintf("%s~~`%s", ws[wi], e.Word))
			} else {
				addWord(&b, fmt.Sprintf("~~%s~~`%s", ws[wi], e.Word))
			}
		case editdist.Ign:
			addWord(&b, ws[wi])
		}
		preCmd = e.Cmd
		wi++
	}
	return b.String()
}

func trim(ls *Lines) *Lines {
	if ls.start.next == ls.end {
		return &Lines{}
	}
	return &Lines{
		start: ls.start.next,
		end:   ls.end.prev,
	}
}

func connect(l1 *Line, l2 *Line) {
	l1.next = l2
	l2.prev = l1
}

func copyFile(path string, f *os.File) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(f, file)
	if err != nil {
		log.Fatal("Backup failed: ", err)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Please specify a filepath.")
		os.Exit(1)
	}
	file := os.Args[1]
	// get the text and position from the file
	ls := readFileAsLines(file)
	ps := locateTargets(ls)

	for _, p := range ps {
		// tokenize the text
		org := tokenize(trim(&p.original))
		rev := tokenize(trim(&p.revision))
		// calculate edit distance
		edt := editdist.WordBased(org, rev)
		// make output text from the EditScript
		orgRev := rewriteText(org, edt)
		orl := Line{
			text: &orgRev,
		}
		*p.original.start.text = ""
		*p.original.end.text = ""
		connect(p.original.start, &orl)
		connect(&orl, p.original.end)
		*p.revision.start.text = ""
		*p.revision.end.text = ""
	}

	// make a backup
	bk, err := os.CreateTemp("", "diffmd")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("backup: %s\n", bk.Name())
	defer os.Remove(bk.Name())

	copyFile(file, bk)

	// overwrite the file at the position of request text
	if err := os.WriteFile(file, []byte(ls.String()), 0666); err != nil {
		fmt.Printf("rollbacking...")
		_, ierr := exec.Command("cp", bk.Name(), file).Output()
		if ierr != nil {
			fmt.Println("Failed to rollback: ", ierr)
		}
		log.Fatal(err)
	}
}

/*
反省: 今回、行が持つ意味は薄いため、LinesではなくBlockとして扱うべきであった
TODO:
- outputの改行が消える問題を解消する
- エラーハンドリングちゃんとする
*/
