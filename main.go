package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/gdamore/tcell"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

var CODE = "v ;11101110;\n>>>>>>>>>>>>>>>a0g68*-90g68*-2*+80g68*-4*+70g68*-8*+v\n@.+***288-*86g03+**88-*86g04+**84-*86g05+**44-*86g06<"

var err error

func main() {
	e := flag.Bool("e",false,"run edit mode")
	t := flag.Int("t",1000,"timeout in microseconds")
	flag.Parse()
	file := flag.Arg(0)
	if file!=""{
		code, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(err)
			return
		}
		CODE = string(code)
	}
	field := field{withoutScreen: !*e}
	field.pointer.vector = 2
	field.changeCode(CODE)

	if !field.withoutScreen{
		field.screen, err = tcell.NewScreen()
		if err != nil {
			fmt.Println(err)
			return
		}
		err = field.screen.Init()
		if err != nil {
			fmt.Println(err)
			return
		}
		field.scaleModify = 1
		field.updScreen()
		defer field.screen.Fini()
	}else {
		for{
			field.do()
			time.Sleep(time.Microsecond*time.Duration(*t))
		}
	}



	for {
		event := field.screen.PollEvent()
		switch eventType := event.(type) {
		case *tcell.EventKey:

			switch eventType.Key() {
			case tcell.KeyCtrlS:
				if file!=""{
					CODE = ""
					for _, cur := range field.field {
						for _, b := range cur {
							CODE+=string(b)
						}
						CODE+="\n"
					}
					ioutil.WriteFile(file,[]byte(CODE),600)
				}
			case tcell.KeyCtrlC:
				return
			case tcell.KeyUp:
				if field.pointer.y > 0 {
					field.pointer.y--
					field.updScreen()
				}
			case tcell.KeyDown:
				if field.pointer.y < len(field.field)-1 {
					field.pointer.y++
					field.updScreen()
				}
			case tcell.KeyLeft:
				if field.pointer.x > 0 {
					field.pointer.x--
					field.updScreen()
				}
			case tcell.KeyRight:
				if field.pointer.x < len(field.field[field.pointer.y])-1 {
					field.pointer.x++
					field.updScreen()
				}
			case tcell.KeyCtrlB:
				field.printInfo = !field.printInfo
				field.updScreen()
			default:
				if eventType.Rune() != 0 {
					if eventType.Rune() == ' ' {
						field.do()
						continue
					}
					field.changePointerCell(byte(eventType.Rune()))
					field.updScreen()
				}
			}
		}
	}
}

type field struct {
	screen        tcell.Screen
	pointer       pointer
	field         [][]byte
	stack         []int
	stackSF       bool
	scaleModify   int
	toPrint       string
	printInfo     bool
	printLine     int
	printMove     int
	withoutScreen bool
}
type pointer struct {
	x      int
	y      int
	vector uint8
}

func (f *field) changePointerCell(newValue byte) {
	f.field[f.pointer.y][f.pointer.x] = newValue
}

func (f *field) changeCode(code string) {
	splitted := bytes.Split([]byte(code), []byte("\n"))

	yMax := len(splitted) // Определение размеров поля
	xMax := len(splitted[0])
	for _, bs := range splitted {
		if len(bs) > xMax {
			xMax = len(bs)
		}
	}

	f.field = make([][]uint8, yMax+1) // Пересоздание поля с нужными размерами
	for i := range f.field {
		f.field[i] = make([]uint8, xMax+1)
	}
	var i int
	for y, bs := range splitted {
		for x, b := range bs {
			i++
			f.field[y][x] = b
		}
	}
}

func (f *field) updScreen() {
	//Командное поле
	for y, row := range f.field {
		for x, cell := range row {
			if cell != 0 {
				f.screen.SetCell(x*f.scaleModify, y*f.scaleModify, tcell.StyleDefault, rune(cell))
			}
			if f.pointer.x == (x) && f.pointer.y == (y) {
				f.screen.ShowCursor(x*f.scaleModify, y*f.scaleModify)
			}
		}
	}

	//Информация об указателе
	if f.printInfo {
		startX := len(f.field[0])
		maxLen := 1
		if len(f.field) > maxLen {
			maxLen = len(f.field)
		}
		if len(f.field[0]) > maxLen {
			maxLen = len(f.field[0])
		}
		maxLen = len(strconv.Itoa(maxLen))
		//print x
		f.screen.SetCell(startX+3, 0, tcell.StyleDefault, 'x')
		f.screen.SetCell(startX+4, 0, tcell.StyleDefault, ':')
		for i, r := range []rune(fmt.Sprintf("%"+strconv.Itoa(maxLen)+"d", f.pointer.x)) {
			f.screen.SetCell(startX+5+i, 0, tcell.StyleDefault, r)
		}
		//print y
		f.screen.SetCell(startX+3, 1, tcell.StyleDefault, 'y')
		f.screen.SetCell(startX+4, 1, tcell.StyleDefault, ':')
		for i, r := range []rune(fmt.Sprintf("%"+strconv.Itoa(maxLen)+"d", f.pointer.y)) {
			f.screen.SetCell(startX+5+i, 1, tcell.StyleDefault, r)
		}
		//print vector
		switch f.pointer.vector {
		case 1:
			f.screen.SetCell(startX+3, 3, tcell.StyleDefault, '^')
		case 2:
			f.screen.SetCell(startX+3, 3, tcell.StyleDefault, '>')
		case 3:
			f.screen.SetCell(startX+3, 3, tcell.StyleDefault, 'v')
		case 4:
			f.screen.SetCell(startX+3, 3, tcell.StyleDefault, '<')
		}
		if f.stackSF {
			f.screen.SetCell(startX+3, 4, tcell.StyleDefault, '1')
		} else {
			f.screen.SetCell(startX+3, 4, tcell.StyleDefault, '0')
		}
		//print stack
		for i, u := range f.stack {
			for n, r := range []rune(fmt.Sprintf("%12d",u)) {
				f.screen.SetCell(startX+7+maxLen+n, len(f.stack)-i-1, tcell.StyleDefault, r)
			}
		}

		for i := 0; ; i++ {
			curRune1, _, _, _ := f.screen.GetContent(startX+7+maxLen+11, len(f.stack)+i)
			if curRune1 == ' ' {
				break
			}
			for n, r := range []rune("            ") {
				f.screen.SetCell(startX+7+maxLen+n, len(f.stack)+i, tcell.StyleDefault, r)
			}
		}
	}

	//print toPrint
	toPrintArr := strings.Split(f.toPrint, "\n")
	for n, s := range toPrintArr {
		for i, r := range []rune(s) {
			f.screen.SetCell(i, len(f.field)+n, tcell.StyleDefault, r)
		}
	}

	f.screen.Sync()
}

func (f *field) step() {
	switch f.pointer.vector {
	case 1:
		if f.pointer.y>0{
			f.pointer.y -= 1
		}else {
			f.pointer.y = len(f.field)-1
		}
	case 2:
		if f.pointer.x<len(f.field[0])-1{
			f.pointer.x += 1
		}else {
			f.pointer.x = 0
		}
	case 3:
		if f.pointer.y<len(f.field)-1{
			f.pointer.y += 1
		}else {
			f.pointer.y = 0
		}
	case 4:
		if f.pointer.x>0{
			f.pointer.x -= 1
		}else {
			f.pointer.x = len(f.field[0])-1
		}
	}
	if !f.withoutScreen{
		f.updScreen()
	}
}
func (f *field) do() {
	defer f.step()
	value := int(f.field[f.pointer.y][f.pointer.x])

	if f.stackSF && value != '"' {
		f.addStack(value)
		return
	}
	switch value {
	case '"':
		f.stackSF = !f.stackSF
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		i, err := strconv.Atoi(string(value))
		if err == nil {
			f.addStack(i)
		}
	case '+', '-', '*', '/', '%':
		for len(f.stack)<2{
			f.addStack(0)
		}
		a := f.popStack()
		b := f.popStack()
		switch value {
		case '+':
			f.addStack(a + b)
		case '-':
			f.addStack(b - a)
		case '*':
			f.addStack(a * b)
		case '/':
			if b == 0 {
				f.addStack(0)
			} else {
				f.addStack(a / b)
			}
		case '%':
			f.addStack(a % b)
		}
	case '!':
		for len(f.stack)<1{
			f.addStack(0)
		}
		a := f.popStack()
		if a == 0 {
			f.addStack(1)
		} else {
			f.addStack(0)
		}
	case '`':
		for len(f.stack)<2{
			f.addStack(0)
		}
		a := f.popStack()
		b := f.popStack()
		if b > a {
			f.addStack(1)
		} else {
			f.addStack(0)
		}
	case '?':
		rand.Seed(time.Now().UnixNano())
		f.pointer.vector = uint8(rand.Intn(4)+1)
	case ':':
		if len(f.stack)==0{
			f.addStack(0)
		}
		a := f.popStack()
		f.addStack(a)
		f.addStack(a)
	case '\\':
		if len(f.stack)<1{
			f.addStack(0)
		}
		if len(f.stack)<2{
			f.addStack(0)
			break
		}
		a := f.popStack()
		b := f.popStack()
		f.addStack(a)
		f.addStack(b)
	case '@':
		if f.withoutScreen{
			fmt.Println()
			os.Exit(0)
		}else {
			return
		}
	case '#':
		f.step()
	case ',':
		f.print(string(f.popStack()))
	case '.':
		f.print(strconv.Itoa(int(f.popStack())))
	case '_':
		if f.popStack() == 0 {
			f.pointer.vector = 2
		} else {
			f.pointer.vector = 4
		}
	case '|':
		if f.popStack() == 0 {
			f.pointer.vector = 3
		} else {
			f.pointer.vector = 1
		}
	case '$':
		f.popStack()
	case '~':
		var add rune
		fmt.Scanln(&add)
		f.addStack(int(add))
	case '&':
		var add uint8
		fmt.Scanln(&add)
		f.addStack(int(add))
	case 'p':
		x := f.popStack()
		y := f.popStack()
		sym := f.popStack()
		f.field[x][y] = byte(sym)
	case 'g':
		x := f.popStack()
		y := f.popStack()
		f.addStack(int(f.field[x][y]))
	case '>':
		f.pointer.vector = 2
	case '<':
		f.pointer.vector = 4
	case '^':
		f.pointer.vector = 1
	case 'v':
		f.pointer.vector = 3
	}
}

func (f *field) popStack() (last int) {
	if len(f.stack)==0{
		return 0
	}
	last = f.stack[len(f.stack)-1]
	f.stack = f.stack[:len(f.stack)-1]
	return
}
func (f *field) addStack(add int) {
	f.stack = append(f.stack, add)
}
func (f *field) print(text string) {
	if f.withoutScreen {
		fmt.Print(text)
	} else {
		f.toPrint += " " + text
	}
}
