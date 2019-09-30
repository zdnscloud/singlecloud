package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	configPath    = "config/config.json"
	chartYamlFile = "Chart.yaml"
	IconPrefix    = "/assets/helm/icons/"
	IconFormat    = ".png"
)

type ChartInfo struct {
	Description string `yaml:"description"`
	SystemChart bool   `yaml:"systemChart"`
}

type ChartManager struct {
	chartDir string
}

func newChartManager(chartDir string) *ChartManager {
	return &ChartManager{chartDir: chartDir}
}

func (m *ChartManager) List(ctx *resource.Context) interface{} {
	chts, err := ioutil.ReadDir(m.chartDir)
	if err != nil {
		log.Warnf("list charts info failed:%s", err.Error())
		return nil
	}

	var charts types.Charts
	for _, cht := range chts {
		if cht.IsDir() {
			if strings.HasPrefix(cht.Name(), ".") {
				continue
			}

			versions, description, isSystemChart, err := listVersions(path.Join(m.chartDir, cht.Name()))
			if err != nil {
				log.Warnf("list charts info when get chart %s failed:%s", cht.Name(), err.Error())
				continue
			} else {
				if isSystemChart {
					continue
				}

				chart := &types.Chart{
					Name:        cht.Name(),
					Description: description,
					Icon:        genChartIcon(cht.Name()),
					Versions:    versions,
				}
				chart.SetID(chart.Name)
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

func (m *ChartManager) Get(ctx *resource.Context) resource.Resource {
	chart := ctx.Resource.(*types.Chart)
	versions, description, isSystemChart, err := listVersions(path.Join(m.chartDir, chart.GetID()))
	if err != nil {
		log.Warnf("get chart %s failed:%s", chart.Name, err.Error())
		return nil
	}

	if isSystemChart {
		log.Warnf("no found chart %s for user %s", chart.Name, getCurrentUser(ctx))
		return nil
	}

	chart.Name = chart.GetID()
	chart.Description = description
	chart.Icon = genChartIcon(chart.Name)
	chart.Versions = versions
	return chart
}

func listVersions(chartPath string) ([]types.ChartVersion, string, bool, error) {
	versionDirs, err := ioutil.ReadDir(chartPath)
	if err != nil {
		return nil, "", false, err
	}

	var versions []types.ChartVersion
	var chartInfo ChartInfo
	for _, versionDir := range versionDirs {
		if versionDir.IsDir() {
			if strings.HasPrefix(versionDir.Name(), ".") {
				continue
			}

			var config []map[string]interface{}
			content, err := ioutil.ReadFile(path.Join(chartPath, versionDir.Name(), configPath))
			if err == nil {
				if err := json.Unmarshal(content, &config); err != nil {
					return nil, "", false, fmt.Errorf("unmarshal config file failed: %s", err.Error())
				}
			}

			if chartInfo.Description == "" {
				if info, err := getChartInfo(path.Join(chartPath, versionDir.Name())); err != nil {
					return nil, "", false, err
				} else {
					chartInfo = *info
				}
			}

			versions = append(versions, types.ChartVersion{
				Version: versionDir.Name(),
				Config:  config,
			})
		} else if versionDir.Name() == chartYamlFile {
			return nil, "", false, fmt.Errorf("chart all files must be in a version dir")
		}
	}

	return versions, chartInfo.Description, chartInfo.SystemChart, nil
}

func getChartInfo(chartYamlPath string) (*ChartInfo, error) {
	var info ChartInfo
	chartYaml, err := ioutil.ReadFile(path.Join(chartYamlPath, chartYamlFile))
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(chartYaml, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

func genChartIcon(chartName string) string {
	return IconPrefix + chartName + IconFormat
}
