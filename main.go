package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/viper"
	"github.com/yuukimiyo/go-totext"
)

func init() {
	_ = flag.Set("stderrthreshold", "WARNING")
	_ = flag.Set("v", "5")
	flag.Parse()

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	_ = viper.ReadInConfig()
}

// Statuses is root object of result of twitter search api.
type Statuses struct {
	Statuses []struct {
		Text string `json:"text"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		QuotedStatus struct {
			User struct {
				Name       string `json:"name"`
				ScreenName string `json:"screen_name"`
			} `json:"user"`
		} `json:"quoted_status"`
	} `json:"statuses"`
}

func getDirs(dataRoot string) []string {
	glog.V(3).Infof("datadir: " + dataRoot)

	var dataDirs []string

	files, err := ioutil.ReadDir(dataRoot)
	if err != nil {
		glog.Error(err)
	}

	for _, file := range files {
		if file.IsDir() {
			dataDirs = append(dataDirs, filepath.Join(dataRoot, file.Name()))
		}
	}

	return dataDirs
}

func getFiles(dataRoot string) []string {
	var dataFiles []string

	files, err := ioutil.ReadDir(dataRoot)
	if err != nil {
		glog.Error(err)
	}

	for _, f := range files {
		fullpath := filepath.Join(dataRoot, f.Name())
		// _, err := os.Stat(fullpath)
		// if err != nil {
		// }
		dataFiles = append(dataFiles, fullpath)
	}

	return dataFiles
}

func parseJson(jsonStr string) Statuses {

	var result Statuses
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Fatal(err)
	}

	return result
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

		// zlib+base64で圧縮されたapiの実行結果を伸長
		apiResult, _ := totext.Inflate(body[2])

		// apiの実行結果をパース
		statuses := parseJson(apiResult)

		if len(statuses.Statuses) > 0 {
			for _, eachStatus := range statuses.Statuses {
				var haveAccountName string = ""
				if strings.Contains(eachStatus.Text, "PokemonGoApp") {
					haveAccountName = "have"
				}

				glog.V(3).Infof("----------")
				glog.V(3).Infof(eachStatus.User.Name)
				glog.V(3).Infof(eachStatus.QuotedStatus.User.ScreenName)
				glog.V(3).Infof(haveAccountName)
			}
		}

		// value := getXpathResult(apiResult,

		break
	}

	return nil
}

/*
func getXpathResult(xml string, xpath string) string {
	// xmlテキストをReader化してパース
	root, err := xmlpath.Parse(strings.NewReader(xml))
	if err != nil {
		glog.Warning(err)
	}

	// xpathをコンパイル
	path := xmlpath.MustCompile(xpath)

	iter := path.Iter(root)

	var lines []string

	for iter.Next() {
		n := iter.Node()
		lines = append(lines, n.String())
	}

	return strings.Join(lines, "\n")
}
*/

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
