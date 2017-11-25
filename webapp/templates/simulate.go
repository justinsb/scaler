package templates

import (
	"github.com/justinsb/scaler/pkg/graph"
	"encoding/json"
	"fmt"
	"html/template"
	"bytes"
	"github.com/justinsb/scaler/pkg/simulate"
)

var simulateTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/nvd3/1.8.6/nv.d3.css" rel="stylesheet" type="text/css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/d3/3.5.17/d3.min.js" charset="utf-8"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/nvd3/1.8.6/nv.d3.js"></script>

    <style>
        text {
            font: 12px sans-serif;
        }
        svg {
            display: block;
        }
        html, body, #chart1, svg {
            margin: 0px;
            padding: 0px;
            height: 90%;
            width: 100%;
        }

        .dashed {
            stroke-dasharray: 5,5;
        }
    </style>
</head>
<body class='with-3d-shadow with-transitions'>
<div id="info" >UpdateCount: {{.Run.UpdateCount}}</div>
<div id="chart1"></div>

<script>
  // Wrapping in nv.addSimulate allows for '0 timeout render', stores rendered charts in nv.graphs, and may do more in the future... it's NOT required
  var chart;

  nv.addGraph(function() {
    chart = nv.models.lineChart()
      .options({
        duration: 0,
        useInteractiveGuideline: true
      })
    ;

	chart.legendPosition("bottom");

    // chart sub-models (ie. xAxis, yAxis, etc) when accessed directly, return themselves, not the parent chart, so need to chain separately
    chart.xAxis
      .axisLabel({{.Graph.XAxis.Label}})
      .tickFormat(d3.format(',.1f'))
      .staggerLabels(false)
    ;

    chart.yAxis
      .axisLabel({{.Graph.YAxis.Label}})
      .tickFormat(function(d) {
        if (d == null) {
          return 'N/A';
        }
        return d3.format(',.2f')(d);
      })
    ;

    var data = {{.SeriesJson}};

    d3.select('#chart1').append('svg')
      .datum(data)
      .call(chart);

    nv.utils.windowResize(chart.update);

    return chart;
  });
</script>
</body>
</html>
`

type simulateData struct {
	SeriesJson template.JS
	Graph      *graph.Model
	Run *simulate.Run
}

func BuildSimulatePage(run *simulate.Run) ([]byte, error) {
	seriesJson, err := json.Marshal(run.Graph.Series)
	if err != nil {
		return nil, fmt.Errorf("error building json for simulate page: %v", err)
	}

	tmpl, err := template.New("simulate").Parse(simulateTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing simulate template: %v", err)
	}

	data := &simulateData{
		SeriesJson: template.JS(seriesJson),
		Graph:      run.Graph,
		Run: run,
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, data); err != nil {
		return nil, fmt.Errorf("error executing simulate template: %v", err)
	}

	return b.Bytes(), nil
}
