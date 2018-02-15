package gomuv

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/golang/glog"
)

type DataModel interface {
	GetBinding(path ...string) (interface{}, error)
}

func GetFuncMap(req *http.Request, templ *template.Template, bindModel DataModel) template.FuncMap {
	return template.FuncMap{
		"CallTemplate": func(name string, data interface{}) (ret template.HTML, err error) {
			buf := bytes.NewBuffer([]byte{})
			err = templ.ExecuteTemplate(buf, name, data)
			if err != nil {
				fmt.Printf("CallTemplate Error %v\n", err)
				limitedStackTrace()
			}
			ret = template.HTML(buf.String())
			return
		},
		"bind": func(name ...string) (ret interface{}) {
			var path []string
			if len(name) == 1 {
				path = strings.Split(name[0], ".")
			} else {
				path = name
			}
			defer func() {
				if re := recover(); re != nil {
					fmt.Printf("bind error looking for '%s' so far %+v err %v\n", name, ret, re)
					limitedStackTrace()
					ret = nil
				}
			}()
			ret, err := bindModel.GetBinding(path...)
			if err != nil {
				// todo user provided logger
				fmt.Printf("err getbinding %v", err)
				return nil
			}
			return
		},
	}
}

type GenericDM map[string]interface{}

func (dm *GenericDM) GetBinding(path ...string) (interface{}, error) {
	p1, exists := (*dm)[path[0]]
	if !exists {
		glog.Warningf("couldn't find %v", path[0])
		return nil, nil
	}
	for i := 1; i < len(path); i++ {
		pX, ok := p1.(map[interface{}]interface{})
		if !ok {
			glog.Warningf("map expected pX is a %T at %d %v", p1, i, path)
			return nil, nil
		}
		p1, exists = pX[path[i]]
		if !exists {
			glog.Warningf("couldn't find %s", path[i])
			return nil, nil
		}
	}
	return p1, nil
}

func GetBinding(dm DataModel, path ...string) (interface{}, error) {
	val := reflect.ValueOf(dm)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	switch val.Kind() {
	case reflect.Map:
		val = val.MapIndex(reflect.ValueOf(path[0]))
	default:
		if !val.IsValid() {
			glog.Warningf("invalid %s %v %s", path[0], val, path)
			return nil, nil
		}
		val = val.FieldByName(path[0])
	}

	for i := 1; i < len(path); i++ {
		switch val.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
			if val.IsNil() {
				glog.Warningf("nil returned for %s in %s", path[0], path)
				return nil, fmt.Errorf("nil returned for %s in %s", path[0], path)
			}
		}
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		switch val.Kind() {
		case reflect.Map:
			val = val.MapIndex(reflect.ValueOf(path[i]))
		default:
			if !val.IsValid() {
				glog.Warningf("invalid %s %v %s", path[i], val, path)
				return nil, nil
			}
			val = val.FieldByName(path[i])
		}
		switch val.Kind() {
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
			if val.IsNil() {
				glog.Warningf("nil returned for %s in %s", path[i], path)
				return nil, fmt.Errorf("nil returned for %s in %s", path[i], path)
			}
		}
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
	}
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		if val.IsNil() {
			glog.Warningf("nil returned for %s %v", path, dm)
			return nil, nil
		}
	}
	if !val.IsValid() {
		glog.Warningf("invalid returned for %s %v", path, dm)
		return nil, nil
	}
	// holly crap, this works!!! ie ret.Interface()
	// It is unexpected as the following Go code isn't valid
	//  for _, x := range val.Interface()
	// but the template works, ie
	// {{ range bind .Name }} where bind returns val.Interface()
	return val.Interface(), nil
}

func limitedStackTrace() {
	pc := make([]uintptr, 100)
	n := runtime.Callers(0, pc)
	pc = pc[:n] // pass only valid pcs to runtime.CallersFrames
	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if strings.Contains(frame.File, "/template/") {
			continue
		}
		if strings.Contains(frame.File, "/runtime/") {
			continue
		}
		if strings.Contains(frame.File, "/reflect/") {
			continue
		}
		if strings.Contains(frame.File, "/http/") {
			continue
		}
		if !more {
			break
		}
		fmt.Printf("%s:%d (%s)\n", frame.File, frame.Line, frame.Function)
	}
}
