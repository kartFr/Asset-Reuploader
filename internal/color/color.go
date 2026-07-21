package color

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mattn/go-colorable"
)

const (
	FgRed = iota + 31
	FgGreen
	FgYellow
	FgBlue
	FgOrange = 33
)

var Output = colorable.NewColorableStdout()

var (
	Error   = New(FgRed)
	Warn    = New(FgYellow)
	Success = New(FgGreen)
	Info    = New(FgBlue)
	Orange  = New(FgOrange)
)

var reset = New(0)

type Color struct {
	params []int
}

func New(params ...int) *Color {
	c := &Color{
		params: make([]int, 0),
	}
	c.params = append(c.params, params...)

	return c
}

func (c *Color) paramsToString() string {
	strArray := make([]string, len(c.params))
	for i, param := range c.params {
		strArray[i] = strconv.Itoa(param)
	}
	return strings.Join(strArray, "")
}

func (c *Color) format() string {
	return fmt.Sprintf("\x1b[%sm", c.paramsToString())
}

func (c *Color) Fprint(w io.Writer, a ...any) (n int, err error) {
	f := fmt.Sprint(c.format(), fmt.Sprint(a...), reset.format())
	return fmt.Fprint(w, f)
}

func (c *Color) Fprintln(w io.Writer, a ...any) (n int, err error) {
	f := fmt.Sprint(c.format(), fmt.Sprint(a...), reset.format())
	return fmt.Fprintln(w, f)
}

func (c *Color) Print(a ...any) (n int, err error) {
	f := fmt.Sprint(c.format(), fmt.Sprint(a...), reset.format())
	return fmt.Fprint(Output, f)
}

func (c *Color) Println(a ...any) (n int, err error) {
	f := fmt.Sprint(c.format(), fmt.Sprint(a...), reset.format())
	return fmt.Fprintln(Output, f)
}
