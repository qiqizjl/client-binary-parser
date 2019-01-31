package main

import (
	"archive/zip"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/qiqizjl/ipapk"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var VERSION = "UNKNOWN"

type appInfo struct {
	Name        string `json:"name"`
	PackageName string `json:"package_name"`
	Version     string `json:"version"`
	BuildID     string `json:"build_id"`
	Size        int64  `json:"size"`
	Platform    int    `json:"platform"`
	Icon        string `json:"icon"`
	IOS         struct {
		Type        int      `json:"type"`
		AllowDevice []string `json:"allow_device"`
		TeamName    string   `json:"team_name"`
	} `json:"ios"`
	System struct {
		DownloadTime int64  `json:"download_time"`
		ParserTime   int64  `json:"parser_time"`
		ProcessTime  string `json:"process_time"`
		Version      string `json:"version"`
	} `json:"system"`
}

func ParserHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	downloadURL := r.URL.Query().Get("download_url")
	url := r.URL.Query().Get("url")
	if downloadURL == "" {
		downloadURL = url
	}
	log.Printf("downloadURL:%s", downloadURL)
	if downloadURL == "" {
		_, _ = fmt.Fprintln(w, result(900, "下载地址必须存在", nil))
		return
	}
	localFile, err := download(downloadURL)
	if err != nil {
		_, _ = fmt.Fprintln(w, result(903, err.Error(), nil))
		return
	}
	defer func() {
		_ = os.Remove(localFile)
	}()
	downloadTime := time.Now().UnixNano() - startTime.UnixNano()
	parserStartTime := time.Now()
	appInfo, err := parse(localFile)
	if err != nil {
		_, _ = fmt.Fprintln(w, result(901, err.Error(), nil))
		return
	}
	parseTime := time.Now().UnixNano() - parserStartTime.UnixNano()
	respResult := fmtAppInfo(appInfo)
	respResult.System.DownloadTime = downloadTime / 1000000
	respResult.System.ParserTime = parseTime / 1000000
	respResult.System.ProcessTime = time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
	respResult.System.Version = VERSION
	_, _ = fmt.Fprintln(w, result(200, "OK", respResult))

}

func fmtAppInfo(info *ipapk.AppInfo) appInfo {
	result := appInfo{}
	result.Name = info.Name
	result.Size = info.Size
	result.Platform = info.Platform
	result.BuildID = info.Build
	result.PackageName = info.BundleId
	result.Version = info.Version
	iconBuffer := new(bytes.Buffer)
	_ = png.Encode(iconBuffer, info.Icon)
	result.Icon = base64.StdEncoding.EncodeToString(iconBuffer.Bytes())
	result.IOS.AllowDevice = make([]string, 0)
	if result.Platform == ipapk.PlatformIOS {
		result.IOS.TeamName = info.IOS.TeamName
		result.IOS.Type = info.IOS.Type
		result.IOS.AllowDevice = info.IOS.AllowDevice
	}
	return result
}

func main() {
	http.HandleFunc("/parser", raven.RecoveryHandler(ParserHandler))
	http.HandleFunc("/handler", raven.RecoveryHandler(ParserHandler))
	_ = http.ListenAndServe("0.0.0.0:9100", nil)
}

type resultMessage struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func result(code int, message string, result interface{}) string {
	resultStr := resultMessage{
		Code:    code,
		Message: message,
		Result:  result,
	}
	if result == nil {
		resultStr.Result = make(map[string]interface{}, 0)
	}
	res, _ := json.Marshal(resultStr)
	return string(res)
}

func download(downloadURL string) (string, error) {
	res, err := http.Get(downloadURL)
	if err != nil {
		return "", err
	}
	localFile := "/tmp/" + makeMD5(downloadURL)
	f, err := os.Create(localFile)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, res.Body)
	if err != nil {
		return "", err
	}
	return localFile, nil
}

func parse(localFile string) (*ipapk.AppInfo, error) {
	appInfo, err := ipapk.NewAppParser(localFile)
	if err == zip.ErrFormat {
		return nil, errors.New("unknown platform")
	}
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return appInfo, nil
}

func makeMD5(text string) string {
	ctx := md5.New()
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}
