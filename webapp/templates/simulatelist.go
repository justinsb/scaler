package templates

import (
	"github.com/justinsb/scaler/pkg/simulate"
	"fmt"
	"html/template"
	"bytes"
)

var simulateListTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
</head>
<body>
<ul>
	{{range .Simulations}}<li><a href="./{{.Key}}">{{ .Key }}</a></li>{{end}}
</ul>
</body>
</html>
`

type simulateListData struct {
	Simulations []*simulate.Metadata
}

func BuildSimulateListPage(simulations []*simulate.Metadata) ([]byte, error) {
	tmpl, err := template.New("simulatelist").Parse(simulateListTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing simulatelist template: %v", err)
	}

	data := &simulateListData{
		Simulations: simulations,
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("error executing simulatelist template: %v", err)
	}

	return b.Bytes(), nil
}
