package main

import (
	"flag"
	"io/ioutil"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/spf13/viper"

    "github.com/yuukimiyo/go-twitter-dump-extractor/TextUtil"
)

func init() {
	flag.Set("stderrthreshold", "WARNING")
	flag.Set("v", "5")
	flag.Parse()

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.ReadInConfig()
}

func getDirs(dataRoot string) []string {
	glog.V(3).Infof("datadir: " + dataRoot)

	files, err := ioutil.ReadDir(dataRoot)
	if err != nil {
		glog.Error(err)
	}

	var dataDirs []string
	for _, file := range files {
		if file.IsDir() {
			dataDirs = append(dataDirs, filepath.Join(dataRoot, file.Name()))
		}
	}

	return dataDirs
}

func getFiles(dataRoot string) []string {

    files, err := ioutil.ReadDir(dataRoot)
    if err != nil {
        glog.Error(err)
    }

	var dataFiles []string
	for _, file := range files {
		dataFiles = append(dataFiles, filepath.Join(dataRoot, file.Name()))
	}

	return dataFiles
}

func main() {
	glog.V(3).Infof("start")

	// 設定ファイルからTwitter APIの結果ファイルのディレクトリを取得
	dataRoot := viper.GetString("datadir")

	for _, eachDir := range getDirs(dataRoot) {
		glog.V(3).Infof("datadir: " + eachDir)

		dataFiles := getFiles(eachDir)
		for _, eachFile := range dataFiles {
			glog.V(3).Infof("datafile: " + eachFile)
		}
	}

	TextUtil.Deflate("test")
}
