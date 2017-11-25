package templates

import (
	"github.com/justinsb/scaler/pkg/graph"
	"fmt"
	"html/template"
	"bytes"
)

var graphListTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
</head>
<body>
<ul>
	{{range .Graphs}}<li><a href="./{{.Key}}">{{ .Key }}</a></li>{{end}}
</ul>
</body>
</html>
`

type graphListData struct {
	Graphs []*graph.Metadata
}

func BuildGraphListPage(graphs []*graph.Metadata) ([]byte, error) {
	tmpl, err := template.New("graphlist").Parse(graphListTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing graphlist template: %v", err)
	}

	data := &graphListData{
		Graphs: graphs,
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("error executing graphlist template: %v", err)
	}

	return b.Bytes(), nil
}
