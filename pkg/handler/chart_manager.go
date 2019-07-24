package handler

import (
	"fmt"
	"io/ioutil"
	"path"
	"sort"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	configPath = "config/config.json"
)

type ChartManager struct {
	api.DefaultHandler
	chartDir string
}

func newChartManager(chartDir string) *ChartManager {
	return &ChartManager{chartDir: chartDir}
}

func (m *ChartManager) List(ctx *resttypes.Context) interface{} {
	chts, err := ioutil.ReadDir(m.chartDir)
	if err != nil {
		log.Warnf("list charts info failed:%s", err.Error())
		return nil
	}

	var charts types.Charts
	for _, cht := range chts {
		if cht.IsDir() {
			versions, err := listVersions(path.Join(m.chartDir, cht.Name()))
			if err != nil {
				log.Warnf("list charts info failed:%s", err.Error())
				return nil
			} else {
				chart := &types.Chart{
					Name:     cht.Name(),
					Versions: versions,
				}
				chart.SetID(chart.Name)
				chart.SetType(types.ChartType)
				charts = append(charts, chart)
			}
		}
	}

	if len(charts) == 0 {
		log.Warnf("no found valid chart in dir %s", m.chartDir)
		return nil
	}

	sort.Sort(charts)
	return charts
}

func (m *ChartManager) Get(ctx *resttypes.Context) interface{} {
	chart := ctx.Object.(*types.Chart)
	versions, err := listVersions(path.Join(m.chartDir, chart.GetID()))
	if err != nil {
		log.Warnf("get chart %s failed:%s", chart.Name, err.Error())
		return nil
	}

	chart.Versions = versions
	chart.SetType(types.ChartType)
	return chart
}

func listVersions(chartPath string) ([]types.ChartVersion, error) {
	versionDirs, err := ioutil.ReadDir(chartPath)
	if err != nil {
		return nil, err
	}

	var versions []types.ChartVersion
	for _, versionDir := range versionDirs {
		if versionDir.IsDir() {
			config := ""
			content, err := ioutil.ReadFile(path.Join(chartPath, versionDir.Name(), configPath))
			if err == nil {
				config = string(content)
			}
			versions = append(versions, types.ChartVersion{
				Version: versionDir.Name(),
				Config:  config,
			})
		} else if versionDir.Name() == "Chart.yaml" {
			return nil, fmt.Errorf("chart all files must be in a version dir")
		}
	}

	return versions, nil
}