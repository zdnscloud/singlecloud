package handler

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	chartYamlFile = "Chart.yaml"
	iconPrefix    = "/assets/helm/icons/"
	iconFormat    = ".png"
	indexFile     = "index.yaml"

	KeywordZcloudSystem = "zcloud-system"
)

var (
	responseHeaderTimeout = 60 * time.Second
	syncChartsInterval    = 60 * time.Second
	errNoFoundVersion     = fmt.Errorf("has no valid chart versions")
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
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
}

type ChartManager struct {
	chartDir string
}

func newChartManager(chartDir, repoUrl string) *ChartManager {
	go syncCloudChartsToLocal(repoUrl, chartDir)

	return &ChartManager{chartDir: chartDir}
}

func (m *ChartManager) List(ctx *resource.Context) interface{} {
	charts, err := getLocalCharts(m.chartDir, false)
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
	chartID := ctx.Resource.(*types.Chart).GetID()
	chart, err := getLocalChart(m.chartDir, chartID, false)
	if err != nil {
		log.Warnf("get chart %s failed:%s", chart.Name, err.Error())
		return nil
	}

	return chart
}

func getLocalChart(chartDir, chartName string, needSystemChart bool) (*types.Chart, error) {
	versions, description, err := listVersions(chartDir, chartName, needSystemChart)
	if err != nil {
		return nil, err
	}

	chart := &types.Chart{
		Name:        chartName,
		Description: description,
		Icon:        genChartIcon(chartName),
		Versions:    versions,
	}
	chart.SetID(chartName)
	return chart, nil
}

func getLocalCharts(chartDir string, needSystemChart bool) (types.Charts, error) {
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

			chart, err := getLocalChart(chartDir, cht.Name(), needSystemChart)
			if err == nil {
				charts = append(charts, chart)
			} else if err != errNoFoundVersion {
				log.Warnf("list charts info when get chart %s failed:%s", cht.Name(), err.Error())
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
		ResponseHeaderTimeout: responseHeaderTimeout,
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

	localCharts, err := getLocalCharts(chartDir, true)
	if err != nil {
		return fmt.Errorf("get local charts failed: %s", err.Error())
	}

	for chartName, chartEntries := range chartsIndex.Entries {
		chartFound, err := checkChartExistAndLoadVersionsIfNeed(repoUrl, chartDir, chartName, localCharts, chartEntries)
		if err != nil {
			return err
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

func checkChartExistAndLoadVersionsIfNeed(repoUrl, chartDir, chartName string, localCharts types.Charts, chartEntries []ChartEntry) (bool, error) {
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
						return chartFound, err
					}
				}
			}
			break
		}
	}

	return chartFound, nil
}

func loadCloudChartByVersion(versionUrls []string, repoUrl, chartDir, chartName, chartVersion string) error {
	if len(versionUrls) == 0 {
		return fmt.Errorf("invalid version urls in index.yaml of registry, it should not be empty")
	}

	chartUrl := repoUrl + "/" + versionUrls[0]
	resp, err := http.Get(chartUrl)
	if err != nil {
		return fmt.Errorf("load chart %s with url %s failed: %s", chartName, chartUrl, err.Error())
	}

	defer resp.Body.Close()
	unzipFileDir, err := expandFile(chartDir, chartName, resp.Body)
	if err != nil {
		return err
	}

	if err := os.Rename(path.Join(chartDir, chartName, unzipFileDir), path.Join(chartDir, chartName, chartVersion)); err != nil {
		return fmt.Errorf("rename chart %s from %s to %s failed: %s", chartName, unzipFileDir, chartVersion, err.Error())
	}

	log.Infof("load chart %s with version %s succeed", chartName, chartVersion)
	return nil
}

func listVersions(chartDir, chartName string, needSystemChart bool) ([]types.ChartVersion, string, error) {
	var description string
	chartPath := path.Join(chartDir, chartName)
	versionDirs, err := ioutil.ReadDir(chartPath)
	if err != nil {
		return nil, description, err
	}

	var versions []types.ChartVersion
	for _, versionDir := range versionDirs {
		if versionDir.IsDir() {
			if strings.HasPrefix(versionDir.Name(), ".") {
				continue
			}

			versionFullDir := path.Join(chartPath, versionDir.Name())
			if description == "" {
				if info, err := getChartInfo(versionFullDir); err != nil {
					log.Warnf("load chart with version %s info failed: %s", versionDir.Name(), err.Error())
					continue
				} else if needSystemChart == false && (slice.SliceIndex(info.Keywords, KeywordZcloudSystem) != -1) {
					break
				} else {
					description = info.Description
				}
			}

			config, err := charts.LoadChartConfigs(versionFullDir)
			if err != nil {
				log.Warnf("load chart with version %s config failed: %s", versionDir.Name(), err.Error())
				continue
			}

			versions = append(versions, types.ChartVersion{
				Version: versionDir.Name(),
				Config:  config,
			})
		} else if versionDir.Name() == chartYamlFile {
			return nil, description, fmt.Errorf("chart all files must be in a version dir")
		}
	}

	if len(versions) == 0 {
		return nil, description, errNoFoundVersion
	}

	return versions, description, nil
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

func expandFile(chartDir, chartName string, r io.Reader) (string, error) {
	chartFileDir := path.Join(chartDir, chartName)
	var unzipFileDir string
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return unzipFileDir, err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return unzipFileDir, err
		}

		headerName := "." + header.Name
		headerDir := filepath.Dir(headerName)
		if unzipFileDir == "" {
			unzipFileDir = headerDir
			os.RemoveAll(path.Join(chartFileDir, headerDir))
		}

		fullDir := filepath.Join(chartFileDir, headerDir)
		_, err = os.Stat(fullDir)
		if err != nil && headerDir != "" {
			if err := os.MkdirAll(fullDir, 0755); err != nil {
				return unzipFileDir, err
			}
		}

		chartFilePath := filepath.Clean(filepath.Join(chartFileDir, headerName))
		if filepath.Base(headerName) == chartName+iconFormat {
			chartFilePath = genChartIcon(chartName)
		}

		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(chartFilePath, info.Mode()); err != nil {
				return unzipFileDir, err
			}
			continue
		}

		chartFile, err := os.OpenFile(chartFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return unzipFileDir, err
		}

		if _, err = io.Copy(chartFile, tarReader); err != nil {
			chartFile.Close()
			return unzipFileDir, err
		}
		chartFile.Close()
	}

	return unzipFileDir, nil
}
