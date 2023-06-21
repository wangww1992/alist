package _Ecpan

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

func encodeURIComponent(str string) string {
	r := url.QueryEscape(str)
	r = strings.Replace(r, "+", "%20", -1)
	r = strings.Replace(r, "%21", "!", -1)
	r = strings.Replace(r, "%27", "'", -1)
	r = strings.Replace(r, "%28", "(", -1)
	r = strings.Replace(r, "%29", ")", -1)
	r = strings.Replace(r, "%2A", "*", -1)
	return r
}

func getTime(t string) time.Time {
	stamp, _ := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	return stamp
}

func (d *Ecpan) request(pathname string, method string, callback base.ReqCallback, resp interface{}) ([]byte, error) {
	url := "https://www.ecpan.cn/drive" + pathname
	req := base.RestyClient.R()
	if callback != nil {
		callback(req)
	}
	req.SetHeaders(map[string]string{
		"Accept":  "application/json, text/plain, */*",
		"Cookie":  "sid=" + d.SessionId,
		"Origin":  "https://www.ecpan.cn",
		"Referer": "https://www.ecpan.cn/web/yunpan/",
	})

	var e BaseResp
	res, err := req.Execute(method, url)
	log.Debugln(res.String())
	err = utils.Json.Unmarshal(res.Body(), &e)
	if err != nil {
		return nil, err
	}
	if e.Code != "S_OK" {
		return nil, errors.New(e.Summary)
	}
	if resp != nil {
		err = utils.Json.Unmarshal(res.Body(), resp)
		if err != nil {
			return nil, err
		}
	}
	return res.Body(), nil
}

func (d *Ecpan) post(pathname string, data interface{}, resp interface{}) ([]byte, error) {
	return d.request(pathname, http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, resp)
}

func (d *Ecpan) getFiles(catalogID string) ([]model.Obj, error) {
	start := 0
	limit := 100
	files := make([]model.Obj, 0)
	for {
		data := base.Json{
			"IsNotCheckSub": 1,
			"diskType":      1,
			"parentId":      catalogID,
			"orderBy":       "desc",
			"orderField":    "modify_date",
			"reqPage":       start + 1,
			"pageSize":      limit,
			"rootUsn":       d.Account,
			"status":        1,
			"picHeight":     72,
			"picThumbnail":  1,
			"picWidth":      72,
			"usn":           d.Account,
		}
		var resp GetDiskResp
		_, err := d.post("/service/disk/search.do?func=disk:search", data, &resp)
		if err != nil {
			return nil, err
		}
		for _, catalog := range resp.Data.ResultList {
			f := model.Object{
				ID:       catalog.AppFileId,
				Name:     catalog.FileName,
				Size:     catalog.FileSize,
				Modified: getTime(catalog.ModifyDate),
				IsFolder: catalog.FileType == 1,
			}
			files = append(files, &f)
		}
		for _, content := range resp.Data.ResultList {
			f := model.ObjThumb{
				Object: model.Object{
					ID:       content.AppFileId,
					Name:     content.FileName,
					Size:     content.FileSize,
					Modified: getTime(content.ModifyDate),
				},
				//Thumbnail: model.Thumbnail{Thumbnail: content.ThumbnailURL},
				//Thumbnail: content.BigthumbnailURL,
			}
			files = append(files, &f)
		}
		if start*limit >= int(resp.Data.RecordCount) {
			break
		}
		start += 1
	}
	return files, nil
}

func (d *Ecpan) getLink(contentId string) (string, error) {
	data := base.Json{
		"diskType":   1,
		"downSource": 1,
		"fileIdList": []base.Json{{
			"appFileId": contentId,
		}},
		"isDownload": 1,
	}
	res, err := d.post("/service/common/file.do?func=common:download",
		data, nil)
	if err != nil {
		return "", err
	}
	return jsoniter.Get(res, "var", "downloadUrl").ToString(), nil
}

func (d *Ecpan) doMoveOrCopy(srcObj, dstDir model.Obj, copyFlag bool) error {
	isCopy := 0
	if copyFlag {
		isCopy = 1
	}
	data := base.Json{
		"comeFrom": 10,
		"diskType": 1,
		"fileIdList": []base.Json{
			{
				"appFileId": srcObj.GetID(),
			},
		},
		"isCopy":     isCopy,
		"toFolderId": dstDir.GetID(),
	}

	pathname := "/service/common/file.do?func=common:newMvOrCp"
	var resp SimpleResp
	_, err := d.post(pathname, data, &resp)
	if err != nil {
		return err
	}
	data = base.Json{
		"taskId": resp.Var,
	}
	pathname = "/service/common/file.do?func=common:OpFileProgress"
	_, err = d.post(pathname, data, nil)
	return err
}

func unicode(str string) string {
	textQuoted := strconv.QuoteToASCII(str)
	textUnquoted := textQuoted[1 : len(textQuoted)-1]
	return textUnquoted
}

const (
	_  = iota //ignore first value by assigning to blank identifier
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

func getPartInfo(size int64, num int64) (int64, int64, int64, int64, bool) {
	var partSize int64
	partSize = 5 * MB
	if size/GB > 50 {
		partSize = 10 * MB
	}
	start := num * partSize
	remainSize := size - start
	if remainSize <= partSize {
		return partSize, remainSize, start, start + remainSize - 1, false
	}
	return partSize, remainSize, start, start + partSize - 1, true
}

func GetFileMd5(f *os.File) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	md5Str := hex.EncodeToString(hash.Sum(nil))
	return md5Str, nil
}

func GetBytesMd5(bytes []byte) string {
	hash := md5.New()
	hash.Write(bytes)
	md5Str := hex.EncodeToString(hash.Sum(nil))
	return md5Str
}

// CreateTempFile create temp file from io.ReadCloser, and seek to 0
func CreateTempFileFromBytes(bytes []byte) (*os.File, error) {
	f, err := os.CreateTemp(conf.Conf.TempDir, "file-*")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(f.Name(), bytes, 0644)
	if err != nil {
		_ = os.Remove(f.Name())
		return nil, err
	}
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		_ = os.Remove(f.Name())
		return nil, err
	}
	return f, nil
}
