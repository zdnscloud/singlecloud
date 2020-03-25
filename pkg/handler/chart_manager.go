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
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	chartYamlFile       = "Chart.yaml"
	iconPrefixForReturn = "/assets/helm/icons/"
	iconPrefixForLoad   = "/helm-icons/"
	iconFormat          = ".png"
	indexFile           = "index.yaml"

	KeywordZcloudSystem = "zcloud-system"
	ZcloudChartDir      = "zcloud"
	ZcloudChartFilter   = "is_zcloud_chart"
	UserChartDir        = "user"
	UserChartFilter     = "is_user_chart"
)

var (
	responseHeaderTimeout = 60 * time.Second
	syncChartsInterval    = 60 * time.Second

	ErrNoFoundVersion = fmt.Errorf("no found valid chart versions")
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
	go syncCloudChartsToLocal(repoUrl, path.Join(chartDir, ZcloudChartDir))
	return &ChartManager{chartDir: chartDir}
}

func (m *ChartManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	charts, err := getCharts(m.chartDir, ctx.GetFilters(), false)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list charts info failed:%s", err.Error()))
	}

	if len(charts) == 0 {
		log.Warnf("no found valid chart in dir %s", m.chartDir)
		return nil, nil
	}

	sort.Sort(charts)
	return charts, nil
}

func getCharts(chartDir string, filters []resource.Filter, needSystemChart bool) (types.Charts, error) {
	var charts types.Charts
	for _, dir := range getChartDirs(chartDir, filters) {
		chts, err := getLocalCharts(dir, needSystemChart)
		if err != nil {
			return nil, err
		}

		charts = append(charts, chts...)
	}

	return charts, nil
}

func getChartDirs(chartDir string, filters []resource.Filter) []string {
	if chartDir, ok := getChartDir(chartDir, filters); ok {
		return []string{chartDir}
	}

	return []string{path.Join(chartDir, ZcloudChartDir), path.Join(chartDir, UserChartDir)}
}

func getChartDir(chartDir string, filters []resource.Filter) (string, bool) {
	for _, filter := range filters {
		if hasChartFilter(filter, ZcloudChartFilter) {
			return path.Join(chartDir, ZcloudChartDir), true
		}

		if hasChartFilter(filter, UserChartFilter) {
			return path.Join(chartDir, UserChartDir), true
		}
	}

	return "", false
}

func hasChartFilter(filter resource.Filter, filterName string) bool {
	return filter.Name == filterName && filter.Modifier == resource.Eq && slice.SliceIndex(filter.Values, "true") != -1
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
			} else if err != ErrNoFoundVersion {
				log.Debugf("get chart %s failed:%s", cht.Name(), err.Error())
			}
		}
	}

	return charts, nil
}

func (m *ChartManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	chartID := ctx.Resource.(*types.Chart).GetID()
	chart, err := getChart(m.chartDir, chartID, false)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get chart %s failed:%s", chartID, err.Error()))
	}

	return chart, nil
}

func getChart(chartDir, chartName string, needSystemChart bool) (*types.Chart, error) {
	chart, err := getLocalChart(path.Join(chartDir, ZcloudChartDir), chartName, needSystemChart)
	if err != nil {
		if isNoSuchFileOrDirError(err) == false {
			return nil, err
		}

		chart, err = getLocalChart(path.Join(chartDir, UserChartDir), chartName, needSystemChart)
		if err != nil {
			if isNoSuchFileOrDirError(err) {
				return nil, fmt.Errorf("no found valid chart file")
			}
			return nil, err
		}
	}

	return chart, nil
}

func isNoSuchFileOrDirError(err error) bool {
	return strings.Contains(err.Error(), "no such file or directory")
}

func getLocalChart(chartDir, chartName string, needSystemChart bool) (*types.Chart, error) {
	versions, description, err := listVersions(chartDir, chartName, needSystemChart)
	if err != nil {
		return nil, err
	}

	chart := &types.Chart{
		Name:        chartName,
		Description: description,
		Icon:        genChartIcon(iconPrefixForReturn, chartName),
		Dir:         chartDir,
		Versions:    versions,
	}
	chart.SetID(chartName)
	return chart, nil
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
		return nil, description, ErrNoFoundVersion
	}

	return versions, description, nil
}

func syncCloudChartsToLocal(repoUrl, zcloudChartDir string) {
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
		if err := loadCloudCharts(repoUrl, zcloudChartDir); err != nil {
			log.Warnf("load cloud charts failed: %s", err.Error())
		}
		time.Sleep(syncChartsInterval)
	}
}

func loadCloudCharts(repoUrl, zcloudChartDir string) error {
	chartsIndex, err := getCloudChartsIndex(repoUrl)
	if err != nil {
		return err
	}

	localCharts, err := getLocalCharts(zcloudChartDir, true)
	if err != nil {
		return fmt.Errorf("get local charts failed: %s", err.Error())
	}

	for chartName, chartEntries := range chartsIndex.Entries {
		chartFound, err := checkChartExistAndLoadVersionsIfNeed(repoUrl, zcloudChartDir, chartName, localCharts, chartEntries)
		if err != nil {
			return err
		}

		if !chartFound {
			log.Infof("found new chart %s in registry, will load it", chartName)
			if err := os.MkdirAll(path.Join(zcloudChartDir, chartName), 0755); err != nil {
				return err
			}

			for _, chartEntry := range chartEntries {
				if err := loadCloudChartByVersion(chartEntry.Urls, repoUrl, zcloudChartDir, chartName, chartEntry.Version); err != nil {
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

func genChartIcon(iconPrefix, chartName string) string {
	return path.Join(iconPrefix, chartName+iconFormat)
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
			chartFilePath = genChartIcon(iconPrefixForLoad, chartName)
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
