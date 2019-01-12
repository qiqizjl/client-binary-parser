package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/qiqizjl/ipapk"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
)

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
}

func ParserHandler(w http.ResponseWriter, r *http.Request) {
	downloadURL := r.URL.Query().Get("download_url")
	if downloadURL == "" {
		_, _ = fmt.Fprintln(w, result(900, "下载地址必须存在", nil))
		return
	}
	appInfo, err := parse(downloadURL)
	if err != nil {
		_, _ = fmt.Fprintln(w, result(901, err.Error(), nil))
		return
	}
	_, _ = fmt.Fprintln(w, result(200, "OK", fmtAppInfo(appInfo)))

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
	result.IOS.AllowDevice = make([]string,0);
	if result.Platform == ipapk.PlatformIOS {
		result.IOS.TeamName = info.IOS.TeamName
		result.IOS.Type = info.IOS.Type
		result.IOS.AllowDevice = info.IOS.AllowDevice
	}
	return result
}

func main() {
	http.HandleFunc("/parser", ParserHandler)
	_ = http.ListenAndServe("0.0.0.0:8080", nil)
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

func parse(downloadURL string) (*ipapk.AppInfo, error) {
	res, err := http.Get(downloadURL)
	if err != nil {
		return nil, err
	}
	localFile := "/tmp/" + makeMD5(downloadURL) + path.Ext(downloadURL)
	f, err := os.Create(localFile)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, res.Body)
	if err != nil {
		return nil, err
	}
	appInfo, err := ipapk.NewAppParser(localFile)
	if err != nil {
		return nil, err
	}
	err = os.Remove(localFile)
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
