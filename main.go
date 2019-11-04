package main

import (
	"fmt"
	"flag"
	"os"
	"io"
	"bufio"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	"gopkg.in/xmlpath.v1"

    "github.com/yuukimiyo/go-totext"
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

func extractEachFile(fileName string) []string {
	glog.V(3).Infof("datafile: " + fileName)
	// fmt.Printf(fileName + "\n")

    fp, err := os.Open(fileName)
    if err != nil {
        panic(err)
    }
    defer fp.Close()

	var rd = bufio.NewReaderSize(fp, 1000000)

	for {
		var line, _ = readLine(rd)
		if line == "" {
			break
		}
		var body = strings.Split(line, "\t")
		// glog.V(3).Infof(body[2])

		apiResult, _ := totext.Inflate(body[2])
		//  glog.V(3).Infof(apiResult)
		fmt.Println(apiResult)

		value := getXpathResult(apiResult, 

		break
	}

	return nil
}

func getXpathResult(xml string, xpath string) string {

    // xmlテキストをReader化してパース
    root, err := xmlpath.Parse(strings.NewReader(xml))
    if err != nil {
        glog.Warning(err)
    }

    // xpathをコンパイル
    path := xmlpath.MustCompile(xpath)

    var lines []string
    iter := path.Iter(root)
    for iter.Next() {
        n := iter.Node()
        lines = append(lines, n.String())
    }

    return strings.Join(lines, "\n")
}

func readLine(rd *bufio.Reader) (string, bool) {
	iseof := false
    buf := make([]byte, 0, 1000000)
    for {
        l, p, e := rd.ReadLine()
        if e != nil {
			if e == io.EOF {
				iseof = true
				break
			} else {
				panic(e)
			}
        }
        buf = append(buf, l...)
        if !p {
            break
        }
    }
	return string(buf), iseof
}

func main() {
	glog.V(3).Infof("start")

	// 設定ファイルからTwitter APIの結果ファイルのディレクトリを取得
	dataRoot := viper.GetString("datadir")

	// 結果ファイルのディレクトリのサブディレクトリ毎に処理
	for _, eachDir := range getDirs(dataRoot) {
		// glog.V(3).Infof("datadir: " + eachDir)

		// サブディレクトリ内の結果ファイル一覧を取得し、ファイル毎に処理
		dataFiles := getFiles(eachDir)
		for _, eachFile := range dataFiles {

			// 個別のファイルからデータを取得
			extractEachFile(eachFile)
			break
		}
		break
	}

}
