package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/wxio/gomuv"
)

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	tasks := Tasks{
		{"1", "client server sync"},
		{"2", "Implement Model Update View architecture"},
		{"3", "Tree diff"},
	}
	uimodel := make(map[string]interface{})
	er := json.Unmarshal([]byte(uimodeljson), &uimodel)
	if er != nil {
		fmt.Printf("decode error %v\n``%v``\n", er, string(uimodeljson))
		os.Exit(1)
	}
	templ := template.New("")
	templ = templ.Funcs(gomuv.GetFuncMap(r, templ, &tasks))
	templ, er = templ.Parse(gohtml)
	if er != nil {
		fmt.Printf("template err %v\n", er)
		os.Exit(1)
	}

	data := struct {
		UiElems map[string]interface{}
	}{
		UiElems: uimodel,
	}
	templ.Lookup("page").Execute(w, data)
}

type Tasks []Task

type Task struct {
	Id   string
	Name string
}

func (dm *Tasks) GetBinding(path ...string) (interface{}, error) {
	return gomuv.GetBinding(dm, path...)
}

const uimodeljson = `
{
	"UiElems" : [
		{
			"Template": "Tasks",
			"Bind": "MyTasks",
			"Title" : "My Tasks"
		}
	]
}
`

const gohtml = `
{{define "Tasks"}}
<div>
	<div class="header">{{.Title}}</div>
	{{range bind .Bind}}
		<div>{{.Name}}</div>
	{{end}}
</div>
{{end}}

{{define "page"}}
<!DOCTYPE html>
<meta charset="utf-8">
<style>
</style>
<body>
	<div>
	{{.}}
	{{range .UiElems}}
	{{.}}
	{{CallTemplate .Template .}}
	{{end}}
	</div>
</body>
{{end}}

`
