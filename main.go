package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	elastic "github.com/olivere/elastic/v7"
	mecab "github.com/shogo82148/go-mecab"
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
		CreatedAt string `json:"created_at"`
		Text      string `json:"text"`
		User      struct {
			Name       string `json:"name"`
			ScreenName string `json:"screen_name"`
		} `json:"user"`
		QuotedStatus struct {
			User struct {
				Name       string `json:"name"`
				ScreenName string `json:"screen_name"`
			} `json:"user"`
		} `json:"quoted_status"`
	} `json:"statuses"`
}

func parseJSON(jsonStr string) Statuses {
	var result Statuses
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Fatal(err)
	}

	return result
}

func cleanText(text string, repl *strings.Replacer, ptns []*regexp.Regexp) string {
	for _, ptn := range ptns {
		text = ptn.ReplaceAllString(text, "")
	}

	text = repl.Replace(text)

	return text
}

// WriteLines is function to write string array.
func WriteLines(filename string, lines []string, linesep string, writeBom bool, modeflag string, permission os.FileMode) error {
	mode := os.O_WRONLY | os.O_CREATE
	if modeflag == "a" {
		mode = os.O_WRONLY | os.O_APPEND
	} else {
	}

	f, err := os.OpenFile(filename, mode, permission)
	if err != nil {
		return err
	}
	defer f.Close()

	if writeBom {
		f.Write([]byte{239, 187, 191})
	}

	for _, line := range lines {
		f.WriteString(line + linesep)
	}

	return nil
}

func extractEachFile(fileName string, mecabModel *mecab.Model) []string {
	repl := strings.NewReplacer(
		"\r\n", "",
		"\r", "",
		"\n", "",
		"\t", "",
		" ", "",
		"　", "",
		",", " ",
	)

	var ptns []*regexp.Regexp
	ptns = append(ptns, regexp.MustCompile(`@[^\s]+`))
	ptns = append(ptns, regexp.MustCompile(`#[^\s]+`))
	ptns = append(ptns, regexp.MustCompile(`RT\s*[:：]`))
	ptns = append(ptns, regexp.MustCompile(`RT`))
	ptns = append(ptns, regexp.MustCompile(`(http|https)://([\w-]+\.)+[\w-]+(/[\w-./?%&=]*)?`))

	// MeCabのtaggerを取得
	tagger, err := mecabModel.NewMeCab()
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	// ファイルの読み込みを開始
	fp, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	var rd = bufio.NewReaderSize(fp, 1024*1024)

	// ファイル名から、データへ書き込む4桁のQueryIDを取得
	queryID := strings.Split(fileName, "_")[2]

	// 結果格納用の配列
	lines := []string{}

	for {
		// 読み込みバッファを元に、1行ずつ取得
		line, err := totext.ReadLine(rd, make([]byte, 0, 1024*1024))
		if err != nil {
			if err == io.EOF {
				break
			}

			glog.Errorf("%s", err)
		}

		body := strings.Split(line, "\t")

		// zlib+base64で圧縮されたapiの実行結果データを伸張
		apiResult, err := totext.Inflate(body[2])
		if err != nil {
			glog.V(3).Infof("%s", err)
			continue
		}

		// 伸張したapiの実行結果をパース
		statuses := parseJSON(apiResult)

		// APIの結果は元の、検索結果のツイートデータが配列として格納されている
		// 各ツイートデータ毎にデータを取得して配列に格納
		if len(statuses.Statuses) > 0 {
			for _, eachStatus := range statuses.Statuses {
				// ツイート日時を取得（フォーマット不正のツイートは無視する）
				createdAtUtc, err := time.Parse(time.UnixDate, eachStatus.CreatedAt)
				if err != nil {
					continue
				}

				// UTCで格納されているツイート日時をJSTに変換
				createdAtJst := createdAtUtc.In(time.FixedZone("Asia/Tokyo", 9*60*60))

				// ツイート本文を取得
				text := eachStatus.Text

				// テキストをクレンジング
				text = cleanText(text, repl, ptns)

				// 分かち書きを実行
				text, err = tagger.Parse(text)
				if err != nil {
					continue
				}

				text = strings.TrimRight(text, "\n")

				lines = append(lines, fmt.Sprintf("%s\t%s\t%s\t%s\t%s", createdAtJst.Format("2006-01-02 15:04:05"), queryID, eachStatus.User.Name, eachStatus.User.ScreenName, text))
			}
		}
	}

	return lines
}

func bulkInsert(lines []string, cli *elastic.Client, c *context.Context) error {
	bulkRequest := cli.Bulk()
	// var req *elastic.BulkIndexRequest

	for _, line := range lines {
		d := strings.Split(line, "\t")

		data := map[string]string{
			"createdAt":      d[0],
			"queryID":        d[1],
			"userName":       d[2],
			"userScreenName": d[3],
			"text":           d[4],
		}
		// req = elastic.NewBulkIndexRequest().Index("tweets").Doc(data)
		bulkRequest = bulkRequest.Add(elastic.NewBulkIndexRequest().Index("tweets").Doc(data))
	}

	bulkResponse, err := bulkRequest.Do(*c)
	if err != nil {
		return err
	}

	for _, eachResponse := range bulkResponse.Created() {
		// glog.V(3).Infof("Created Result: %s", eachResponse.Result)
		if eachResponse.Status != 201 {
			glog.V(3).Infof("Created status: %d", eachResponse.Status)
			panic(eachResponse.Result)
		}
	}

	for _, eachResponse := range bulkResponse.Indexed() {
		// glog.V(3).Infof("Indexed Result: %s", eachResponse.Result)
		if eachResponse.Status != 201 {
			glog.V(3).Infof("Indexed status: %d", eachResponse.Status)
			panic(eachResponse.Result)
		}
	}

	return nil
}

func fileWalker(i int, filePath string, outdir string, mecabModel *mecab.Model, wg *sync.WaitGroup, ch *chan int) {
	glog.V(3).Infof("[%d] extract: %s", i, filePath)

	texts := extractEachFile(filePath, mecabModel)

	/*
		// 出力先のファイル名を作成
		filebase := filepath.Base(strings.Replace(filepath.Base(filePath), ".tsv", "", -1))
		outfile := outdir + "/" + filebase + "_out.tsv"

		err := WriteLines(outfile, texts, "\n", true, "w", 0644)
		if err != nil {
			glog.Errorf("%s", err)
		}
	*/

	c := context.Background()

	cli, err := elastic.NewClient(
		elastic.SetURL("http://localhost:9200"),
		elastic.SetSniff(false),
	)

	if err != nil {
		glog.Errorf("%s", err)
	}

	defer cli.Stop()

	err = bulkInsert(texts, cli, &c)
	if err != nil {
		glog.Errorf("%s", err)
	}

	// チャンネルから値を取り出す(一つ空ける)
	// 空くので、次のスレッドが開始可能になる
	<-*ch
	wg.Done()
}

func main() {
	glog.V(3).Infof("start")

	// 設定ファイルからTwitter APIの結果ファイルのディレクトリを取得
	dataRoot := viper.GetString("datadir")

	var outdir string = dataRoot + "extract"

	// 出力先ディレクトリを、なければ作成
	err := totext.MakeDir(outdir)
	if err != nil {
		glog.Errorf("%s", err)
	}

	mecabModel, err := mecab.NewModel(map[string]string{"output-format-type": "wakati"})
	if err != nil {
		panic(err)
	}
	defer mecabModel.Destroy()

	// マルチスレッドのチャンネルを初期化、引数は最大スレッド数
	ch := make(chan int, 4)

	// マルチスレッドの処理待ちのため
	wg := &sync.WaitGroup{}

	startTime := time.Now()

	// 入力ディレクトリ内の個別フォルダ/個別ファイルに対して処理を実施
	// 4007=白猫
	files, _ := filepath.Glob(dataRoot + "4007/*.tsv")
	for i, filePath := range files {
		ch <- 1

		wg.Add(1)

		go fileWalker(i, filePath, outdir, &mecabModel, wg, &ch)

		/*
			if i >= 50 {
				break
			}
		*/
	}

	wg.Wait()
	glog.V(3).Infof("%s", time.Since(startTime))
	glog.V(3).Infof("end")
}
