package csv4g

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

type Csv4g struct {
	name       string
	fields     []reflect.StructField
	lines      [][]string
	lineNo     int
	LineLen    int
	lineOffset int
}

func New(filePath string, comma rune, o interface{}, skipLine int) (*Csv4g, error) {
	file, openErr := os.Open(filePath)
	if openErr != nil {
		return nil, fmt.Errorf("%s open file error %v", file.Name, openErr)
	}
	defer file.Close()
	r := csv.NewReader(file)
	r.Comma = comma
	var err error
	var fields []string
	fields, err = r.Read() // first line is field's description
	if err != nil {
		return nil, fmt.Errorf("%s read first line error %v", file.Name, err)
	}
	offset := skipLine
	for skipLine > 0 {
		skipLine--
		_, err = r.Read()
		if err != nil {
			return nil, fmt.Errorf("%s skip line error %v", file.Name, err)
		}
	}

	tType := reflect.TypeOf(o)
	if tType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("interface must be a struct")
	}
	ret := &Csv4g{
		name:       file.Name(),
		fields:     make([]reflect.StructField, tType.NumField()),
		lineNo:     0,
		lineOffset: offset + 1}

	for i := 0; i < tType.NumField(); i++ {
		f := tType.Field(i)
		ret.fields[i] = f
		isOk := false
		for j, _ := range fields {
			if fields[j] == f.Name {
				isOk = true
				break
			}
		}
		if !isOk {
			return nil, fmt.Errorf("%s cannot find field %s", file.Name(), f.Name)
		}
	}

	var lines [][]string
	lines, err = r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("Read error %v", err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%s has no data!", file.Name())
	}
	ret.lines = lines
	ret.LineLen = len(lines)
	return ret, nil
}

func (this *Csv4g) Parse(obj interface{}) (err error) {
	if this.lineNo >= len(this.lines) {
		return io.EOF
	}
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%s error on parse line %d", this.name, this.lineNo+this.lineOffset+1)
			return
		}
		this.lineNo++
	}()
	values := this.lines[this.lineNo]
	elem := reflect.ValueOf(obj).Elem()
	for index, _ := range this.fields {
		f := elem.FieldByIndex(this.fields[index].Index)
		if conv, ok := converters[f.Kind()]; ok {
			v := conv(values[index])
			f.Set(v)
		} else {
			if f.Kind() == reflect.Slice {
				if sliceConv, ok := sliceConvertes[f.Type()]; ok {
					f.Set(sliceConv(strings.Split(values[index], "|")))
				} else {
					err = fmt.Errorf("%s:[%d] unsupported field set %v -> %v :[%d].",
						this.name, this.lineNo+this.lineOffset, this.fields[index], values[index])
				}
			} else {
				err = fmt.Errorf("%s:[%d] unsupported field set %v -> %v :[%d].",
					this.name, this.lineNo+this.lineOffset, this.fields[index], values[index])
			}
		}
	}
	return
}
