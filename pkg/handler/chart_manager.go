package handler

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"helm.sh/helm/pkg/chartutil"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	configPath    = "config/config.json"
	chartYamlFile = "Chart.yaml"
	iconPrefix    = "/assets/helm/icons/"
	iconFormat    = ".png"
	indexFile     = "index.yaml"
)

var (
	timeout            = 60 * time.Second
	syncChartsInterval = 60 * time.Second
)

type ChartsIndex struct {
	Entries map[string][]ChartEntry `yaml:"entries"`
}

type ChartEntry struct {
	Name    string   `yaml:"name"`
	Version string   `yaml:"version"`
	Urls    []string `yaml:"urls"`
}

type ChartInfo struct {
	Description string `yaml:"description"`
	SystemChart bool   `yaml:"systemChart"`
}

type ChartManager struct {
	chartDir string
}

func newChartManager(chartDir, repoUrl string) *ChartManager {
	go syncCloudChartsToLocal(repoUrl, chartDir)

	return &ChartManager{chartDir: chartDir}
}

func (m *ChartManager) List(ctx *resource.Context) interface{} {
	charts, err := getLocalCharts(m.chartDir)
	if err != nil {
		log.Warnf("list charts info failed:%s", err.Error())
		return nil
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

func getLocalCharts(chartDir string) (types.Charts, error) {
	chts, err := ioutil.ReadDir(chartDir)
	if err != nil {
		return nil, err
	}

	var charts types.Charts
	for _, cht := range chts {
		if cht.IsDir() {
			if strings.HasPrefix(cht.Name(), ".") {
				continue
			}

			versions, description, isSystemChart, err := listVersions(path.Join(chartDir, cht.Name()))
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

	return charts, nil
}

func syncCloudChartsToLocal(repoUrl, chartDir string) {
	if repoUrl == "" {
		return
	}

	http.DefaultClient.Transport = &http.Transport{
		ResponseHeaderTimeout: timeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	for {
		if err := loadCloudCharts(repoUrl, chartDir); err != nil {
			log.Warnf("load cloud charts failed: %s", err.Error())
		}
		time.Sleep(syncChartsInterval)
	}
}

func loadCloudCharts(repoUrl, chartDir string) error {
	chartsIndex, err := getCloudChartsIndex(repoUrl)
	if err != nil {
		return err
	}

	localCharts, err := getLocalCharts(chartDir)
	if err != nil {
		return fmt.Errorf("get local charts failed: %s", err.Error())
	}

	for chartName, chartEntries := range chartsIndex.Entries {
		chartFound := false
		for _, chart := range localCharts {
			if chart.Name == chartName {
				chartFound = true
				for _, chartEntry := range chartEntries {
					versionFound := false
					for _, chartVersion := range chart.Versions {
						if chartVersion.Version == chartEntry.Version {
							versionFound = true
							break
						}
					}
					if !versionFound {
						log.Infof("found chart %s new version %s in registry, will load it", chartName, chartEntry.Version)
						err := loadCloudChartByVersion(chartEntry.Urls, repoUrl, chartDir, chartName, chartEntry.Version)
						if err != nil {
							return err
						}
					}
				}
				break
			}
		}

		if !chartFound {
			log.Infof("found new chart %s in registry, will load it", chartName)
			if err := os.MkdirAll(path.Join(chartDir, chartName), 0755); err != nil {
				return err
			}

			for _, chartEntry := range chartEntries {
				if err := loadCloudChartByVersion(chartEntry.Urls, repoUrl, chartDir, chartName, chartEntry.Version); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getCloudChartsIndex(repoUrl string) (*ChartsIndex, error) {
	resp, err := http.Get(repoUrl + "/" + indexFile)
	if err != nil {
		return nil, fmt.Errorf("get charts index failed: %s", err.Error())
	}

	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read charts index failed: %s", err.Error())
	}

	var index ChartsIndex
	if err := yaml.Unmarshal(content, &index); err != nil {
		return nil, fmt.Errorf("unmarshal charts index failed: %s", err.Error())
	}

	return &index, nil
}

func loadCloudChartByVersion(versionUrls []string, repoUrl, chartDir, chartName, chartVersion string) error {
	if len(versionUrls) == 0 {
		return fmt.Errorf("invalid version urls, it should not be empty")
	}

	chartUrl := repoUrl + "/" + versionUrls[0]
	resp, err := http.Get(chartUrl)
	if err != nil {
		return fmt.Errorf("load chart %s with url %s failed: %s", chartName, chartUrl, err.Error())
	}
	defer resp.Body.Close()

	tmpdir, err := ioutil.TempDir("", "helm-")
	if err != nil {
		return fmt.Errorf("gen temp dir failed: %s", err.Error())
	}
	defer os.RemoveAll(tmpdir)

	err = chartutil.Expand(tmpdir, resp.Body)
	if err != nil {
		panic("expand redis failed: " + err.Error())
	}

	versionDir := path.Join(chartDir, chartName, chartVersion)
	err = os.Rename(path.Join(tmpdir, chartName), versionDir)
	if err != nil {
		return fmt.Errorf("move chart %s from temp dir to %s failed: %s", chartName, versionDir, err.Error())
	}

	log.Infof("load chart %s with version %s succeed", chartName, chartVersion)
	return nil
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
	return iconPrefix + chartName + iconFormat
}
